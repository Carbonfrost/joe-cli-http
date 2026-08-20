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
	"github.com/Carbonfrost/joe-cli-http/internal/pattern"
	"github.com/Carbonfrost/joe-cli/extensions/exec"
)

//go:generate go tool counterfeiter -generate

// Server provides an HTTP server (which encapsulates an http.Server)
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
// The server is configured exclusively with Options, either passed to New or
// applied later using Apply.  The underlying http.Server is created on demand
// the first time it is needed, which is available from NewServer.  How it gets
// created can be customized with WithServer or WithServerFactory.
//
// The cmd/weave package provides weave, which is a command line utility
// that hosts a server for files and some built-in handlers, which is
// similar to what the default server  does.
//
// If you only want to add the Server to the context (typically in
// advanced scenarios where you are deeply customizing the behavior),
// you only use the action httpserver.ContextValue() with the server
// you want to add instead of add the server to the pipeline directly.
type Server struct {
	cli.Action

	server  cacheable[*http.Server]
	handler cacheable[http.Handler]

	addr              string
	readTimeout       time.Duration
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	maxHeaderBytes    int

	tlsCertFile     string
	tlsKeyFile      string
	shutdownTimeout time.Duration
	staticDir       string
	ready           ReadyFunc
	shutdown        ReadyFunc
	hideDirListings bool
	accessLog       string

	// handlerErr records the error, if any, from setting up the handler,
	// which is reported from NewServer
	handlerErr error

	actualBind struct {
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

type optionFunc func(*Server) error

func (f optionFunc) Execute(ctx context.Context) error {
	return f(FromContext(ctx))
}

func (f optionFunc) apply(s *Server) {
	f(s)
}

// MiddlewareFunc defines a function that creates a middleware wrapper around another
// handler
type MiddlewareFunc func(next http.Handler) http.Handler

type cacheable[T comparable] = pattern.Cacheable[T]

type mux interface {
	Handle(string, http.Handler)
}

type contextKey string

const servicesKey contextKey = "httpserver_services"

const (
	defaultShutdownTimeout = 3 * time.Second
)

var (
	// ErrNotListening is reported when the server is not listening
	ErrNotListening = errors.New("server is not listening")
)

// New creates a new HTTP server with the given handler creation callback.
func New(options ...Option) *Server {
	s := &Server{}
	s.Apply(defaultOptions(s)...)
	s.Apply(options...)
	return s
}

func defaultOptions(s *Server) []Option {
	return []Option{
		WithDefaultAction(),
		WithDefaultServerFactory(),
		WithAddr("localhost:8000"),
		WithShutdownTimeout(defaultShutdownTimeout),
		WithAccessLog(defaultAccessLog),
		WithReadyFunc(DefaultReadyFunc),
		WithShutdownFunc(DefaultShutdownFunc),
		WithMiddleware(func(h http.Handler) http.Handler {
			if s.accessLog != "" {
				return NewRequestLoggerMiddleware(s.accessLog, os.Stderr)(h)
			}
			return h
		}),
	}
}

func (s *Server) Pipeline() cli.Action {
	return s.Action
}

func (s *Server) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(s)
	}
}

func NewDefault() *Server {
	return New(
		WithHandlerFactory(func(s *Server) (http.Handler, error) {
			return newFileServerHandler(s.staticDir, s.HideDirectoryListing()), nil
		}),
	)
}

// WithAction sets the action
func WithAction(a cli.Action) Option {
	return withAdapter[cli.Action]((*Server).setAction, a)
}

// WithDefaultAction sets the action to the default
func WithDefaultAction() Option {
	return option[cli.Action]{
		nil, func(s *Server, _ cli.Action) error {
			s.Action = cli.Pipeline(
				ContextValue(s),
				FlagsAndArgs(),
			)
			return nil
		},
	}
}

// WithServer sets the http.Server to use directly, bypassing the default
// factory.  The connection settings and handler which have been configured
// with the other options are still applied to it.
func WithServer(srv *http.Server) Option {
	return withAdapter((*Server).setServer, srv)
}

// WithServerFactory provides a factory for obtaining the http.Server.
func WithServerFactory(fn func(context.Context) (*http.Server, error)) Option {
	return withAdapter((*Server).setServerFactory, fn)
}

