// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient_test

import (
	"encoding/json"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContentType", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(opt httpclient.ContentType, expected string) {
			actual, _ := json.Marshal(opt)
			Expect(string(actual)).To(Equal("\"" + expected + "\""))

			var o httpclient.ContentType
			_ = json.Unmarshal(actual, &o)
			Expect(o).To(Equal(opt))
		},
			Entry("FormData", httpclient.ContentTypeFormData, "FORM_DATA"),
			Entry("Raw", httpclient.ContentTypeRaw, "RAW"),
			Entry("URLEncodedFormData", httpclient.ContentTypeURLEncodedFormData, "URL_ENCODED_FORM_DATA"),
			Entry("MultipartFormData", httpclient.ContentTypeMultipartFormData, "MULTIPART_FORM_DATA"),
			Entry("JSON", httpclient.ContentTypeJSON, "JSON"),
		)
	})

	Describe("Set", func() {

		DescribeTable("examples", func(arg string, expected httpclient.ContentType) {
			var actual httpclient.ContentType = -1
			err := cli.Set(&actual, arg)

			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("FormData", "form", httpclient.ContentTypeFormData),
			Entry("Raw", "raw", httpclient.ContentTypeRaw),
			Entry("URLEncodedFormData", "urlencoded", httpclient.ContentTypeURLEncodedFormData),
			Entry("MultipartFormData", "multipart", httpclient.ContentTypeMultipartFormData),
			Entry("JSON", "json", httpclient.ContentTypeJSON),

			Entry("empty string", "", httpclient.ContentType(-1)),
			Entry("FormData marshal", "FORM_DATA", httpclient.ContentTypeFormData),
			Entry("Raw marshal", "RAW", httpclient.ContentTypeRaw),
			Entry("URLEncodedFormData marshal", "URL_ENCODED_FORM_DATA", httpclient.ContentTypeURLEncodedFormData),
			Entry("MultipartFormData marshal", "MULTIPART_FORM_DATA", httpclient.ContentTypeMultipartFormData),
			Entry("JSON marshal", "JSON", httpclient.ContentTypeJSON),
		)
	})
})
