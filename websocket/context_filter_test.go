// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket_test

import (
	"context"
	"encoding/json"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/websocket"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContextFilter", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(val websocket.ContextFilter, expected string) {
			actual, _ := json.Marshal(val)
			Expect(string(actual)).To(Equal("\"" + expected + "\""))

			var o websocket.ContextFilter
			_ = json.Unmarshal(actual, &o)
			Expect(o).To(Equal(val))
			Expect(o.String()).To(Equal(expected))
		},
			Entry("Defines", websocket.Defines, "websocket.DEFINES"),
		)
	})

	Describe("Describe", func() {

		DescribeTable("examples", func(val websocket.ContextFilter, expected string) {
			actual := val.Describe()
			Expect(actual).To(Equal(expected))
		},
			Entry("Defines", websocket.Defines, "defined in joe-cli-http/websocket pkg"),
		)
	})

	Describe("Defines", func() {
		It("defines on flags set up by this package", func() {
			actual := map[string]bool{}
			app := &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{Name: "defined", Uses: websocket.SetOrigin()},
					{Name: "other", Value: new(bool)},
				},
				Action: func(c *cli.Context) {
					for _, flag := range c.Flags() {
						actual[flag.Name] = c.ContextOf(flag).Matches(websocket.Defines)
					}
				},
			}
			_ = app.RunContext(context.Background(), nil...)

			Expect(actual).To(HaveKeyWithValue("defined", true))
			Expect(actual).To(HaveKeyWithValue("other", false))
			Expect(actual).To(HaveKeyWithValue("help", false))
		})
	})
})
