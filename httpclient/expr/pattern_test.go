// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr_test

import (
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ExpandURL", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		u, _ := url.Parse("https://me:password@example.com/whistle?query=1#fragment")
		e := expander.Compile(text)

		expander := expander.Prefix("url", expr.ExpandURL(u))
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
		e := expander.Compile(text)
		exp := expander.Func(expr.ExpandGlobals)
		Expect(e.Expand(exp)).To(expected)
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
