// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pattern provides several patterns
package pattern

import (
	"context"
)

// Cacheable provides a value that can be set or calculated from a factory.
// Aside from this, it accumulates and applies middleware using a common function
// signature
type Cacheable[T comparable] struct {
	discrete   T
	factory    func(context.Context) (T, error)
	middleware []func(context.Context, T) T
	cached     T
	cachedErr  error
}

// New creates the value, either from the discrete value or the factory and applies
// the middleware. The result is cached, and if an error is generated
// it is returned on each of the subsequent calls.
func (c *Cacheable[T]) New(ctx context.Context) (T, error) {
	var zero T
	if c.cachedErr != nil {
		return zero, c.cachedErr
	}
	if c.cached != zero {
		return c.cached, nil
	}

	var result T
	if c.discrete != zero {
		result = c.discrete
	} else if c.factory != nil {
		result, c.cachedErr = c.factory(ctx)
		if c.cachedErr != nil {
			return zero, c.cachedErr
		}
	}

	// Apply middleware
	for _, m := range c.middleware {
		result = m(ctx, result)
	}

	c.cached = result
	return c.cached, nil
}

// Discrete the discrete value to use as the basis of the result
func (c *Cacheable[T]) Discrete() T {
	return c.discrete
}

// SetDiscrete sets a discrete value to use as the basis of the result
func (c *Cacheable[T]) SetDiscrete(value T) {
	c.discrete = value
}

// SetFactory sets the factory that is used to create the value
func (c *Cacheable[T]) SetFactory(value func(context.Context) (T, error)) {
	c.factory = value
}

// AddMiddleware adds a middleware function that will be applied with the result
// is calculated
func (c *Cacheable[T]) AddMiddleware(values ...func(context.Context, T) T) {
	c.middleware = append(c.middleware, values...)
}
