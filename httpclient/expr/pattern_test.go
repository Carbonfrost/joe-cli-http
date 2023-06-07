package expr_test

import (
	"bytes"
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli-http/internal/build"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("CompilePattern", func() {

	DescribeTable("expected output",
		func(start, end, pattern, expected string) {
			pat := expr.CompilePattern(pattern, start, end)
			actual := pat.Expand(expr.ExpandMap(map[string]any{"hello": "world"}))
			Expect(actual).To(Equal(expected))
		},
		Entry("quote with percent sign", "%(", ")", "hello %(hello)", "hello world"),
	)
})

var _ = Describe("Compile", func() {

	DescribeTable("expected output",
		func(pattern, expected string) {
			pat := expr.Compile(pattern)
			actual := pat.Expand(expr.ExpandMap(map[string]any{"hello": "world"}))
			Expect(actual).To(Equal(expected))
		},
		Entry("nominal", "hello %(hello)", "hello world"),
		Entry("missing value", "hello %(planet)", "hello %!(unknown: planet)"),
		Entry("whitespace: empty expansion newline", "%(empty)%(newline)", "\n"),
		Entry("whitespace: nominal expansion newline", "%(hello)%(newline)", "world\n"),
		Entry("whitespace: nominal multiple newlines", "%(hello)%(newline)%(newline)", "world\n\n"),
		Entry("whitespace: literal expansion newline", "literal%(newline)", "literal\n"),
		Entry("whitespace: literal multiple newlines", "literal%(newline)%(newline)", "literal\n\n"),

		// Starting with these whitespace tokens treats as if a literal
		Entry("whitespace: empty literal newline", "%(newline)", "\n"),
		Entry("whitespace: adjacent newlines", "%(newline)%(newline)", "\n\n"),
	)

	Context("when using colors", func() {
		DescribeTable("example",
			func(pattern, expected string) {
				pat := expr.Compile(pattern)
				actual := pat.Expand(expr.Prefix("color", expr.ExpandColors))
				Expect(actual).To(Equal(expected))
			},
			Entry("yellow", "%(color.yellow)", "\x1b[33m"),
		)
	})

	Context("when using renderer", func() {
		DescribeTable("examples",
			func(pattern, expectedOut, expectedErr string) {
				var out, err bytes.Buffer

				pat := expr.Compile(pattern)
				renderer := expr.NewRenderer(&out, &err)
				_, _ = expr.Fprint(renderer, pat, expr.ExpandMap(map[string]any{"hello": "world"}))
				Expect(out.String()).To(Equal(expectedOut))
				Expect(err.String()).To(Equal(expectedErr))
			},
			Entry("default to out", "abc", "abc", ""),
			Entry("switch to err from start", "%(stderr)abc", "", "abc"),
			Entry("switch to err", "abc%(stderr)xyz", "abc", "xyz"),
			Entry("switch back to out", "abc%(stderr)xyz%(stdout)bar", "abcbar", "xyz"),
		)
	})
})

var _ = Describe("String", func() {

	DescribeTable("examples",
		func(pattern, expected string) {
			pat := expr.Compile(pattern)
			Expect(pat.String()).To(Equal(expected))
		},
		Entry("literal", "hello", "hello"),
		Entry("expansion", "hello %(planet)", "hello %(planet)"),
		Entry("untruncated expansion", "hello %(p", "hello %(p"),
		Entry("default access log", "%(accessLog.default)", `- - [%(start:02/Jan/2006 15:04:05)] "%(method) %(urlPath) %(protocol)" %(statusCode) -`),
		Entry("whitespace", "%(newline)%(tab)%(space)", "%(newline)%(tab)%(space)"),
	)
})

var _ = Describe("ExpandURL", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		u, _ := url.Parse("https://me:password@example.com/whistle?query=1#fragment")
		e := expr.Compile(text)

		expander := expr.Prefix("url", expr.ExpandURL(u))
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("scheme", "%(url.scheme)", Equal("https")),
		Entry("authority", "%(url.authority)", Equal("me:password@example.com")),
		Entry("query", "%(url.query)", Equal("query=1")),
		Entry("userInfo", "%(url.userInfo)", Equal("me:password")),
		Entry("user", "%(url.user)", Equal("me")),
		Entry("host", "%(url.host)", Equal("example.com")),
		Entry("path", "%(url.path)", Equal("/whistle")),
		Entry("requestURI", "%(url.requestURI)", Equal("/whistle?query=1")),
		Entry("fragment", "%(url.fragment)", Equal("fragment")),
	)
})

var _ = Describe("ExpandGlobals", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expr.Compile(text)
		expander := expr.ExpandGlobals
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("go version", "%(go.version)", MatchRegexp(`go\d(\.\d+)+`)),
		Entry("wig version", "%(wig.version)", Equal(build.Version)),
		Entry("time", "%(time.now)", MatchRegexp(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)),
		Entry("time (alias)", "%(time)", MatchRegexp(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)),
		Entry("time UTC", "%(time.now.utc)", MatchRegexp(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)), // ends with Z
		Entry("random", "%(random)", MatchRegexp(`\d+`)),
		Entry("random.float", "%(random.float)", MatchRegexp(`\d(\.\d+)`)),
	)
})
