package pattern

import (
	"context"
)

type Cacheable[T comparable] struct {
	discrete   T
	factory    func(context.Context) (T, error)
	middleware []func(context.Context, T) T
	cached     T
	cachedErr  error
}

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
	} else {
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

func (c *Cacheable[T]) Discrete() T {
	return c.discrete
}

func (c *Cacheable[T]) SetDiscrete(value T) {
	c.discrete = value
}

func (c *Cacheable[T]) SetFactory(value func(context.Context) (T, error)) {
	c.factory = value
}

func (c *Cacheable[T]) AddMiddleware(values ...func(context.Context, T) T) {
	c.middleware = append(c.middleware, values...)
}
