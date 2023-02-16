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
		httpclient.NewRequestIDMiddleware(v).Handle(req)

		Expect(req.Header.Get("X-Request-ID")).To(expected)
	},
		Entry("string", "static string", Equal("static string")),
		Entry("string func", func() string { return "s" }, Equal("s")),
		Entry("func", func(context.Context) (string, error) { return "s", nil }, Equal("s")),
		Entry("nil", nil, HaveLen(16)),
	)

	It("uses default on unspecified arg", func() {
		req, _ := http.NewRequest("GET", "https://example.com", nil)
		httpclient.NewRequestIDMiddleware().Handle(req)

		Expect(req.Header.Get("X-Request-ID")).To(HaveLen(16))
	})

	It("panics on unknown type", func() {
		v := 69

		Expect(func() {
			httpclient.NewRequestIDMiddleware(v)
		}).To(Panic())
	})
})
