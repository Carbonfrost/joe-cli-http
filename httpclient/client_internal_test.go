// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {

	Describe("RedactHeader", func() {

		DescribeTable("examples", func(name, value string, expected OmegaMatcher) {
			Expect(redactHeader(name, value)).To(expected)
		},
			Entry(
				"API Key low entropy",
				"Api-Key",
				"Private",
				Equal("********"),
			),
			Entry(
				"Bearer low entropy",
				"Authorization",
				"Bearer XXX",
				Equal("Bearer ********"),
			),
			Entry("Bearer 8 characters",
				"Authorization",
				"Bearer 12345678",
				Equal("Bearer ********"),
			),
			Entry("Bearer medium entropy",
				"Authorization",
				"Bearer 1234567890",
				Equal("Bearer ********90"),
			),
			Entry("Bearer 16 characters",
				"Authorization",
				"Bearer 1234567890123456",
				Equal("Bearer **************56"),
			),
			Entry("Bearer high entropy",
				"Authorization",
				"Bearer very_long_bearer_token_with_sufficent_representation",
				Equal("Bearer very**********************************************on"),
			),
		)
	})
})