// WithDefaultServerFactory sets up the default server factory and built-in
// server middleware (connection settings and handler setup).  This option is
// applied automatically by New.
func WithDefaultServerFactory() Option {
	return option[*http.Server]{
		nil, func(s *Server, _ *http.Server) error {
			s.server.SetFactory(s.defaultServerFactory)
			s.server.AddMiddleware( // FIXME A bit dubious since it makes the factory very anemic
				s.setupConnectionSettings,
				s.setupHandler,
			)
			return nil
		},
	}
}

// WithHandler sets the handler which will run on the server
func WithHandler(handler http.Handler) Option {
	return withAdapter((*Server).setHandler, handler)
}

// WithHandlerFactory sets how to create the handler which will run on the server
func WithHandlerFactory(f func(*Server) (http.Handler, error)) Option {
	return withAdapter((*Server).setHandlerFactory, f)
}

// WithReadyFunc sets a callback for when the server is listening
func WithReadyFunc(ready ReadyFunc) Option {
	return withAdapter((*Server).setReadyFunc, ready)
}

// AddReadyFunc appends a callback for when the server is ready
func AddReadyFunc(ready ReadyFunc) Option {
	return withAdapter((*Server).addReadyFunc, ready)
}

// WithShutdownFunc sets a callback for when the server is shutting down
func WithShutdownFunc(shutdown ReadyFunc) Option {
	return withAdapter((*Server).setShutdownFunc, shutdown)
}

// AddShutdownFunc adds a callback for when the server is shutting down
func AddShutdownFunc(shutdown ReadyFunc) Option {
	return withAdapter((*Server).addShutdownFunc, shutdown)
}

// WithMiddleware adds handler middleware
func WithMiddleware(m MiddlewareFunc) Option {
	return withAdapter((*Server).addMiddleware, m)
}

// WithAddr sets the corresponding server field
func WithAddr(addr string) Option {
	return withAdapter((*Server).setAddr, addr)
}

// WithHostname sets the server hostname
func WithHostname(v string) Option {
	return withAdapter((*Server).setHostname, v)
}

// WithPort sets the server port
func WithPort(v int) Option {
	return withAdapter((*Server).setPort, v)
}

// WithShutdownTimeout sets the amount of time to allow the server to shutdown
func WithShutdownTimeout(d time.Duration) Option {
	return withAdapter((*Server).setShutdownTimeout, d)
}

// WithReadTimeout sets the amount of time to allow for reading requests
func WithReadTimeout(d time.Duration) Option {
	return withAdapter((*Server).setReadTimeout, d)
}

// WithReadHeaderTimeout sets the amount of time to allow for reading headers
func WithReadHeaderTimeout(d time.Duration) Option {
	return withAdapter((*Server).setReadHeaderTimeout, d)
}

// WithWriteTimeout sets the amount of time to allow for writing responses
func WithWriteTimeout(d time.Duration) Option {
	return withAdapter((*Server).setWriteTimeout, d)
}

// WithIdleTimeout sets the amount of time to allow for idling
func WithIdleTimeout(d time.Duration) Option {
	return withAdapter((*Server).setIdleTimeout, d)
}

// WithMaxHeaderBytes sets the amount max header bytes
func WithMaxHeaderBytes(n int) Option {
	return withAdapter((*Server).setMaxHeaderBytes, n)
}

// WithTLSKeyFile sets the file to use for the TLS key
func WithTLSKeyFile(filename string) Option {
	return withAdapter((*Server).setTLSKeyFile, filename)
}

// WithTLSCertFile sets the file to use for the TLS cert
func WithTLSCertFile(filename string) Option {
	return withAdapter((*Server).setTLSCertFile, filename)
}

// WithServerHeader sets the contents of the server header
func WithServerHeader(s string) Option {
	return withAdapter((*Server).setServerHeader, s)
}

// WithAccessLog sets the format string for the access log
func WithAccessLog(s string) Option {
	return withAdapter((*Server).setAccessLog, s)
}

// WithNoAccessLog disables the access log
func WithNoAccessLog() Option {
	return WithAccessLog("")
}

// WithStaticDirectory hosts a static directory
func WithStaticDirectory(path string) Option {
	return withAdapter((*Server).setStaticDirectory, path)
}

