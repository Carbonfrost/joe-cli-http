// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package websocket can connect to a WebSocket server from within a CLI app.
package websocket

import (
	"bytes"
	"context"
	gotls "crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/internal/pattern"
	joetls "github.com/Carbonfrost/joe-cli-http/tls"
	ws "github.com/gorilla/websocket"
)

// Client provides a WebSocket client (which encapsulates a websocket.Dialer)
// that can be accessed from commands, flags, and args within Joe applications.
// The client is used within the Uses pipeline where it registers itself as a
// context service together with the flags and args that configure it.  The
// action ConnectAndPrint is used to actually exchange messages:
//
//	&cli.App{
//	   Name: "gows",
//	   Uses: websocket.New(),
//	   Action: websocket.ConnectAndPrint(),
//	}
//
// This simple app has numerous flags to configure the handshake and message
// handling, and its simplest invocation could be something like
//
//	gows ws://example.com/graphql
//
// The client is configured exclusively with Options, either passed to New or
// applied later using Apply.  The underlying websocket.Dialer is created on
// demand the first time it is needed, which is available from NewDialer.  How
// it gets created can be customized with WithDialer or WithDialerFactory.
//
// The cmd/mop package provides mop, which is a command line utility very
// similar to this.
//
// If you only want to add the Client to the context (typically in advanced
// scenarios where you are deeply customizing the behavior), you only use the
// action websocket.ContextValue() with the client you want to add instead of
// add the client to the pipeline directly.
type Client struct {
	cli.Action

	dialer cacheable[*ws.Dialer]
	tls    cacheable[*gotls.Config]

	location *URLValue
	header   http.Header

	subprotocols     []string
	handshakeTimeout time.Duration
	readBufferSize   int
	writeBufferSize  int
	readLimit        int64
	compression      bool

	messageType  MessageType
	readTimeout  time.Duration
	writeTimeout time.Duration
	closeTimeout time.Duration

	messages []string
	input    *cli.FileSet
	verbose  bool

	// dialerErr records the error, if any, from setting up the dialer,
	// which is reported from NewDialer
	dialerErr error
}

// Option is an option to configure the client.
// Option can be used as an Action, typically within the Uses or Before pipeline.
type Option interface {
	cli.Action
	apply(*Client)
}

type option[T any] struct {
	val T
	fn  func(*Client, T) error
}

type cacheable[T comparable] = pattern.Cacheable[T]

type contextKey string

const servicesKey contextKey = "websocket_services"

const (
	defaultReadTimeout  = 60 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultCloseTimeout = 3 * time.Second
)

var (
	// ErrNoLocation is reported when no URL has been specified for the client
	ErrNoLocation = errors.New("no WebSocket URL specified")

	newline = []byte("\n")
)

// New creates a new WebSocket client with the given options.
func New(options ...Option) *Client {
	c := &Client{}
	c.Apply(defaultOptions()...)
	c.Apply(options...)
	return c
}

func defaultOptions() []Option {
	return []Option{
		WithDefaultAction(),
		WithDefaultDialerFactory(),
		WithMessageType(TextMessage),
		WithReadTimeout(defaultReadTimeout),
		WithWriteTimeout(defaultWriteTimeout),
		WithCloseTimeout(defaultCloseTimeout),
	}
}

// Apply applies the given options to the client
func (c *Client) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(c)
	}
}

// Pipeline obtains the action that the client contributes to the app
func (c *Client) Pipeline() cli.Action {
	return c.Action
}

// FromContext obtains the client stored in the context
func FromContext(ctx context.Context) *Client {
	return ctx.Value(servicesKey).(*Client)
}

// Do connects the context client and exchanges its messages
func Do(ctx context.Context) error {
	return FromContext(ctx).Do(ctx)
}

// WithAction sets the action
func WithAction(a cli.Action) Option {
	return withAdapter((*Client).setAction, a)
}

// WithDefaultAction sets the action to the default, which registers the
// client in the context along with the flags and args that configure it.
func WithDefaultAction() Option {
	return option[cli.Action]{
		nil, func(c *Client, _ cli.Action) error {
			c.Action = cli.Pipeline(
				ContextValue(c),
				FlagsAndArgs(),
				joetls.New(),
				WithDefaultTLSConfigFactory(),
			)
			return nil
		},
	}
}

