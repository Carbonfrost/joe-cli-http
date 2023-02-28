package uritemplates_test

import (
	"github.com/Carbonfrost/joe-cli-http/internal/uritemplates"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UriTemplate", func() {

	Describe("PartialExpand", func() {
		DescribeTable("examples", func(t string, vars any, expected string) {
			u, _ := uritemplates.Parse(t)
			actual, _ := u.PartialExpand(vars)
			Expect(actual).To(Equal(expected))
		},
			Entry(
				"full expand", "{scheme}://{.domain*}",
				map[string]any{"scheme": "https", "domain": []any{"app", "example", "com"}},
				"https://.app.example.com"),
			Entry(
				"nominal", "{scheme}://{.domain}",
				map[string]any{"scheme": "https"},
				"https://{.domain}"),
			Entry(
				"escapes", "{scheme}://example.com{/path}{?query}",
				map[string]any{"scheme": "https"},
				"https://example.com{/path}{?query}"),
			Entry(
				"query partial fill", "https://example.com{?a,b}",
				map[string]any{"a": "a"},
				"https://example.com?a=a{&b}"),
			Entry(
				"explode partial fill", "https://example.com{?query*}",
				map[string]any{},
				"https://example.com{?query*}"),
		)
	})

})
