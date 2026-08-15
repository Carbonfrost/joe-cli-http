// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver_test

import (
	"context"
	"io"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Set actions", func() {

	DescribeTable("examples", func(act cli.Action, command string, transform any, expected Fields) {
		server := httpserver.New()
		app := &cli.App{
			Uses: httpserver.New(
				// Override default action so no flags are registered; only place client
				// in the context
				httpserver.WithAction(
					httpserver.ContextValue(server),
				),
			),
			Action: func() {},
			Stdout: io.Discard,
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: act,
				},
			},
		}
		args, _ := cli.Split(command)

		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(httpserver.Attributes(server)).To(WithTransform(transform, PointTo(MatchFields(IgnoreExtras, expected))))
	},
		Entry(
			"SetAccessLog",
			httpserver.SetAccessLog(),
			"app --flag x",
			OnServer, Fields{"AccessLog": Equal("x")},
		),
		XEntry( // TODO Requires joe-cli@futures where No/OptionalValue is viable
			"SetNoAccessLog",
			httpserver.SetAccessLog(),
			"app --no-flag",
			OnServer, Fields{"AccessLog": Equal("")},
		),
	)
})

func OnServer(v *httpserver.ServerAttributes) *httpserver.ServerAttributes {
	return v
}
