// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpserver can host an HTTP server in the CLI app.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

	TLSCertFile string
	TLSKeyFile  string

	// ShutdownTimeout specifies how long to wait for the server to shutdown
	// when a signal is received
	ShutdownTimeout time.Duration

	staticDir       string
	handlerFactory  func(*Server) (http.Handler, error)
	ready           ReadyFunc
	shutdown        ReadyFunc
	hideDirListings bool
	middleware      []MiddlewareFunc
	accessLog       string
	actualBind      struct {
		addr string
		tls  bool
	}
}


// Option is an option to configure the server
// Option can be used as an Action, typically within the Uses or Before pipeline.
type Option interface {
	cli.Action
	apply(*Server)
}

type option[T any] struct {
	val T
	fn  func(*Server, T) error
}

func (o option[_]) Execute(ctx context.Context) error {
	o.apply(FromContext(ctx))
	return nil
}

func (o option[_]) apply(s *Server) {
	o.fn(s, o.val)
}

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

	defaultShutdownTimeout = 3 * time.Second
)

var (
	// ErrNotListening is reported when the server is not listening
	ErrNotListening = errors.New("server is not listening")
)

// New creates a new HTTP server with the given handler creation callback.
func New(options ...Option) *Server {
	s := &Server{
		Server: &http.Server{},
	}
	s.Apply(defaultOptions(s)...)
	s.Apply(options...)
	return s
}

func defaultOptions(s *Server) []Option {
	return []Option{
		WithAddr("localhost:8000"),
		WithShutdownTimeout(defaultShutdownTimeout),
		WithAccessLog(defaultAccessLog),
		WithMiddleware(func(h http.Handler) http.Handler {
			if s.accessLog != "" {
				return NewRequestLogger(s.accessLog, os.Stderr, h)
			}
			return h
		}),
	}
}

func (s *Server) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(s)
	}
}

func DefaultServer() *Server {
	return New(
		WithReadyFunc(DefaultReadyFunc),
		WithHandlerFactory(func(s *Server) (http.Handler, error) {
			return newFileServerHandler(s.staticDir, s.HideDirectoryListing()), nil
		}),
	)
}

// WithHandler sets the handler which will run on the server
func WithHandler(handler http.Handler) Option {
	return withAdapter((*Server).WithHandler, handler)
}

// WithHandlerFactory sets how to create the handler which will run on the server
func WithHandlerFactory(f func(*Server) (http.Handler, error)) Option {
	return withAdapter((*Server).WithHandlerFactory, f)
}

// WithReadyFunc sets a callback for when the server is listening
func WithReadyFunc(ready ReadyFunc) Option {
	return withAdapter((*Server).WithReadyFunc, ready)
}

// AddReadyFunc appends a callback for when the server is ready
func AddReadyFunc(ready ReadyFunc) Option {
	return withAdapter((*Server).AddReadyFunc, ready)
}

// WithShutdownFunc sets a callback for when the server is shutting down
func WithShutdownFunc(shutdown ReadyFunc) Option {
	return withAdapter((*Server).WithShutdownFunc, shutdown)
}

// AddShutdownFunc adds a callback for when the server is shutting down
func AddShutdownFunc(shutdown ReadyFunc) Option {
	return withAdapter((*Server).AddShutdownFunc, shutdown)
}

// WithMiddleware adds handler middleware
func WithMiddleware(m MiddlewareFunc) Option {
	return withAdapter((*Server).AddMiddleware, m)
}

// WithAddr sets the corresponding server field
func WithAddr(addr string) Option {
	return withAdapter((*Server).SetAddr, addr)
}

// WithHostname sets the server hostname
func WithHostname(v string) Option {
	return withAdapter((*Server).SetHostname, v)
}

// WithPort sets the server port
func WithPort(v int) Option {
	return withAdapter((*Server).SetPort, v)
}

// WithShutdownTimeout sets the amount of time to allow the server to shutdown
func WithShutdownTimeout(d time.Duration) Option {
	return withAdapter((*Server).SetShutdownTimeout, d)
}

// WithReadTimeout sets the amount of time to allow for reading requests
func WithReadTimeout(d time.Duration) Option {
	return withAdapter((*Server).SetReadTimeout, d)
}

// WithReadHeaderTimeout sets the amount of time to allow for reading headers
func WithReadHeaderTimeout(d time.Duration) Option {
	return withAdapter((*Server).SetReadHeaderTimeout, d)
}

// WithWriteTimeout sets the amount of time to allow for writing responses
func WithWriteTimeout(d time.Duration) Option {
	return withAdapter((*Server).SetWriteTimeout, d)
}

// WithIdleTimeout sets the amount of time to allow for idling
func WithIdleTimeout(d time.Duration) Option {
	return withAdapter((*Server).SetIdleTimeout, d)
}

// WithMaxHeaderBytes sets the amount max header bytes
func WithMaxHeaderBytes(n int) Option {
	return withAdapter((*Server).SetMaxHeaderBytes, n)
}

