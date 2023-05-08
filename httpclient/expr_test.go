package httpclient_test

import (
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Expr", func() {

	var res = &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header: http.Header{
			"X-Request-Id":     []string{"3305039"},
			"Content-Location": []string{"https://example.com/d"},
		},
		ContentLength: 80,
	}
	DescribeTable("examples", func(text string, res *http.Response, expected types.GomegaMatcher) {
		e := httpclient.Expr(text).Compile()
		expander := expr.ComposeExpanders(
			httpclient.ExpandResponse(&httpclient.Response{Response: res}),
			expr.Unknown,
		)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("status", "%(status)", res, Equal("200 OK")),
		Entry("status code", "%(statusCode)", res, Equal("200")),
		Entry("HTTP version", "%(http.version)", res, Equal("1.0")),
		Entry("HTTP proto", "%(http.proto)", res, Equal("HTTP/1.0")),
		Entry("HTTP proto major", "%(http.protoMajor)", res, Equal("1")),
		Entry("HTTP proto minor", "%(http.protoMinor)", res, Equal("0")),
		Entry("content length", "%(contentLength)", res, Equal("80")),
		Entry("unescape carriage return", `\n`, res, Equal("\n")),
		Entry("all response headers", "%(header)", res, And(
			ContainSubstring("Content-Location: https://example.com/d"),
			ContainSubstring("X-Request-Id: 3305039"))),
		Entry("bad token", `unknown %(wut) token`, res, Equal("unknown %!(unknown: wut) token")),
		Entry("header direct name", "%(header.X-Request-ID)", res, Equal("3305039")),
		Entry("header non-canonical name", "%(header.x-request-id)", res, Equal("3305039")),
		Entry("header camel name", "%(header.xRequestId)", res, Equal("3305039")),
	)
})
