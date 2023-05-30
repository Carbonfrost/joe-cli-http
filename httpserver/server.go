// Package httpserver can host an HTTP server in the CLI app.
package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Server provides an HTTP server (indeed http.Server is embedded)
// that can be initialized and hosted within a CLI app.  The server is used
// within the Uses pipeline where it registers itself as a context service.
// The action RunServer is used to actually run the server.
//
// The simplest action to use is RunServer(), which runs the server:
//
//	&cli.App{
//	   Name: "goserv",
//	   Uses: &httpserver.New(httpserver.WithHandler(...)),
//	   Action: httpserver.RunServer(),
//	}
//
// This simple app has numerous flags to configure connection handling
// and to dynamically construct the server's router.  Many dynamic server
// routing actions depend upon the server having a handler which is also a
// mux that contains a method Handle(string, http.Handler) to register
// additional handlers.  (This is the same API provided by the built-in Go
// mux, http.ServeMux).  There are other APIs provided by convention, documented
// in their respective contexts.
//
// The cmd/weave package provides weave, which is a command line utility
// that hosts a server for files and some built-in handlers, which is
// similar to what the DefaultServer() does.
//
// If you only want to add the Server to the context (typically in
// advanced scenarios where you are deeply customizing the behavior),
// you only use the action httpserver.ContextValue() with the server
// you want to add instead of add the server to the pipeline directly.
type Server struct {
	*http.Server

	staticDir       string
	handlerFactory  func(*Server) (http.Handler, error)
	ready           func(context.Context)
	hideDirListings bool
	middleware      []MiddlewareFunc
	accessLog       string
	actualBindAddr  string
}

// Option is an option to configure the server
type Option func(*Server)

// MiddlewareFunc defines a function that creates a middleware wrapper around another
// handler
type MiddlewareFunc func(next http.Handler) http.Handler

type mux interface {
	Handle(string, http.Handler)
}

type contextKey string

const servicesKey contextKey = "httpserver_services"

const (
	expectedOneArg = "expected 0 or 1 arg"
)

// New creates a new HTTP server with the given handler creation callback.
func New(options ...Option) *Server {
	s := &Server{
		Server: &http.Server{
			Addr: "localhost:8000",
		},
		accessLog: defaultAccessLog,
	}
	s.AddMiddleware(func(h http.Handler) http.Handler {
		if s.accessLog != "" {
			return NewRequestLogger(s.accessLog, os.Stderr, h)
		}
		return h
	})

	for _, o := range options {
		o(s)
	}
	return s
}

func DefaultServer() *Server {
	return New(WithHandlerFactory(func(s *Server) (http.Handler, error) {
		return newFileServerHandler(s.staticDir, s.HideDirectoryListing()), nil
	}))
}

// WithHandler sets the handler which will run on the server
func WithHandler(handler http.Handler) Option {
	return func(s *Server) {
		s.Server.Handler = handler
	}
}

// WithHandlerFactory sets how to create the handler which will run on the server
func WithHandlerFactory(f func(*Server) (http.Handler, error)) Option {
	return func(s *Server) {
		s.handlerFactory = f
	}
}

// WithReadyFunc sets a callback for when the server is listening
func WithReadyFunc(ready func(context.Context)) Option {
	return func(s *Server) {
		s.ready = ready
	}
}

// WithMiddleware adds handler middleware
func WithMiddleware(m MiddlewareFunc) Option {
	return func(s *Server) {
		s.AddMiddleware(m)
	}
}

// OpenInBrowser is a function to open the server in the browser.  This
// function is passed as a value to WithReadyFunc
func OpenInBrowser(c context.Context) {
	FromContext(c).OpenInBrowser()
}

// FromContext obtains the server from the context.
func FromContext(ctx context.Context) *Server {
	return ctx.Value(servicesKey).(*Server)
}

// AddMiddleware appends additional middleware to the server
func (s *Server) AddMiddleware(m MiddlewareFunc) {
	s.middleware = append(s.middleware, m)
}

func (s *Server) HideDirectoryListing() bool {
	return s.hideDirListings
}

