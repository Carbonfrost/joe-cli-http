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

// WithTransport sets the default transport
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) {
		c.transport = t
	}
}

func (c *Client) AddTransportMiddleware(m TransportMiddleware) {
	c.transportMiddleware = append(c.transportMiddleware, m)
}

func (c *Client) actualTransport(ctx context.Context) http.RoundTripper {
	t := c.transport
	if t == nil {
		defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
		defaultTransport.DialContext = c.dialer.DialContext
		defaultTransport.Proxy = http.ProxyFromEnvironment
		t = defaultTransport
	}
	for _, m := range c.generateTransportMiddleware() {
		t = m(ctx, t)
	}
	return t
}

func (c *Client) generateTransportMiddleware() []TransportMiddleware {
	return append([]TransportMiddleware{
		c.setupTLSConfigTransport,
		c.setupTraceLevelTransport,
	}, c.transportMiddleware...)
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
