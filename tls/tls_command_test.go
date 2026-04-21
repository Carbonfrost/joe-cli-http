// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tls_test

import (
	"context"
	gotls "crypto/tls"
	"io"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Set actions", func() {

	It("has Source annotation", func() {
		app := &cli.App{
			Uses: tls.New(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("--list-curves")
		Expect(f.Data).To(HaveKeyWithValue("Source", "github.com/Carbonfrost/joe-cli-http/tls"))
	})

	DescribeTable("examples", func(act cli.Action, command string, expected Fields) {
		config := tls.New()
		// Override default action so no flags are registered; only placein the context
		config.Action = tls.ContextValue(config)
		app := &cli.App{
			Uses:   config,
			Action: func() {},
			Stdout: io.Discard,
			Flags: []*cli.Flag{
				{
					Name: "a",
					Uses: act,
				},
			},
		}
		args, _ := cli.Split(command)

		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Config).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},

		Entry(
			"SetCiphers",
			tls.SetCiphers(),
			"app -a TLS_AES_128_GCM_SHA256",
			Fields{
				"CipherSuites": Equal([]uint16{gotls.TLS_AES_128_GCM_SHA256}),
			}),

		Entry(
			"SetCurves",
			tls.SetCurves(),
			"app -a P521",
			Fields{"CurvePreferences": Equal([]gotls.CurveID{gotls.CurveP521})}),

		Entry(
			"SetInsecureSkipVerify",
			tls.SetInsecureSkipVerify(),
			"app -a",
			Fields{"InsecureSkipVerify": BeTrue()}),

		Entry(
			"SetNextProtos",
			tls.SetNextProtos(),
			"app -a F -a G",
			Fields{"NextProtos": Equal([]string{"F", "G"})}),

		Entry(
			"SetServerName",
			tls.SetServerName(),
			"app -a dot.example",
			Fields{"ServerName": Equal("dot.example")}),

		Entry(
			"SetTLSv1",
			tls.SetTLSv1(),
			"app -a",
			Fields{"MinVersion": Equal(uint16(gotls.VersionTLS10)), "MaxVersion": Equal(uint16(gotls.VersionTLS13))}),

		Entry(
			"SetTLSv1_0",
			tls.SetTLSv1_0(),
			"app -a",
			Fields{"MinVersion": Equal(uint16(gotls.VersionTLS10)), "MaxVersion": Equal(uint16(gotls.VersionTLS10))}),

		Entry(
			"SetTLSv1_1",
			tls.SetTLSv1_1(),
			"app -a",
			Fields{"MinVersion": Equal(uint16(gotls.VersionTLS11)), "MaxVersion": Equal(uint16(gotls.VersionTLS11))}),

		Entry(
			"SetTLSv1_2",
			tls.SetTLSv1_2(),
			"app -a",
			Fields{"MinVersion": Equal(uint16(gotls.VersionTLS12)), "MaxVersion": Equal(uint16(gotls.VersionTLS12))}),

		Entry(
			"SetTLSv1_3",
			tls.SetTLSv1_3(),
			"app -a",
			Fields{"MinVersion": Equal(uint16(gotls.VersionTLS13)), "MaxVersion": Equal(uint16(gotls.VersionTLS13))}),
	)

})
