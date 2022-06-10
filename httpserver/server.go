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
)

// Server provides an HTTP server (indeed http.Server is embedded)
// that can be initialized and hosted within a CLI app.  The server is used
// within the Uses pipeline where it registers itself as a context service.
// The action RunServer is used to actually run the server.
type Server struct {
	*http.Server

	staticDir       string
	handlerFactory  func(*Server) (http.Handler, error)
	ready           func(context.Context)
	hideDirListings bool
}

// Option is an option to configure the server
type Option func(*Server)

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
	}
	for _, o := range options {
		o(s)
	}
	return s
}

func DefaultServer() *Server {
	return New(WithHandlerFactory(func(s *Server) (http.Handler, error) {
		staticDir := s.staticDir
		if staticDir == "" {
			return nil, nil
		}
		handler := http.FileServer(http.Dir(staticDir))

		if s.hideDirListings {
			handler = hideListing(handler)
		}
		return handler, nil
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

// FromContext obtains the server from the context.
func FromContext(ctx context.Context) *Server {
	return ctx.Value(servicesKey).(*Server)
}

func (s *Server) ListenAndServe() error {
	if s.handlerFactory != nil {
		h, err := s.handlerFactory(s)
		if err != nil {
			return err
		}
		s.Server.Handler = h
	}
	fmt.Fprintf(os.Stderr, "Listening on %s%s... (Press ^C to exit)", s.proto(), s.Server.Addr)
	return s.Server.ListenAndServe()
}

func (s *Server) Execute(c *cli.Context) error {
	return c.Do(
		FlagsAndArgs(),
		cli.ContextValue(servicesKey, s),
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

func (s *Server) SetStaticDirectory(path string) error {
	s.staticDir = path
	return nil
}

func (s *Server) SetNoDirectoryListings(v bool) error {
	s.hideDirListings = true
	return nil
}

func (s *Server) setStaticDirectoryHelper(f *cli.File) error {
	return s.SetStaticDirectory(f.Name)
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

func hideListing(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, req)
	}
}
