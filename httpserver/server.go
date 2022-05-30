package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

// Server provides an HTTP server (indeed http.Server is embedded)
// that can be initialized and hosted within a CLI app.  The server is used
// within the Uses pipeline where it registers itself as a context service.
// The action RunServer is used to actually run the server.
type Server struct {
	*http.Server

	ready func(context.Context)
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

// WithHandler sets the handler which will run on the server
func WithHandler(handler http.Handler) Option {
	return func(s *Server) {
		s.Server.Handler = handler
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

func (s *Server) actualReady() func(context.Context) {
	if s.ready == nil {
		return func(_ context.Context) {}
	}
	return s.ready
}

func (s *Server) proto() string {
	return "http://"
}