// WithDialer sets the websocket.Dialer to use directly, bypassing the default
// factory.  The connection settings which have been configured with the other
// options are still applied to it.
func WithDialer(d *ws.Dialer) Option {
	return withAdapter((*Client).setDialer, d)
}

// WithDialerFactory provides a factory for obtaining the websocket.Dialer.
func WithDialerFactory(fn func(context.Context) (*ws.Dialer, error)) Option {
	return withAdapter((*Client).setDialerFactory, fn)
}

// WithDefaultDialerFactory sets up the default dialer factory and the built-in
// dialer middleware (connection settings and TLS setup).  This option is
// applied automatically by New.
func WithDefaultDialerFactory() Option {
	return option[*ws.Dialer]{
		nil, func(c *Client, _ *ws.Dialer) error {
			c.dialer.SetFactory(defaultDialerFactory)
			c.dialer.AddMiddleware(
				c.setupConnectionSettings,
				c.setupTLSConfig,
			)
			return nil
		},
	}
}

// WithTLSConfig sets the TLS config for use on the client
func WithTLSConfig(t *gotls.Config) Option {
	return withAdapter((*Client).setTLSConfig, t)
}

// WithTLSConfigFactory provides a factory for obtaining TLS config
func WithTLSConfigFactory(fn func(context.Context) (*gotls.Config, error)) Option {
	return withAdapter((*Client).setTLSConfigFactory, fn)
}

// WithDefaultTLSConfigFactory provides the default factory, which provides
// TLS from the context
func WithDefaultTLSConfigFactory() Option {
	return WithTLSConfigFactory(func(ctx context.Context) (*gotls.Config, error) {
		return joetls.FromContext(ctx).Config, nil
	})
}

// WithURL sets the location that the client connects to.  The text is
// interpreted as described by URLValue.
func WithURL(loc string) Option {
	return withURLValue(NewURLValue(loc))
}

// WithHeader adds a header to the handshake request.  Note that the
// Sec-WebSocket-Protocol header is set from the subprotocols which have been
// configured with WithSubprotocol rather than being set directly.
func WithHeader(name, value string) Option {
	return withAdapter((*Client).addHeader, httpclient.HeaderValue{Name: name, Value: value})
}

// WithOrigin sets the Origin header used in the handshake request
func WithOrigin(origin string) Option {
	return WithHeader("Origin", origin)
}

// WithSubprotocol adds a subprotocol which is requested during the handshake
func WithSubprotocol(name string) Option {
	return withAdapter((*Client).addSubprotocol, name)
}

// WithMessage adds a message which is sent once the connection is established.
// Messages are sent before any which are read from the input.
func WithMessage(text string) Option {
	return withAdapter((*Client).addMessage, text)
}

// WithInput sets the files that provide the messages to send, one message per
// line.  When the file set is empty, standard input is used unless it is a
// terminal.
func WithInput(files *cli.FileSet) Option {
	return withAdapter((*Client).setInput, files)
}

// WithMessageType sets the type of the messages which are sent
func WithMessageType(m MessageType) Option {
	return withAdapter((*Client).setMessageType, m)
}

// WithHandshakeTimeout sets the amount of time to allow the handshake
// to complete
func WithHandshakeTimeout(d time.Duration) Option {
	return withAdapter((*Client).setHandshakeTimeout, d)
}

// WithReadTimeout sets the amount of time to wait for each message which is
// received.  When the timeout elapses, the exchange has ended.
func WithReadTimeout(d time.Duration) Option {
	return withAdapter((*Client).setReadTimeout, d)
}

// WithWriteTimeout sets the amount of time to allow for writing each message
func WithWriteTimeout(d time.Duration) Option {
	return withAdapter((*Client).setWriteTimeout, d)
}

// WithCloseTimeout sets the amount of time to allow for the closing handshake
func WithCloseTimeout(d time.Duration) Option {
	return withAdapter((*Client).setCloseTimeout, d)
}

// WithReadBufferSize sets the size of the read buffer in bytes
func WithReadBufferSize(n int) Option {
	return withAdapter((*Client).setReadBufferSize, n)
}

// WithWriteBufferSize sets the size of the write buffer in bytes
func WithWriteBufferSize(n int) Option {
	return withAdapter((*Client).setWriteBufferSize, n)
}

// WithReadLimit sets the maximum size in bytes of the messages which can be
// received
func WithReadLimit(n int64) Option {
	return withAdapter((*Client).setReadLimit, n)
}

