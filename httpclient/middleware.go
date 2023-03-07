package httpclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
)

type Middleware interface {
	Handle(req *http.Request) error
}

type MiddlewareFunc func(req *http.Request) error

type requestIDGenerator interface {
	generate(context.Context) (string, error)
}

type requestIDGeneratorFunc func(context.Context) (string, error)
type staticRequestID string

const requestIDBytes = 12

func NewRequestIDMiddleware(v ...any) Middleware {
	var gen requestIDGenerator
	switch len(v) {
	case 0:
		gen = asRequestIDGenerator(nil)
	case 1:
		gen = asRequestIDGenerator(v[0])
	default:
		panic(expectedOneArg)
	}

	return WithHeader("X-Request-ID", func(req *http.Request) (string, error) {
		return gen.generate(req.Context())
	})
}

// WithHeader sets the specified header.  The value may be:
//   - string
//   - func()string.
//   - func(*http.Request)(string, error).
//
// Other types using their default string format.
func WithHeader(name string, value any) Middleware {
	return MiddlewareFunc(func(r *http.Request) (err error) {
		var headerValue string
		switch v := value.(type) {
		case string:
			headerValue = v
		case func() string:
			headerValue = v()
		case func(*http.Request) (string, error):
			headerValue, err = v(r)
			if err != nil {
				return
			}
		default:
			headerValue = fmt.Sprint(v)
		}

		ensureHeader(r).Set(name, headerValue)
		return
	})
}

// WithHeaders sets the specified headers.
func WithHeaders(headers http.Header) Middleware {
	return MiddlewareFunc(func(r *http.Request) error {
		to := ensureHeader(r)
		for k, v := range headers {
			to[http.CanonicalHeaderKey(k)] = v
		}
		return nil
	})
}

func setupBodyContent(c *Client) MiddlewareFunc {
	return func(r *http.Request) error {
		if len(c.bodyForm) > 0 {
			c.ensureBodyContent()
		}
		if c.BodyContent != nil {
			for _, k := range c.bodyForm {
				err := c.BodyContent.Set(k.Name, k.Name)
				if err != nil {
					return err
				}
			}
			if c.Request.Header.Get("Content-Type") == "" {
				if ct := c.BodyContent.ContentType(); ct != "" {
					c.Request.Header.Set("Content-Type", ct)
				}
			}
			c.Request.Body = wrapReader(c.BodyContent.Read())
		}
		return nil
	}
}

func setupQueryString(c *Client) MiddlewareFunc {
	return func(r *http.Request) error {
		query := c.Request.URL.Query()
		for k, v := range c.queryString {
			query[k] = append(query[k], v...)
		}

		c.Request.URL.RawQuery = query.Encode()
		return nil
	}
}

func processAuth(c *Client) MiddlewareFunc {
	return func(r *http.Request) error {
		err := c.applyAuth()
		if err != nil {
			return err
		}
		return c.loadClientTLSCreds()
	}
}

func (f requestIDGeneratorFunc) generate(c context.Context) (string, error) {
	return f(c)
}

func (f MiddlewareFunc) Handle(req *http.Request) error {
	if f == nil {
		return nil
	}
	return f(req)
}

func (s staticRequestID) generate(context.Context) (string, error) {
	return string(s), nil
}

func asRequestIDGenerator(v any) requestIDGenerator {
	switch t := v.(type) {
	case nil:
		return requestIDGeneratorFunc(defaultRequestIDGenerator)
	case string:
		return staticRequestID(t)
	case func() string:
		return requestIDGeneratorFunc(func(context.Context) (string, error) {
			return t(), nil
		})
	case requestIDGenerator:
		return t
	case func(context.Context) (string, error):
		return requestIDGeneratorFunc(t)
	}
	panic(fmt.Errorf("unusable type for request ID generator: %T", v))
}

func defaultRequestIDGenerator(context.Context) (string, error) {
	b := make([]byte, requestIDBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	res := base64.StdEncoding.EncodeToString(b)
	return res, nil
}
