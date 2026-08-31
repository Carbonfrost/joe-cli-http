// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	joeclifakes "github.com/Carbonfrost/joe-cli-http/internal/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {

	Describe("Addr", func() {

		DescribeTable("examples", func(opts []httpserver.Option, expected string) {
			s := httpserver.New()
			s.Apply(opts...)

			Expect(s.Addr()).To(Equal(expected))
		},

			Entry("default", nil, "localhost:8000"),
			Entry("host",
				[]httpserver.Option{
					httpserver.WithHostname("elvis.localhost"),
				}, "elvis.localhost:8000"),
			Entry("port",
				[]httpserver.Option{
					httpserver.WithPort(1619),
				}, "localhost:1619"),
			Entry("host and port",
				[]httpserver.Option{
					httpserver.WithHostname("elvis.localhost"),
					httpserver.WithPort(1619),
				}, "elvis.localhost:1619"),
			Entry("addr",
				[]httpserver.Option{
					httpserver.WithAddr("elvis.localhost:8900"),
				}, "elvis.localhost:8900"),
		)
	})

	Describe("RunServer", func() {

		It("runs the actions before server", func() {
			fakeAct := new(joeclifakes.FakeAction)
			app := &cli.App{
				Uses: cli.Pipeline(
					httpserver.New(
						httpserver.WithPort(-1),
						httpserver.WithReadyFunc(func(c context.Context) {
							httpserver.FromContext(c).Shutdown(c)
						}),
					),
					httpserver.RunServer(fakeAct),
				),
			}

			_ = app.RunContext(context.Background(), nil)
			Expect(fakeAct.ExecuteCallCount()).To(Equal(1))
		})
	})
})
