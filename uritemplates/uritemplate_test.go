// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates_test

import (
	"encoding/json"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("URITemplate", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(s string, expected string) {
			uri, _ := uritemplates.Parse(s)
			actual, _ := json.Marshal(uri)
			Expect(string(actual)).To(Equal(expected))

			var o *uritemplates.URITemplate
			_ = json.Unmarshal(actual, &o)
			Expect(o.String()).To(Equal(uri.String()))
		},
			Entry("nominal", "https://localhost", `"https://localhost"`),
			Entry("var", "https://{host}{/path}{resource}.json{?q}", `"https://{host}{/path}{resource}.json{?q}"`),
		)
	})
})
