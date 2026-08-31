// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {

	Describe("AddRequestHeader", func() {
		It("aggregates header values", func() {
			s := httpclient.New()
			s.Apply(
				httpclient.AddRequestHeader(&httpclient.HeaderValue{"Link", "Something"}),
				httpclient.AddRequestHeader(&httpclient.HeaderValue{"Link", "SomethingElse"}),
			)

			Expect(newRequest(s).Header).To(HaveKeyWithValue("Link", []string{"Something", "SomethingElse"}))
		})
	})

	Describe("WithBodyContentString", func() {
		It("sets raw body value", func() {
			s := httpclient.New()
			s.Apply(httpclient.WithBodyContentString("raw content"))

			body, _ := io.ReadAll(s.BodyContent.Read())
			Expect(string(body)).To(Equal("raw content"))
		})
	})

	Describe("New", func() {
		It("sets up the default user agent string", func() {
			s := httpclient.New()
			expected := "Go-http-client/1.1 (joe-cli-http/(devel), +https://github.com/Carbonfrost/joe-cli-http)"
			Expect(newRequest(s).Header).To(HaveKeyWithValue("User-Agent", []string{expected}))
		})
	})

	Describe("NewRequest", func() {
		It("applies the configured method to the request", func() {
			s := httpclient.New(httpclient.WithRequestMethod("patch"))

			Expect(newRequest(s).Method).To(Equal("PATCH"))
		})

		It("caches the request", func() {
			s := httpclient.New()

			Expect(newRequest(s)).To(BeIdenticalTo(newRequest(s)))
		})

		It("applies the configured values to the request from WithRequest", func() {
			s := httpclient.New(
				httpclient.WithRequest(&http.Request{Method: http.MethodPut, Close: true}),
				httpclient.AddRequestHeader(&httpclient.HeaderValue{"Link", "Something"}),
			)

			Expect(newRequest(s)).To(HaveField("Method", Equal(http.MethodPut)))
			Expect(newRequest(s)).To(HaveField("Close", BeTrue()))
			Expect(newRequest(s).Header).To(HaveKeyWithValue("Link", []string{"Something"}))
		})
	})
})

var _ = Describe("Do", func() {

	It("processes middleware on a copy of the request per location", func() {
		var actual []*http.Request

		u, _ := url.Parse("https://example.com/a")
		v, _ := url.Parse("https://example.com/b")

		client := httpclient.New(
			httpclient.WithTransport(httpclient.RoundTripperFunc(func(r *http.Request) *http.Response {
				actual = append(actual, r)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
				}
			})),
			httpclient.WithURL(u),
			httpclient.WithURL(v),
			httpclient.WithQueryString(&cli.NameValue{Name: "q", Value: "1"}),
			httpclient.WithRequestID(),
		)
		app := &cli.App{
			Uses:   client,
			Action: httpclient.FetchAndPrint(),
			Stdout: io.Discard,
		}

		err := app.RunContext(context.Background(), []string{"_"})
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(HaveLen(2))

		// Each location has its own URL and query string rather than
		// accumulating the values from the previous one
		Expect(actual[0].URL.String()).To(Equal("https://example.com/a?q=1"))
		Expect(actual[1].URL.String()).To(Equal("https://example.com/b?q=1"))

		// Middleware runs for each request, so each has its own request ID
		Expect(actual[0].Header.Get("X-Request-Id")).NotTo(BeEmpty())
		Expect(actual[1].Header.Get("X-Request-Id")).NotTo(Equal(actual[0].Header.Get("X-Request-Id")))
	})
})

func newRequest(c *httpclient.Client) *http.Request {
	r, err := c.NewRequest(context.Background())
	Expect(err).NotTo(HaveOccurred())
	return r
}
