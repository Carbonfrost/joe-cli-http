// Copyright 2023, 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient_test

import (
	"context"
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("NewRequestIDMiddleware", func() {
	DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		httpclient.NewRequestIDMiddleware(v).Handle(req, nil)

		Expect(req.Header.Get("X-Request-ID")).To(expected)
	},
		Entry("string", "static string", Equal("static string")),
		Entry("string func", func() string { return "s" }, Equal("s")),
		Entry("func", func(context.Context) (string, error) { return "s", nil }, Equal("s")),
		Entry("nil", nil, HaveLen(16)),
	)

	It("uses default on unspecified arg", func() {
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		httpclient.NewRequestIDMiddleware().Handle(req, nil)

		Expect(req.Header.Get("X-Request-ID")).To(HaveLen(16))
	})

	It("panics on unknown type", func() {
		v := 69

		Expect(func() {
			httpclient.NewRequestIDMiddleware(v)
		}).To(Panic())
	})
})

var _ = Describe("WithMethod", func() {
	DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		httpclient.WithMethod(v).Handle(req, nil)

		Expect(req.Method).To(expected)
	},
		Entry("string", "static string", Equal("static string")),
		Entry("string func", func() string { return "s" }, Equal("s")),
		Entry("func", func(*http.Request) (string, error) { return "s", nil }, Equal("s")),
	)
})

var _ = Describe("WithHeader", func() {
	DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		httpclient.WithHeader("TestHead", v).Handle(req, nil)

		Expect(req.Header["Testhead"]).To(expected)
	},
		Entry("string", "static string", Equal([]string{"static string"})),
		Entry("string slice", []string{"a", "b"}, Equal([]string{"a", "b"})),
		Entry("string func", func() string { return "s" }, Equal([]string{"s"})),
		Entry("func", func(*http.Request) (string, error) { return "s", nil }, Equal([]string{"s"})),
		Entry("func", func(*http.Request) ([]string, error) { return []string{"s", "t"}, nil }, Equal([]string{"s", "t"})),
		Entry("other", 120, Equal([]string{"120"})),
	)
})

var _ = Describe("ComposeMiddleware", func() {

	It("processes all middleware funcs", func() {
		var (
			aCalled bool
			bCalled bool

			a httpclient.MiddlewareFunc = func(req *http.Request) error {
				aCalled = true
				return nil
			}
			b httpclient.MiddlewareFunc = func(req *http.Request) error {
				bCalled = true
				return nil
			}
		)

		mw := httpclient.ComposeMiddleware(a, b)
		mw.Handle(nil, nil)

		Expect(aCalled).To(BeTrue())
		Expect(bCalled).To(BeTrue())
	})

	It("custom middleware can skip", func() {
		var (
			bCalled bool

			skipper httpclient.Middleware = &skipperMiddleware{}

			b httpclient.MiddlewareFunc = func(req *http.Request) error {
				bCalled = true
				return nil
			}
		)

		mw := httpclient.ComposeMiddleware(skipper, b)
		mw.Handle(nil, nil)

		Expect(bCalled).To(BeFalse())
	})

	It("ignores nils", func() {
		mw := httpclient.ComposeMiddleware(nil, nil)
		Expect(func() {
			mw.Handle(nil, nil)
		}).ToNot(Panic())
	})

})

type skipperMiddleware struct{}

func (*skipperMiddleware) Handle(r *http.Request, next func(*http.Request) error) error {
	return nil
}
