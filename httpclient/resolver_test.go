// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resolver", func() {

	DescribeTable("secure examples", func(locations []string, expected []string) {
		r := httpclient.NewDefaultLocationResolver()
		r.SetSecure(true)
		Expect(r.Secure()).To(BeTrue())
		for _, loc := range locations {
			r.Add(loc)
		}

		actual, err := r.Resolve(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(urisToStrings(actual)).To(Equal(expected))
	},
		Entry(
			"fixup host scheme",
			[]string{"example.com"},
			[]string{"https://example.com"},
		),
		Entry(
			"fixup port",
			[]string{":8080", "hello"},
			[]string{"https://localhost:8080", "https://localhost:8080/hello"},
		),
		Entry(
			"explicit scheme unaffected",
			[]string{"http://example.com"},
			[]string{"http://example.com"},
		),
	)

	Describe("BaseURL", func() {
		It("is deferred until needed", func() {
			r := httpclient.NewDefaultLocationResolver()
			r.SetBaseURL("example.com")
			Expect(r.BaseURL()).To(Equal("http://example.com"))

			r.SetSecure(true)
			Expect(r.BaseURL()).To(Equal("https://example.com"))
		})

		It("resolves successive bases", func() {
			r := httpclient.NewDefaultLocationResolver()
			r.SetBaseURL("https://example.com/a/")
			r.SetBaseURL("b")
			Expect(r.BaseURL()).To(Equal("https://example.com/a/b"))
		})
	})

	DescribeTable("examples", func(locations []string, expected []string) {
		r := httpclient.NewDefaultLocationResolver()
		for _, loc := range locations {
			r.Add(loc)
		}

		actual, err := r.Resolve(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(urisToStrings(actual)).To(Equal(expected))
	},
		Entry(
			"fixup host scheme",
			[]string{"example.com"},
			[]string{"http://example.com"},
		),
		Entry(
			"resolve relative",
			[]string{"https://example.com", "hello"},
			[]string{"https://example.com", "https://example.com/hello"},
		),
		Entry(
			"fixup host scheme and relative",
			[]string{"example.com", "hello"},
			[]string{"http://example.com", "http://example.com/hello"},
		),
	)
})

func urisToStrings(results []httpclient.Location) []string {
	res := make([]string, len(results))
	for i, item := range results {
		_, u, _ := item.URL(context.Background())
		res[i] = u.String()
	}
	return res
}
