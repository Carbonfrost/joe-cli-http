// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"net/http"
	"os"
)

// TransportMiddleware provides middleware to the roundtripper
type TransportMiddleware func(context.Context, http.RoundTripper) http.RoundTripper

// RoundTripperFunc provides a transport implementation based upon a function
type RoundTripperFunc func(req *http.Request) *http.Response

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// WithTransport sets the transport to use directly, bypassing the default factory
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) {
		c.transport.discrete = t
	}
}

// WithTransportFactory provides a factory for obtaining the transport
func WithTransportFactory(fn func(context.Context) (http.RoundTripper, error)) Option {
	return func(c *Client) {
		c.transport.factory = fn
	}
}

// WithDefaultTransportFactory sets up the default transport factory and built-in
// transport middleware (TLS config and trace level).  This option is applied
// automatically by New.
func WithDefaultTransportFactory() Option {
	return func(c *Client) {
		c.transport.factory = c.defaultTransportFactory
		c.transport.middleware = append(
			[]func(context.Context, http.RoundTripper) http.RoundTripper{
				c.setupTLSConfigTransport,
				c.setupTraceLevelTransport,
			},
			c.transport.middleware...,
		)
	}
}

func (c *Client) defaultTransportFactory(_ context.Context) (http.RoundTripper, error) {
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.DialContext = c.dialer.DialContext
	defaultTransport.Proxy = http.ProxyFromEnvironment
	return defaultTransport, nil
}

// NewTransport creates (or returns the cached) transport for the client
func (c *Client) NewTransport(ctx context.Context) (http.RoundTripper, error) {
	return c.transport.New(ctx)
}

func (c *Client) AddTransportMiddleware(m TransportMiddleware) {
	c.transport.middleware = append(c.transport.middleware, m)
}

func (c *Client) actualTransport(ctx context.Context) http.RoundTripper {
	t, _ := c.transport.New(ctx)
	return t
}

func (c *Client) setupTLSConfigTransport(ctx context.Context, t http.RoundTripper) http.RoundTripper {
	// TODO Error if not default transport; better error handling
	if defaultTransport, ok := t.(*http.Transport); ok {
		defaultTransport.TLSClientConfig, _ = c.tls.New(ctx)
	}

	return t
}

func (c *Client) setupTraceLevelTransport(ctx context.Context, t http.RoundTripper) http.RoundTripper {
	var logger TraceLogger
	if c.traceLevel == TraceOff {
		logger = nopTraceLogger{}
	} else {
		logger = &defaultTraceLogger{
			template: traceTemplate(ctx),
			out:      os.Stderr,
			flags:    c.traceLevel,
		}
	}
	c.logger = logger
	return &traceableTransport{
		logger:    logger,
		Transport: t,
	}
}