// WithHideDirectoryListings disables directory listings
func WithHideDirectoryListings(v bool) Option {
	return withAdapter((*Server).setHideDirectoryListings, v)
}

// FromContext obtains the server from the context.
func FromContext(ctx context.Context) *Server {
	return ctx.Value(servicesKey).(*Server)
}

// Handle registers the given handler with the context server
func Handle(path string, h http.Handler) Option {
	return optionFunc(func(s *Server) error {
		return s.Handle(path, h)
	})
}

// HandleFunc registers the given handler with the context server
func HandleFunc(path string, h http.HandlerFunc) Option {
	return optionFunc(func(s *Server) error {
		return s.Handle(path, h)
	})
}

// NewServer creates (or returns the cached) http.Server for the server
func (s *Server) NewServer(ctx context.Context) (*http.Server, error) {
	srv, err := s.server.New(ctx)
	if err != nil {
		return nil, err
	}
	if s.handlerErr != nil {
		return nil, s.handlerErr
	}
	return srv, nil
}

func (s *Server) defaultServerFactory(_ context.Context) (*http.Server, error) {
	return &http.Server{}, nil
}

// setupConnectionSettings copies the connection settings which have been
// configured onto the server.  Only values which were actually set are copied
// so that a server provided by WithServer retains its own settings.
func (s *Server) setupConnectionSettings(_ context.Context, srv *http.Server) *http.Server {
	if s.addr != "" {
		srv.Addr = s.addr
	}
	if s.readTimeout != 0 {
		srv.ReadTimeout = s.readTimeout
	}
	if s.readHeaderTimeout != 0 {
		srv.ReadHeaderTimeout = s.readHeaderTimeout
	}
	if s.writeTimeout != 0 {
		srv.WriteTimeout = s.writeTimeout
	}
	if s.idleTimeout != 0 {
		srv.IdleTimeout = s.idleTimeout
	}
	if s.maxHeaderBytes != 0 {
		srv.MaxHeaderBytes = s.maxHeaderBytes
	}
	return srv
}

func (s *Server) setupHandler(ctx context.Context, srv *http.Server) *http.Server {
	// A server provided by WithServer can supply its own handler, which takes
	// over when no handler was otherwise configured
	if s.handler.Discrete() == nil && srv.Handler != nil {
		s.handler.SetDiscrete(srv.Handler)
	}

	h, err := s.handler.New(ctx)
	if err != nil {
		s.handlerErr = err
		return srv
	}
	srv.Handler = h
	return srv
}

func (s *Server) setServer(srv *http.Server) error {
	s.server.SetDiscrete(srv)
	return nil
}

func (s *Server) setServerFactory(fn func(context.Context) (*http.Server, error)) error {
	s.server.SetFactory(fn)
	return nil
}

func (s *Server) addMiddleware(m MiddlewareFunc) error {
	s.handler.AddMiddleware(func(_ context.Context, h http.Handler) http.Handler {
		return m(h)
	})
	return nil
}

func (s *Server) addShutdownFunc(shutdown ReadyFunc) error {
	s.shutdown = ComposeReadyFuncs(s.shutdown, shutdown)
	return nil
}

func (s *Server) setShutdownFunc(shutdown ReadyFunc) error {
	s.shutdown = shutdown
	return nil
}

func (s *Server) setHandler(handler http.Handler) error {
	s.handler.SetDiscrete(handler)
	return nil
}

func (s *Server) setHandlerFactory(f func(*Server) (http.Handler, error)) error {
	s.handler.SetFactory(func(_ context.Context) (http.Handler, error) {
		return f(s)
	})
	return nil
}

func (s *Server) setReadyFunc(ready ReadyFunc) error {
	s.ready = ready
	return nil
}

func (s *Server) addReadyFunc(ready ReadyFunc) error {
	s.ready = ComposeReadyFuncs(s.ready, ready)
	return nil
}

func (s *Server) HideDirectoryListing() bool {
	return s.hideDirListings
}

