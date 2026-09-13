// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver_test

import (
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"github.com/Carbonfrost/joe-cli-http/internal/httpserverfakes"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ExpandRequest", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		request, _ := http.NewRequest("GET", "https://example.com/whistle", nil)
		e := expander.Compile(text)

		ww := new(httpserverfakes.FakeWrapResponseWriter)
		ww.BytesWrittenReturns(800)
		ww.HeaderReturns(http.Header{
			http.CanonicalHeaderKey("X-Request-ID"): []string{"732"},
		})
		ww.StatusReturns(429)
		expander := httpserver.ExpandRequest(request, ww)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("bytesWritten", "%(bytesWritten)", Equal("800")),
		Entry("method", "%(method)", Equal("GET")),
		Entry("protocol", "%(protocol)", Equal("HTTP/1.1")),
		Entry("status", "%(status)", Equal("429 Too Many Requests")),
		Entry("statusCode", "%(statusCode)", Equal("429")),
		Entry("urlPath", "%(urlPath)", Equal("/whistle")),
		Entry("header", "%(header)", Equal("X-Request-Id: 732\r\n")),
		Entry("header direct name", "%(header.X-Request-ID)", Equal("732")),
		Entry("header non-canonical name", "%(header.x-request-id)", Equal("732")),
		Entry("header camel name", "%(header.xRequestId)", Equal("732")),
	)
})
