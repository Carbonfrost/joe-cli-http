package expr_test

import (
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
				actual := pat.Expand(expr.ExpandColors)
				Expect(actual).To(Equal(expected))
			},
			Entry("yellow", "%(color.yellow)", "\x1b[33m"),
		)
	})
})
