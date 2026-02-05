// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

type requestIDGenerator interface {
	generate(context.Context) (string, error)
}

type requestIDGeneratorFunc func(context.Context) (string, error)
type staticRequestID string

const requestIDBytes = 12

// NewRequestIDMiddleware generates a request ID for each request
// using the X-Request-ID header
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

func (f requestIDGeneratorFunc) generate(c context.Context) (string, error) {
	return f(c)
}

func (s staticRequestID) generate(context.Context) (string, error) {
	return string(s), nil
}
