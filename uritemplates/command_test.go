// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates_test

import (
	"context"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Expand", func() {

	DescribeTable("examples", func(args string, expected string) {
		var capture strings.Builder
		captureOutput := func(c *cli.Context) {
			w := cli.NewWriter(&capture)
			w.SetColorCapable(false)
			c.Stdout = w
			c.Stderr = w
		}
		app := cli.NewApp(&cli.Command{
			Uses: cli.Pipeline(
				uritemplates.FlagsAndArgs(),
				uritemplates.Expand(),
				captureOutput,
			)})

		arguments, _ := cli.Split(args)
		app.RunContext(context.Background(), arguments)
		Expect(strings.TrimSpace(capture.String())).To(Equal(expected))
	},
		Entry(
			"url",
			"expand https://example.com",
			"https://example.com",
		),
		Entry(
			"nominal",
			"expand '{+baseURL}/{var}' -TbaseURL=https://example.com -Tvar=f",
			"https://example.com/f",
		),
		Entry(
			"array",
			"expand 'https://example.com{/path*}{?query}' -Tarray,path=a -Tarray,path=b",
			"https://example.com/a/b",
		),
		XEntry(
			"var overflow to query string",
			"expand https://example.com -Ts hello=world",
			"https://example.com?hello=world",
		),
	)
})

var _ = Describe("Set actions", func() {

	It("has Source annotation", func() {
		app := &cli.App{
			Uses: uritemplates.FlagsAndArgs(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("params")
		Expect(f.Data).To(HaveKeyWithValue("Source", "github.com/Carbonfrost/joe-cli-http/uritemplates"))
	})

})