func (s *Server) ListenAndServe() error {
	if s.Server.Handler == nil && s.handlerFactory != nil {
		h, err := s.handlerFactory(s)
		if err != nil {
			return err
		}
		s.Server.Handler = h
	}

	listener, err := net.Listen("tcp", s.Server.Addr)
	if err != nil {
		return err
	}

	s.actualBindAddr = listener.Addr().String()
	s.applyMiddleware()

	fmt.Fprintf(os.Stderr, "Listening on %s%s... (Press ^C to exit)", s.proto(), s.actualBindAddr)
	return s.Server.Serve(listener)
}

func (s *Server) ensureMux() (mux, error) {
	if m, ok := s.Server.Handler.(mux); ok {
		return m, nil
	}
	if s.Server.Handler == nil {
		m := http.NewServeMux()
		s.Server.Handler = m
		return m, nil
	}
	return nil, fmt.Errorf("server handler does not support mux")
}

func (s *Server) applyMiddleware() {
	h := s.Server.Handler
	for _, m := range s.middleware {
		h = m(h)
	}
	s.Server.Handler = h
}

// OpenInBrowser opens in the browser.  The request path can also be
// specified
func (s *Server) OpenInBrowser(path ...string) error {
	if s.actualBindAddr == "" {
		return fmt.Errorf("can't open in browser: server is not listening")
	}
	bind := s.proto() + s.actualBindAddr + strings.Join(path, "")
	fmt.Fprintf(os.Stderr, "Opening default web browser %s...", bind)
	return exec.Open(bind)
}

func (s *Server) Execute(c context.Context) error {
	return cli.Do(
		c,
		FlagsAndArgs(),
		ContextValue(s),
	)
}

func (s *Server) updateAddr(hostname string, port string) error {
	h, p, err := net.SplitHostPort(s.Server.Addr)
	if err != nil {
		s.Server.Addr = net.JoinHostPort(hostname, port)
		return nil
	}
	if hostname == "" {
		hostname = h
	}
	if port == "" {
		port = p
	}
	s.Server.Addr = net.JoinHostPort(hostname, port)
	return nil
}

func (s *Server) SetHostname(name string) error {
	return s.updateAddr(name, "")
}

func (s *Server) SetPort(port int) error {
	return s.updateAddr("", strconv.Itoa(port))
}

func (s *Server) SetAddr(addr string) error {
	s.Server.Addr = addr
	return nil
}

func (s *Server) SetReadTimeout(v time.Duration) error {
	s.Server.ReadTimeout = v
	return nil
}

func (s *Server) SetWriteTimeout(v time.Duration) error {
	s.Server.WriteTimeout = v
	return nil
}

func (s *Server) SetReadHeaderTimeout(v time.Duration) error {
	s.Server.ReadHeaderTimeout = v
	return nil
}

func (s *Server) SetIdleTimeout(v time.Duration) error {
	s.Server.IdleTimeout = v
	return nil
}

func (s *Server) SetMaxHeaderBytes(v int) error {
	s.Server.MaxHeaderBytes = v
	return nil
}

func (s *Server) SetStaticDirectory(path string) error {
	s.staticDir = path
	return nil
}

func (s *Server) SetNoDirectoryListings(v bool) error {
	s.hideDirListings = true
	return nil
}

func (s *Server) SetAccessLog(v string) error {
	s.accessLog = v
	return nil
}

func (s *Server) SetNoAccessLog(v bool) error {
	if v {
		s.accessLog = ""
	} else {
		s.accessLog = defaultAccessLog
	}
	return nil
}

func (s *Server) SetServer(name string) error {
	s.AddMiddleware(NewHeaderMiddleware("Server", name))
	return nil
}

func (s *Server) setStaticDirectoryHelper(f *cli.File) error {
	return s.SetStaticDirectory(f.Name)
}

func (s *Server) setOpenInBrowserHelper(v bool) error {
	WithReadyFunc(OpenInBrowser)(s)
	return nil
}

func (s *Server) Handle(path string, h http.Handler) error {
	m, err := s.ensureMux()
	if err != nil {
		return err
	}
	m.Handle(path, h)
	return nil
}

func (s *Server) actualReady() func(context.Context) {
	if s.ready == nil {
		return func(_ context.Context) {}
	}
	return s.ready
}

func (s *Server) proto() string {
	return "http://"
}

func (o Option) Execute(c context.Context) error {
	o(FromContext(c))
	return nil
}
