// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mop_test

import (
	"bytes"
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/cmd/mop"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mop", func() {

	It("registers the websocket flags", func() {
		app := mop.NewApp()
		app.Initialize(context.Background())

		f, ok := app.Flag("protocol")
		Expect(ok).To(BeTrue())
		Expect(f.Data).To(HaveKeyWithValue("Source", "github.com/Carbonfrost/joe-cli-http/websocket"))
	})

	It("displays the help screen when no URL is specified", func() {
		var out bytes.Buffer

		app := mop.NewApp()
		app.Stdout = &out
		app.Stderr = &out

		args, _ := cli.Split("mop")

		// The help screen exits with the usage status
		Expect(app.RunContext(context.Background(), args)).To(MatchError("exited with status 2"))
		Expect(out.String()).To(ContainSubstring("usage: mop"))
	})
})
