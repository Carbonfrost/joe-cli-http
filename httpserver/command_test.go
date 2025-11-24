// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpserver_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Set actions", func() {

	It("has Source annotation", func() {
		app := &cli.App{
			Uses: httpserver.FlagsAndArgs(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("key")
		Expect(f.Data).To(HaveKeyWithValue("Source", "github.com/Carbonfrost/joe-cli-http/httpserver"))
	})

})
