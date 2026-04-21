// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver_test

import (
	"context"
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"github.com/Carbonfrost/joe-cli-http/httpserver/httpserverfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReloadableHandler", func() {

	Describe("ServeHTTP", func() {

		It("creates the underlying handler", func() {
			fakeHandler := new(httpserverfakes.FakeHandler)

			handler := httpserver.NewReloadableHandler(func(_ context.Context) (http.Handler, error) {
				return fakeHandler, nil
			})
			request, _ := http.NewRequest("GET", "example.com", nil)
			handler.ServeHTTP(nil, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
		})

		It("reuses the underlying handler on subsequent calls", func() {
			var callCount int

			handler := httpserver.NewReloadableHandler(func(_ context.Context) (http.Handler, error) {
				callCount++
				return new(httpserverfakes.FakeHandler), nil
			})

			request, _ := http.NewRequest("GET", "example.com", nil)
			handler.ServeHTTP(nil, request)
			handler.ServeHTTP(nil, request)

			Expect(callCount).To(Equal(1))
		})
	})

	Describe("Invalidate", func() {

		It("recreates the underlying handler", func() {
			var handlers []http.Handler

			handler := httpserver.NewReloadableHandler(func(_ context.Context) (http.Handler, error) {
				result := new(httpserverfakes.FakeHandler)
				handlers = append(handlers, result)
				return result, nil
			})

			request, _ := http.NewRequest("GET", "example.com", nil)
			handler.ServeHTTP(nil, request)

			handler.Invalidate()
			handler.ServeHTTP(nil, request)

			Expect(handlers).To(HaveLen(2))
			Expect(handlers[0]).NotTo(BeIdenticalTo(handlers[1]))
		})
	})

})