// WithCompression enables negotiation of per-message compression
func WithCompression(v bool) Option {
	return withAdapter((*Client).setCompression, v)
}

// WithVerbose causes the messages which are sent to be echoed to stderr
func WithVerbose(v bool) Option {
	return withAdapter((*Client).setVerbose, v)
}

// NewDialer creates (or returns the cached) websocket.Dialer for the client
func (c *Client) NewDialer(ctx context.Context) (*ws.Dialer, error) {
	d, err := c.dialer.New(ctx)
	if err != nil {
		return nil, err
	}
	if c.dialerErr != nil {
		return nil, c.dialerErr
	}
	return d, nil
}

// NewTLSConfig creates the TLS config
func (c *Client) NewTLSConfig(ctx context.Context) (*gotls.Config, error) {
	return c.tls.New(ctx)
}

// Location obtains the URL that the client connects to
func (c *Client) Location() (*URLValue, error) {
	if c.location == nil {
		return nil, ErrNoLocation
	}
	return c.location, nil
}

// Dial establishes the connection to the location which has been configured
func (c *Client) Dial(ctx context.Context) (*Conn, error) {
	dialer, err := c.NewDialer(ctx)
	if err != nil {
		return nil, err
	}

	loc, err := c.Location()
	if err != nil {
		return nil, err
	}

	u, err := loc.URL()
	if err != nil {
		return nil, err
	}

	conn, resp, err := dialer.DialContext(ctx, u.String(), c.header)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", u, err)
	}
	if c.readLimit > 0 {
		conn.SetReadLimit(c.readLimit)
	}

	return &Conn{
		Conn:         conn,
		response:     resp,
		messageType:  c.messageType,
		readTimeout:  c.readTimeout,
		writeTimeout: c.writeTimeout,
		closeTimeout: c.closeTimeout,
	}, nil
}

// Do establishes the connection, sends each message which has been configured,
// and then copies each message which is received to the output until the peer
// closes the connection or the read timeout elapses.
func (c *Client) Do(ctx context.Context) error {
	conn, err := c.Dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := c.send(ctx, conn); err != nil {
		return err
	}
	if err := c.receive(ctx, conn); err != nil {
		return err
	}
	return conn.CloseNormal()
}

func (c *Client) send(ctx context.Context, conn *Conn) error {
	for _, msg := range c.messages {
		if err := c.sendOne(ctx, conn, msg); err != nil {
			return err
		}
	}

	input := c.ensureInput(ctx).Input()
	for msg := range input.Lines() {
		if err := c.sendOne(ctx, conn, msg); err != nil {
			return err
		}
	}
	return input.Err()
}

func (c *Client) sendOne(ctx context.Context, conn *Conn, msg string) error {
	if err := conn.Send([]byte(msg)); err != nil {
		return err
	}
	c.trace(ctx, "→ %s", msg)
	return nil
}

func (c *Client) receive(ctx context.Context, conn *Conn) error {
	out := stdout(ctx)
	for {
		_, msg, err := conn.Receive()
		if err != nil {
			if isEndOfStream(err) {
				return nil
			}
			return err
		}
		if _, err := out.Write(msg); err != nil {
			return err
		}

		// Each message occupies its own line, but the message could have
		// already provided the line ending itself
		if !bytes.HasSuffix(msg, newline) {
			if _, err := out.Write(newline); err != nil {
				return err
			}
		}
	}
}

func (c *Client) trace(ctx context.Context, format string, args ...any) {
	if !c.verbose {
		return
	}
	fmt.Fprintf(stderr(ctx), format+"\n", args...)
}

// ensureInput obtains the file set that provides the messages to send.  The
// file system comes from the context so that standard input is the one that
// the app was configured with.
func (c *Client) ensureInput(ctx context.Context) *cli.FileSet {
	if c.input == nil {
		c.input = &cli.FileSet{}
	}
	if c.input.FS == nil {
		if cc, ok := cli.TryFromContext(ctx); ok {
			c.input.FS = cc.FS
		}
	}
	return c.input
}

func (c *Client) ensureHeader() http.Header {
	if c.header == nil {
		c.header = http.Header{}
	}
	return c.header
}