// WithTLSKeyFile sets the file to use for the TLS key
func WithTLSKeyFile(filename string) Option {
	return withAdapter((*Server).SetTLSKeyFile, filename)
}

// WithTLSCertFile sets the file to use for the TLS cert
func WithTLSCertFile(filename string) Option {
	return withAdapter((*Server).SetTLSCertFile, filename)
}

// WithServerHeader sets the contents of the server header
func WithServerHeader(s string) Option {
	return withAdapter((*Server).SetServerHeader, s)
}

// WithAccessLog sets the format string for the access log
func WithAccessLog(s string) Option {
	return withAdapter((*Server).SetAccessLog, s)
}

// WithNoAccessLog disables the access log
func WithNoAccessLog() Option {
	return withAdapter((*Server).SetNoAccessLog, true)
}

// WithStaticDirectory hosts a static directory
func WithStaticDirectory(path string) Option {
	return withAdapter((*Server).SetStaticDirectory, path)
}

// WithNoDirectoryListings disables directory listings
func WithNoDirectoryListings() Option {
	return withAdapter((*Server).SetNoDirectoryListings, true)
}

// FromContext obtains the server from the context.
func FromContext(ctx context.Context) *Server {
	return ctx.Value(servicesKey).(*Server)
}

// AddMiddleware appends additional middleware to the server
func (s *Server) AddMiddleware(m MiddlewareFunc) error {
	s.middleware = append(s.middleware, m)
	return nil
}

// AddShutdownFunc appends additional shutdown functions
func (s *Server) AddShutdownFunc(shutdown ReadyFunc) error {
	s.shutdown = ComposeReadyFuncs(s.shutdown, shutdown)
	return nil
}

// WithShutdownFunc sets the shutdown function
func (s *Server) WithShutdownFunc(shutdown ReadyFunc) error {
	s.shutdown = shutdown
	return nil
}

// WithHandler sets the handler for the server
func (s *Server) WithHandler(handler http.Handler) error {
	s.Server.Handler = handler
	return nil
}

// WithHandlerFactory sets the handler factory for the server
func (s *Server) WithHandlerFactory(f func(*Server) (http.Handler, error)) error {
	s.handlerFactory = f
	return nil
}

// WithReadyFunc sets the ready function for the server
func (s *Server) WithReadyFunc(ready ReadyFunc) error {
	s.ready = ready
	return nil
}

// AddReadyFunc appens additional ready functions
func (s *Server) AddReadyFunc(ready ReadyFunc) error {
	s.ready = ComposeReadyFuncs(s.ready, ready)
	return nil
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

	s.actualBind.addr = listener.Addr().String()
	s.actualBind.tls = (s.TLSCertFile != "")
	s.applyMiddleware()

	if s.TLSCertFile == "" {
		return s.Server.Serve(listener)
	}

	return s.Server.ServeTLS(listener, s.TLSCertFile, s.TLSKeyFile)
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
	if err := s.checkIfListening(); err != nil {
		return fmt.Errorf("can't open in browser: %w", err)
	}

	bind := s.url().JoinPath(path...)
	fmt.Fprintf(os.Stderr, "Opening default web browser %s...\n", bind)
	return exec.Open(bind.String())
}

func (s *Server) Execute(c context.Context) error {
	return cli.Do(
		c,
		cli.Pipeline(
			FlagsAndArgs(),
			ContextValue(s),
		),
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

func (s *Server) ReportListening() error {
	if err := s.checkIfListening(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Listening on %s... (Press ^C to exit)\n", s.url().String())
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

func (s *Server) SetShutdownTimeout(d time.Duration) error {
	s.ShutdownTimeout = d
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
	s.hideDirListings = v
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

func (s *Server) SetServerHeader(name string) error {
	s.AddMiddleware(NewHeaderMiddleware("Server", name))
	return nil
}

func (s *Server) SetTLSCertFile(v string) error {
	s.TLSCertFile = v
	return nil
}

func (s *Server) SetTLSKeyFile(v string) error {
	s.TLSKeyFile = v
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

func (s *Server) actualReady() ReadyFunc {
	if s.ready == nil {
		return func(_ context.Context) {}
	}
	return s.ready
}

func (s *Server) actualShutdown() ReadyFunc {
	if s.shutdown == nil {
		return func(_ context.Context) {}
	}
	return s.shutdown
}

func (s *Server) checkIfListening() error {
	if s.actualBind.addr == "" {
		return ErrNotListening
	}
	return nil
}

func (s *Server) url() *url.URL {
	if s.actualBind.addr == "" {
		return nil
	}
	proto := "http://"
	if s.actualBind.tls {
		proto = "https://"
	}
	res, _ := url.Parse(proto + s.actualBind.addr)
	return res
}

func withAdapter[T any](fn func(*Server, T) error, value T) Option {
	return option[T]{value, fn}
}
