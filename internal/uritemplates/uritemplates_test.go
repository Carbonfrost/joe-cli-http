package uritemplates_test

import (
	"github.com/Carbonfrost/joe-cli-http/internal/uritemplates"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Names", func() {

	DescribeTable("examples", func(template string, expected []string) {
		tpl, _ := uritemplates.Parse(template)
		names := tpl.Names()
		Expect(names).To(Equal(expected))
	},
		Entry("struct",
			"{/path*,Version}{?opts*}",
			[]string{"path", "Version", "opts"},
		), Entry("Pointer to struct:",
			"{?opts*}",
			[]string{"opts"},
		), Entry("Map expansion cannot be truncated:",
			"{?opts:3}",
			[]string{"opts"},
		), Entry("Map whose values are not all strings:",
			"{?one*}",
			[]string{"one"},
		), Entry("Value of inappropriate type:",
			"{?one*}",
			[]string{"one"},
		), Entry("Truncated array whose values are not all strings:",
			"{?one:3}",
			[]string{"one"},
		),
	)
})

var _ = Describe("Expand", func() {

	DescribeTable("examples", func(v any, template string, expected string) {
		tpl, _ := uritemplates.Parse(template)
		expanded, _ := tpl.Expand(v)
		Expect(expanded).To(Equal(expected))
	},
		Entry("struct",
			Location{
				Path:    []any{"main", "quux"},
				Version: 2,
				Opts: Options{
					Format: "pdf",
				},
			},
			"{/path*,Version}{?opts*}",
			"/main/quux/2?fmt=pdf",
		), Entry("Pointer to struct:",
			&Location{Opts: Options{Format: "pdf"}},
			"{?opts*}",
			"?fmt=pdf",
		), Entry("Map expansion cannot be truncated:",
			Location{Opts: Options{Format: "pdf"}},
			"{?opts:3}",
			"",
		), Entry("Map whose values are not all strings:",
			map[string]any{
				"one": map[string]any{
					"two": 42,
				},
			},
			"{?one*}",
			"?two=42",
		), Entry("Value of inappropriate type:",
			42,
			"{?one*}",
			"",
		), Entry("Truncated array whose values are not all strings:",
			map[string]any{"one": []any{1234}},
			"{?one:3}",
			"?one=123",
		),
	)

	DescribeTable("errors", func(template string, expected types.GomegaMatcher) {
		_, err := uritemplates.Parse(template)
		Expect(err).To(expected)
	},
		Entry("too many colons", "{opts:1:2}", HaveOccurred()),
	)
})

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

type Location struct {
	Path    []any   `uri:"path"`
	Version int     `json:"version"`
	Opts    Options `opts`
}

type Options struct {
	Format string `uri:"fmt"`
}