// ListenAndServe creates the server if necessary and starts listening
func (s *Server) ListenAndServe(ctx context.Context) error {
	srv, err := s.NewServer(ctx)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	s.actualBind.addr = listener.Addr().String()
	s.actualBind.tls = (s.TLSCertFile() != "")

	if s.TLSCertFile() == "" {
		return srv.Serve(listener)
	}

	return srv.ServeTLS(listener, s.TLSCertFile(), s.TLSKeyFile())
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	srv, err := s.NewServer(ctx)
	if err != nil {
		return err
	}
	return srv.Shutdown(ctx)
}

func (s *Server) ensureMux() (mux, error) {
	h := s.handler.Discrete()
	if m, ok := h.(mux); ok {
		return m, nil
	}
	if h == nil {
		m := &reloadSupport{
			ServeMux: http.NewServeMux(),
		}
		s.handler.SetDiscrete(m)
		return m, nil
	}
	return nil, fmt.Errorf("server handler does not support mux")
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

// ReloadAll causes the server to reload all reloadable handlers.
// To support reloading, the built-in mux must be used. In particular,
// you can't specify a handler or handler factory directly.
func (s *Server) ReloadAll() {
	mux, _ := s.ensureMux()
	if reload, ok := mux.(reloadableMux); ok {
		reload.ReloadAll()
	}
}

func (s *Server) updateAddr(hostname string, port string) error {
	h, p, err := net.SplitHostPort(s.addr)
	if err != nil {
		s.addr = net.JoinHostPort(hostname, port)
		return nil
	}
	if hostname == "" {
		hostname = h
	}
	if port == "" {
		port = p
	}
	s.addr = net.JoinHostPort(hostname, port)
	return nil
}

func (s *Server) ReportListening() error {
	if err := s.checkIfListening(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Listening on %s... (Press ^C to exit)\n", s.url().String())
	return nil
}

// Addr specifies the address that the server will listen on
func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) setHostname(name string) error {
	return s.updateAddr(name, "")
}

func (s *Server) setPort(port int) error {
	return s.updateAddr("", strconv.Itoa(port))
}

func (s *Server) setAddr(addr string) error {
	s.addr = addr
	return nil
}

func (s *Server) setShutdownTimeout(d time.Duration) error {
	s.shutdownTimeout = d
	return nil
}

func (s *Server) setReadTimeout(v time.Duration) error {
	s.readTimeout = v
	return nil
}

func (s *Server) setWriteTimeout(v time.Duration) error {
	s.writeTimeout = v
	return nil
}

func (s *Server) setReadHeaderTimeout(v time.Duration) error {
	s.readHeaderTimeout = v
	return nil
}

func (s *Server) setIdleTimeout(v time.Duration) error {
	s.idleTimeout = v
	return nil
}

func (s *Server) setMaxHeaderBytes(v int) error {
	s.maxHeaderBytes = v
	return nil
}

func (s *Server) setStaticDirectory(path string) error {
	s.staticDir = path
	return nil
}

func (s *Server) setHideDirectoryListings(v bool) error {
	s.hideDirListings = v
	return nil
}

func (s *Server) setAccessLog(v string) error {
	s.accessLog = v
	return nil
}

func (s *Server) setServerHeader(name string) error {
	s.addMiddleware(NewHeaderMiddleware("Server", name))
	return nil
}

// TLSCertFile specifies the certificate file to use in TLS
func (s *Server) TLSCertFile() string {
	return s.tlsCertFile
}

func (s *Server) setTLSCertFile(v string) error {
	s.tlsCertFile = v
	return nil
}

// TLSKeyFile specifies the certificate file to use in TLS
func (s *Server) TLSKeyFile() string {
	return s.tlsKeyFile
}

func (s *Server) setTLSKeyFile(v string) error {
	s.tlsKeyFile = v
	return nil
}

// ShutdownTimeout specifies how long to wait for the server to shutdown
// when a signal is received
func (s *Server) ShutdownTimeout() time.Duration {
	return s.shutdownTimeout
}

func (s *Server) setAction(value cli.Action) error {
	s.Action = value
	return nil
}

func (s *Server) Handle(path string, h http.Handler) (err error) {
	var m mux
	m, err = s.ensureMux()
	if err != nil {
		return
	}

	// Safely handle the panic possible by registering same pattern
	defer func() {
		if rvr := recover(); rvr != nil {
			err = fmt.Errorf("%s", fmt.Sprint(rvr))
		}
	}()
	m.Handle(path, h)
	return
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
