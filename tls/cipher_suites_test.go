// Copyright 2022, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tls_test

import (
	"bytes"
	"context"
	gotls "crypto/tls"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/tls"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CipherSuites", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected tls.CipherSuites) {
				cs := new(tls.CipherSuites)
				for _, a := range args {
					err := cs.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(*cs).To(Equal(expected))
			},
			Entry(
				"simple",
				[]string{"TLS_RSA_WITH_AES_128_CBC_SHA"},
				tls.CipherSuites([]uint16{gotls.TLS_RSA_WITH_AES_128_CBC_SHA}),
			),
		)
	})
})

var _ = Describe("CurveID", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected tls.CurveIDs) {
				cs := new(tls.CurveIDs)
				for _, a := range args {
					err := cs.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(*cs).To(Equal(expected))
			},
			Entry("P256", []string{"P256"}, tls.CurveIDs([]gotls.CurveID{gotls.CurveP256})),
			Entry("P384", []string{"P384"}, tls.CurveIDs([]gotls.CurveID{gotls.CurveP384})),
			Entry("P521", []string{"P521"}, tls.CurveIDs([]gotls.CurveID{gotls.CurveP521})),
			Entry("X25519", []string{"X25519"}, tls.CurveIDs([]gotls.CurveID{gotls.X25519})),
			Entry("multi", []string{"P256", "X25519"}, tls.CurveIDs([]gotls.CurveID{gotls.CurveP256, gotls.X25519})),
		)
	})
})

var _ = Describe("ListCiphers", func() {

	It("generates expected output", func() {
		var buf bytes.Buffer
		app := &cli.App{
			Stdout: &buf,
			Action: tls.ListCiphers(),
		}
		app.RunContext(context.Background(), []string{"app"})
		Expect(buf.String()).To(ContainSubstring("TLS_RSA_WITH_AES_128_CBC_SHA\tTLS 1.0, TLS 1.1, TLS 1.2"))
	})

})

var _ = Describe("ListCurves", func() {

	It("generates expected output", func() {
		var buf bytes.Buffer
		app := &cli.App{
			Stdout: &buf,
			Action: tls.ListCurves(),
		}
		app.RunContext(context.Background(), []string{"app"})
		Expect(buf.String()).To(ContainSubstring("P521"))
	})

})
