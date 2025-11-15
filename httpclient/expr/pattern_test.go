// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr_test

import (
	"bytes"
	"net/url"
	"os"

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

	Describe("SyntaxRecursive", func() {

		DescribeTable("examples", func(pattern, expected string) {
			pat := expr.SyntaxRecursive.Compile(pattern)

			actual := pat.Expand(
				expr.ExpandMap(map[string]any{
					"hello":   "world",
					"goodbye": "earth",
					"foo":     "bar",
				}),
			)
			Expect(actual).To(Equal(expected), "Expected: debug pattern %s", expr.DebugPattern(pat))
		},
			Entry("nominal", "hello %(hello)", "hello world"),
			Entry("fallback to literal", "bonjour %(missing:le monde)", "bonjour le monde"),
			Entry("fallback to var", "hello %(missing:%(goodbye))", "hello earth"),
			Entry("fallback var 2", "hello %(missing:%(missing:%(foo)))", "hello bar"),
			Entry("fallback literal 2", "hello %(missing:%(missing:baz))", "hello baz"),
			Entry("redundant literal fallback", "hello %(missing:literal:bar)", "hello literal"),
		)
	})
})

var _ = Describe("Compile", func() {

	DescribeTable("expected output",
		func(pattern, expected string) {
			pat := expr.Compile(pattern)
			actual := pat.Expand(expr.ExpandMap(map[string]any{"hello": "world"}))
			Expect(actual).To(Equal(expected))
		},
		Entry("nominal", "hello %(hello)", "hello world"),
		Entry("missing value", "hello %(planet)", "hello <nil>"),
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

		Entry("whitespace", "%(newline)%(tab)%(space)", "%(newline)%(tab)%(space)"),
	)

	It("expands meta reference", func() {
		meta := expr.Compile("%(a) %(b)")
		pat := expr.Compile("%(meta) %(c)").WithMeta("meta", meta)
		Expect(pat.String()).To(Equal("%(a) %(b) %(c)"))
	})
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

var _ = Describe("ExpandEnv", func() {

	os.Setenv("ENV_VAR", "an env var")

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expr.Compile(text)

		expander := expr.Prefix("env", expr.ExpandEnv)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("os env", "%(env.ENV_VAR)", Equal("an env var")),
		Entry("os env non-existing", "%(env.ENV_VAR__NON_EXISTENT)", Equal("<nil>")),
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
