package httpclient_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/Carbonfrost/joe-cli"
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

	Context("when redirected", func() {
		DescribeTable("examples", func(start string, expr string, expected string) {
			var actual bytes.Buffer

			// TODO  Revisit dependency on CLI - This orchestration of the app is necessary
			// since the internals of http.Client currently depend upon cli.Context
			app := &cli.App{
				Uses: httpclient.New(
					httpclient.WithTransport(httpclient.RoundTripperFunc(func(req *http.Request) *http.Response {
						var num int
						_, err := fmt.Sscanf(req.URL.RequestURI(), "/redirect%d", &num)

						if err != nil {
							return &http.Response{
								StatusCode: http.StatusNotFound,
							}
						}

						location := fmt.Sprintf("/redirect%d", num-1)
						if num <= 1 {
							location = "/404"
						}
						return &http.Response{
							StatusCode: http.StatusTemporaryRedirect,
							Header: http.Header{
								"Location": []string{location},
							},
						}
					})),
				),
				Action: httpclient.FetchAndPrint(),
				Stdout: &actual,
			}
			args, _ := cli.Split(fmt.Sprintf(`app --write-out="%v" "%v"`, expr, start))

			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual.String()).To(Equal(expected))
		},
			Entry("redirect.location",
				"/redirect1", "%(redirect.location)", "/404"),
			Entry("redirect.location.path",
				"/redirect1", "%(redirect.location.path)", "/404"),
			Entry("redirect.header.referer",
				"/redirect1", "%(redirect.header.referer)", "/redirect1"),
			Entry("redirect.method",
				"/redirect1", "%(redirect.method)", "GET"),
			Entry("cascaded redirects",
				"/redirect2", "%(redirect.location)%(newline)", "/redirect1\n/404\n"),
			Entry("no redirects",
				"/404", "%(redirect.location)%(newline)", ""),
			Entry("no redirects: nested key", "/404", "%(redirect.location.path)", ""),
			Entry("no redirects: nested key", "/404", "%(redirect.header.referer)", ""),
			Entry("no redirects: blank key", "/404", "%(redirect.method)", ""),
			Entry("no redirects: blank key", "/404", "%(redirect.protocol)", ""),
			Entry("no redirects: blank key", "/404", "%(redirect.header)", ""),
			Entry("no redirects: blank key", "/404", "%(redirect.location)", ""),

			Entry("only actual request",
				"/redirect2", "%(request.url.path)%(newline)", "/redirect2\n"),
		)

	})
})
