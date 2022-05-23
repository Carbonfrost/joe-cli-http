package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DownloadMode", func() {

	Describe("FileName", func() {
		DescribeTable("examples",
			func(mode httpclient.DownloadMode, u string, expected string) {
				request, _ := http.NewRequest("GET", u, nil)
				cs := mode.FileName(&httpclient.Response{
					Response: &http.Response{
						Request: request,
					},
				})
				Expect(cs).To(Equal(expected))
			},
			Entry("empty", httpclient.PreserveRequestFile, "https://example.com/", ""),
			Entry("simple", httpclient.PreserveRequestFile, "https://example.com/hello", "hello"),
			Entry("query string", httpclient.PreserveRequestFile, "https://example.com/hello?a=b", "hello?a=b"),

			Entry("empty", httpclient.PreserveRequestPath, "https://example.com/", ""),
			Entry("simple", httpclient.PreserveRequestPath, "https://example.com/hello/world", "hello/world"),
			Entry("query string", httpclient.PreserveRequestPath, "https://example.com/hello/world?a=b", "hello/world?a=b"),
		)
	})
})