// setupConnectionSettings copies the connection settings which have been
// configured onto the dialer.  Only values which were actually set are copied
// so that a dialer provided by WithDialer retains its own settings.
func (c *Client) setupConnectionSettings(_ context.Context, d *ws.Dialer) *ws.Dialer {
	if c.handshakeTimeout != 0 {
		d.HandshakeTimeout = c.handshakeTimeout
	}
	if c.readBufferSize != 0 {
		d.ReadBufferSize = c.readBufferSize
	}
	if c.writeBufferSize != 0 {
		d.WriteBufferSize = c.writeBufferSize
	}
	if len(c.subprotocols) > 0 {
		d.Subprotocols = c.subprotocols
	}
	if c.compression {
		d.EnableCompression = true
	}
	return d
}

func (c *Client) setupTLSConfig(ctx context.Context, d *ws.Dialer) *ws.Dialer {
	config, err := c.tls.New(ctx)
	if err != nil {
		c.dialerErr = err
		return d
	}
	if config != nil {
		d.TLSClientConfig = config
	}
	return d
}

func (c *Client) setAction(v cli.Action) error {
	c.Action = v
	return nil
}

func (c *Client) setDialer(d *ws.Dialer) error {
	c.dialer.SetDiscrete(d)
	return nil
}

func (c *Client) setDialerFactory(fn func(context.Context) (*ws.Dialer, error)) error {
	c.dialer.SetFactory(fn)
	return nil
}

func (c *Client) setTLSConfig(t *gotls.Config) error {
	c.tls.SetDiscrete(t)
	return nil
}

func (c *Client) setTLSConfigFactory(fn func(context.Context) (*gotls.Config, error)) error {
	c.tls.SetFactory(fn)
	return nil
}

func (c *Client) setURLValue(v *URLValue) error {
	c.location = v
	return nil
}

func (c *Client) addHeader(v httpclient.HeaderValue) error {
	c.ensureHeader().Add(v.Name, v.Value)
	return nil
}

func (c *Client) addSubprotocol(name string) error {
	c.subprotocols = append(c.subprotocols, name)
	return nil
}

func (c *Client) addMessage(text string) error {
	c.messages = append(c.messages, text)
	return nil
}

func (c *Client) setInput(files *cli.FileSet) error {
	c.input = files
	return nil
}

func (c *Client) setMessageType(m MessageType) error {
	if m == 0 {
		return nil
	}
	c.messageType = m
	return nil
}

func (c *Client) setHandshakeTimeout(d time.Duration) error {
	c.handshakeTimeout = d
	return nil
}

func (c *Client) setReadTimeout(d time.Duration) error {
	c.readTimeout = d
	return nil
}

func (c *Client) setWriteTimeout(d time.Duration) error {
	c.writeTimeout = d
	return nil
}

func (c *Client) setCloseTimeout(d time.Duration) error {
	c.closeTimeout = d
	return nil
}

func (c *Client) setReadBufferSize(n int) error {
	c.readBufferSize = n
	return nil
}

func (c *Client) setWriteBufferSize(n int) error {
	c.writeBufferSize = n
	return nil
}

func (c *Client) setReadLimit(n int64) error {
	c.readLimit = n
	return nil
}

func (c *Client) setCompression(v bool) error {
	c.compression = v
	return nil
}

func (c *Client) setVerbose(v bool) error {
	c.verbose = v
	return nil
}

func (o option[_]) Execute(ctx context.Context) error {
	return o.fn(FromContext(ctx), o.val)
}

func (o option[_]) apply(c *Client) {
	_ = o.fn(c, o.val)
}

func withAdapter[T any](fn func(*Client, T) error, value T) Option {
	return option[T]{value, fn}
}

func withURLValue(v *URLValue) Option {
	return withAdapter((*Client).setURLValue, v)
}

func withHeaderValue(v *httpclient.HeaderValue) Option {
	return WithHeader(v.Name, v.Value)
}

func defaultDialerFactory(_ context.Context) (*ws.Dialer, error) {
	return &ws.Dialer{
		Proxy: http.ProxyFromEnvironment,
	}, nil
}

// isEndOfStream determines whether the error indicates that no further messages
// will be received, which happens when the peer closes the connection or when
// the read timeout elapses.
func isEndOfStream(err error) bool {
	if ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
		return true
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return true
	}
	return false
}

func stdout(ctx context.Context) io.Writer {
	if c, ok := cli.TryFromContext(ctx); ok {
		return c.Stdout
	}
	return os.Stdout
}

func stderr(ctx context.Context) io.Writer {
	if c, ok := cli.TryFromContext(ctx); ok {
		return c.Stderr
	}
	return os.Stderr
}
