// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dialer provides a context-bound service which wraps a net.Dialer
// and the flags that configure it.
package dialer

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/pattern"
)

type contextKey string

const defaultContextKey contextKey = "net_dialer_services"

var errNoDialer = errors.New("dialer: no dialer found in context")

type cacheable[T comparable] = pattern.Cacheable[T]

// Dialer represents a net.Dialer plus a default action.  Dialer is created
// with New, and its settings are configured either with options or by the
// flags which the default action registers.
type Dialer struct {
	*net.Dialer
	cli.Action

	contextKey    any
	defaultAction bool
	resolver      cacheable[*net.Resolver]

	// mu guards the lazy initialization of the resolver
	mu       sync.Mutex
	prepared bool
	err      error
}

// Option provides an option to the dialer.  Option can be used as an Action,
// typically within the Uses or Before pipeline, in which case it applies to
// the dialer found in the context.
type Option func(*Dialer) error

// ResolverMiddleware provides middleware to the resolver
type ResolverMiddleware func(context.Context, *net.Resolver) *net.Resolver

// New creates a new dialer.  By default, it is also initialized with a default
// action that registers the dialer in the context and registers useful flags.
func New(opts ...Option) *Dialer {
	d := &Dialer{
		Dialer:     new(net.Dialer),
		contextKey: defaultContextKey,
	}

	if err := d.Apply(defaultOptions()...); err != nil {
		d.err = err

	} else if err := d.Apply(opts...); err != nil {
		d.err = err
	}

	return d
}

func defaultOptions() []Option {
	return []Option{
		WithDefaultAction(),
		WithDefaultResolverFactory(),
	}
}

// Apply applies the given options to the dialer. Options must be applied before
// the first connection is dialed.
func (d *Dialer) Apply(opts ...Option) error {
	for _, o := range opts {
		err := o(d)
		if err != nil {
			return err
		}
	}
	return nil
}

// Pipeline obtains the action that the dialer contributes to the app
func (d *Dialer) Pipeline() cli.Action {
	return d.Action
}

// ContextKey gets the key used to store the dialer in the context.
func (d *Dialer) ContextKey() any {
	return d.contextKey
}

// WithAction sets the action
func WithAction(a cli.Action) Option {
	return func(d *Dialer) error {
		d.Action = a
		d.defaultAction = false
		return nil
	}
}

// WithDefaultAction sets the action to the default, which registers the
// dialer in the context along with the flags that configure it.
func WithDefaultAction() Option {
	return func(d *Dialer) error {
		d.Action = d.newDefaultAction()
		d.defaultAction = true
		return nil
	}
}

// WithContextKey sets the key under which the dialer is stored in the
// context.  If this option is not specified, a default key is used.
func WithContextKey(contextKey any) Option {
	return func(d *Dialer) error {
		d.contextKey = contextKey
		if d.defaultAction {
			d.Action = d.newDefaultAction()
		}
		return nil
	}
}

func (d *Dialer) newDefaultAction() cli.Action {
	return cli.Pipeline(
		ContextValue(d),
		FlagsAndArgs(d.contextKey),
	)
}

// ContextValue provides an action that registers the dialer in the context
// using its context key.
func ContextValue(d *Dialer) Action {
	return cli.WithContext(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, d.contextKey, d)
	})
}

// FromContext obtains the dialer from the context.  When the context is
// a *cli.Context, the ContextKeyAnnotation of the flag, arg, or command being
// executed can be used to determine which dialer is used; otherwise, or if
// the annotation is not present, the default context key is used to resolve
// the dialer.
func FromContext(ctx context.Context) *Dialer {
	var key any = defaultContextKey
	if c, ok := cli.TryFromContext(ctx); ok {
		if k, ok := LookupContextKey(c); ok {
			key = k
		}
	}
	return FromContextKey(ctx, key)
}

// FromContextKey obtains the dialer from the context using the given
// context key, which is the one specified with WithContextKey.  If no dialer
// is present, nil is returned.
func FromContextKey(ctx context.Context, contextKey any) *Dialer {
	d, _ := ctx.Value(contextKey).(*Dialer)
	return d
}

// Dial connects to the address on the named network.  See net.Dialer.Dial.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext connects to the address on the named network using the context.
// It first obtains the resolver if that hasn't already happened.  See
// net.Dialer.DialContext.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if err := d.prepare(ctx); err != nil {
		return nil, err
	}
	return d.Dialer.DialContext(ctx, network, address)
}

// NewResolver creates (or returns the cached) resolver for the dialer.  It
// is created the first time that it is needed, which is either from Dial,
// DialContext, or calling this method.
func (d *Dialer) NewResolver(ctx context.Context) (*net.Resolver, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.resolver.New(ctx)
}

func (d *Dialer) prepare(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.prepared {
		return nil
	}
	if d.err != nil {
		return d.err
	}
	r, err := d.resolver.New(ctx)
	if err != nil {
		return err
	}
	if r != nil {
		d.Dialer.Resolver = r
	}
	d.prepared = true
	return nil
}

// WithDialTimeout sets the maximum amount of time a dial waits for a
// connect to complete
func WithDialTimeout(v time.Duration) Option {
	return func(d *Dialer) error {
		d.Timeout = v
		return nil
	}
}

// WithDialKeepAlive sets the interval between keep-alive probes for an
// active network connection
func WithDialKeepAlive(v time.Duration) Option {
	return func(d *Dialer) error {
		d.KeepAlive = v
		return nil
	}
}

// WithDisableDialKeepAlive disables dialer keep-alive probes
func WithDisableDialKeepAlive(v bool) Option {
	return func(d *Dialer) error {
		if v {
			d.KeepAlive = time.Duration(-1)
		}
		return nil
	}
}

// WithBindAddress binds connections to the given address on the local
// machine
func WithBindAddress(v string) Option {
	return func(d *Dialer) error {
		addr, err := net.ResolveTCPAddr("tcp", v)
		if err != nil {
			return err
		}
		d.LocalAddr = addr
		return nil
	}
}

// WithInterface uses the network interface, named by name or address, to
// connect
func WithInterface(v string) Option {
	return func(d *Dialer) error {
		addr, err := resolveInterface(v)
		if err != nil {
			return err
		}
		d.LocalAddr = addr
		return nil
	}
}

// WithResolver sets the resolver to use directly, bypassing the default
// factory.  Resolver middleware is still applied to it.
func WithResolver(r *net.Resolver) Option {
	return func(d *Dialer) error {
		d.resolver.SetDiscrete(r)
		return nil
	}
}

// WithResolverFactory provides a factory for obtaining the resolver.
func WithResolverFactory(fn func(context.Context) (*net.Resolver, error)) Option {
	return func(d *Dialer) error {
		d.resolver.SetFactory(fn)
		return nil
	}
}

// WithDefaultResolverFactory sets up the default resolver factory, which
// provides a resolver that uses the system defaults.  This option is applied
// automatically by New.
func WithDefaultResolverFactory() Option {
	return WithResolverFactory(func(context.Context) (*net.Resolver, error) {
		return new(net.Resolver), nil
	})
}

// WithResolverMiddleware adds middleware for the resolver
func WithResolverMiddleware(fns ...ResolverMiddleware) Option {
	return func(d *Dialer) error {
		for _, fn := range fns {
			d.resolver.AddMiddleware(fn)
		}
		return nil
	}
}

// Execute applies the option to the dialer found in the context.
func (o Option) Execute(ctx context.Context) error {
	d := FromContext(ctx)
	if d == nil {
		return errNoDialer
	}
	return o(d)
}
