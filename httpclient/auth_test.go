// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient_test

import (
	"encoding/json"
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("AuthMode", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(opt httpclient.AuthMode, expected string) {
			actual, err := json.Marshal(opt)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal("\"" + expected + "\""))

			var o httpclient.AuthMode
			err = json.Unmarshal(actual, &o)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(opt))
		},
			Entry("NoAuth", httpclient.NoAuth, "NO_AUTH"),
			Entry("Basic", httpclient.BasicAuth, "BASIC"),
		)
	})

	Describe("String", func() {

		DescribeTable("examples", func(opt httpclient.AuthMode, expected string) {
			Expect(opt.String()).To(Equal(expected))
		},
			Entry("NoAuth", httpclient.NoAuth, "none"),
			Entry("Basic", httpclient.BasicAuth, "basic"),
		)
	})
})

var _ = Describe("NewAuthenticator", func() {

	DescribeTable("examples", func(name string, v any, requiresUserInfo bool) {
		actual, err := httpclient.NewAuthenticator(name, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(v))
		Expect(actual.RequiresUserInfo()).To(Equal(requiresUserInfo))
	},
		Entry("basic", "basic", httpclient.BasicAuth, true),
		Entry("blank", "", httpclient.NoAuth, false),
		Entry("BASIC", "BASIC", httpclient.BasicAuth, true),
		Entry("none", "none", httpclient.NoAuth, false),
	)
})

var _ = Describe("NewBearerAuthenticator", func() {

	DescribeTable("examples", func(headers string, expected types.GomegaMatcher) {
		auth := httpclient.NewBearerTokenAuthenticator("TOKEN", headers)
		r, _ := http.NewRequest("GET", "https://example.com", nil)
		auth.Authenticate(r, nil)
		Expect(r.Header).To(expected)
	},
		Entry("empty", "", HaveKeyWithValue("Authentication", []string{"Bearer TOKEN"})),
		Entry("Authentication", "Authentication", HaveKeyWithValue("Authentication", []string{"Bearer TOKEN"})),
		Entry("Custom", "X-Token", HaveKeyWithValue("X-Token", []string{"TOKEN"})),
		Entry("Custom plus value", "X-Token Bearer", HaveKeyWithValue("X-Token", []string{"Bearer TOKEN"})),
		Entry("Custom plus values 2", "X-Token Bearer A", HaveKeyWithValue("X-Token", []string{"Bearer A TOKEN"})),
	)
})
