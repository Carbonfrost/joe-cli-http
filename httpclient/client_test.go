// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {

	Describe("AddRequestHeader", func() {
		It("aggregates header values", func() {
			s := httpclient.New()
			s.Apply(
				httpclient.AddRequestHeader(&httpclient.HeaderValue{"Link", "Something"}),
				httpclient.AddRequestHeader(&httpclient.HeaderValue{"Link", "SomethingElse"}),
			)

			Expect(s.Request.Header).To(HaveKeyWithValue("Link", []string{"Something", "SomethingElse"}))
		})
	})

	Describe("WithBodyContentString", func() {
		It("sets raw body value", func() {
			s := httpclient.New()
			s.Apply(httpclient.WithBodyContentString("raw content"))

			body, _ := io.ReadAll(s.BodyContent.Read())
			Expect(string(body)).To(Equal("raw content"))
		})
	})

	Describe("New", func() {
		It("sets up the default user agent string", func() {
			s := httpclient.New()
			expected := "Go-http-client/1.1 (joe-cli-http/(devel), +https://github.com/Carbonfrost/joe-cli-http)"
			Expect(s.Request.Header).To(HaveKeyWithValue("User-Agent", []string{expected}))
		})
	})
})
