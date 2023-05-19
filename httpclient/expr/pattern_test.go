package expr_test

import (
	"bytes"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	)
})
