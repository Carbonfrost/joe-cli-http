// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Content", func() {

	Describe("NewContent", func() {

		DescribeTable("examples", func(arg httpclient.ContentType, expected any) {
			actual := httpclient.NewContent(arg)
			Expect(actual).To(BeAssignableToTypeOf(expected))
		},
			Entry("Raw", httpclient.ContentTypeRaw, &httpclient.RawContent{}),
			Entry("FormData", httpclient.ContentTypeFormData, &httpclient.FormDataContent{}),
			Entry("JSON", httpclient.ContentTypeJSON, &httpclient.JSONContent{}),
			Entry("MultipartFormData", httpclient.ContentTypeMultipartFormData, &httpclient.MultipartFormDataContent{}),
			Entry("URLEncodedFormData", httpclient.ContentTypeURLEncodedFormData, &httpclient.URLEncodedFormDataContent{}),
		)
	})
})
