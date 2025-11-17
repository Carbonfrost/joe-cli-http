// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

const someJSON = `{"error":"Character not found"}`

var _ = Describe("FetchAndPrint", func() {
	DescribeTable("examples", func(command string, expected Fields) {
		var actual *http.Request

		app := &cli.App{
			Uses: httpclient.New(
				httpclient.WithTransport(httpclient.RoundTripperFunc(func(r *http.Request) *http.Response {
					actual = r
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       io.NopCloser(strings.NewReader(someJSON)),
					}
				})),
			),
			Action: httpclient.FetchAndPrint(),
			Stdout: io.Discard,
		}
		args, _ := cli.Split(command)

		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry(
			"--body implies use of POST",
			"_ https://example.com --body body",
			Fields{
				"Method": Equal("POST"),
			},
		),
	)

	It("generates output from response", func() {
		var out bytes.Buffer

		app := &cli.App{
			Uses: httpclient.New(
				httpclient.WithTransport(httpclient.RoundTripperFunc(func(*http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       io.NopCloser(strings.NewReader(someJSON)),
					}
				})),
			),
			Action: httpclient.FetchAndPrint(),
			Stdout: &out,
		}

		args, _ := cli.Split("_ https://example.com")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(Equal(someJSON))
	})
})

var _ = Describe("Set actions", func() {

	DescribeTable("examples", func(act cli.Action, command string, transform any, expected Fields) {
		client := httpclient.New(
			httpclient.WithTransport(httpclient.RoundTripperFunc(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
				}
			})),
		)
		app := &cli.App{
			Uses:   client,
			Action: func() {},
			Stdout: io.Discard,
			Flags: []*cli.Flag{
				{
					Name: "a",
					Uses: act,
				},
			},
		}
		args, _ := cli.Split(command)

		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(httpclient.Attributes(client)).To(WithTransform(transform, PointTo(MatchFields(IgnoreExtras, expected))))
	},
		Entry(
			"SetBaseURL",
			httpclient.SetBaseURL(),
			"app -a https://example.com",
			OnClient, Fields{"BaseURL": Equal("https://example.com")},
		),
		Entry(
			"SetBody",
			httpclient.SetBody(),
			"app -a RawContent",
			OnClient, Fields{"BodyContentString": Equal("RawContent")},
		),
		Entry(
			"SetBodyContent",
			httpclient.SetBodyContent(),
			"app -a FORM_DATA",
			OnClient, Fields{"BodyContent": BeAssignableToTypeOf(&httpclient.FormDataContent{})},
		),
		Entry(
			"SetFillValue",
			httpclient.SetFillValue(),
			"app -a F=V",
			OnClient, Fields{"BodyForm": HaveKeyWithValue("F", []string{"V"})}),
		Entry(
			"SetFollowRedirects",
			httpclient.SetFollowRedirects(),
			"app -a",
			OnClient, Fields{"CheckRedirect": BeNil()}),
		Entry(
			"SetHeader",
			httpclient.SetHeader(),
			"app -a H:V",
			OnRequest, Fields{"Header": HaveKeyWithValue("H", []string{"V"})}),
		Entry(
			"SetIncludeResponseHeaders",
			httpclient.SetIncludeResponseHeaders(),
			"app -a",
			OnClient, Fields{"IncludeResponseHeaders": BeTrue()}),
		Entry(
			"SetMethod",
			httpclient.SetMethod(),
			"app -a patch",
			OnRequest, Fields{"Method": Equal("PATCH")}),
		Entry(
			"SetUserAgent",
			httpclient.SetUserAgent(),
			"app -a FI",
			OnRequest,
			Fields{"Header": HaveKeyWithValue("User-Agent", []string{"FI"})}),
		Entry(
			"SetStripComponents",
			httpclient.SetStripComponents(),
			"app -a 3",
			OnClient,
			Fields{"DownloaderWithMiddleware": Equal(httpclient.PreserveRequestPath.WithStripComponents(3))}),
	)

})

func OnClient(v *httpclient.ClientAttributes) *httpclient.ClientAttributes {
	return v
}

func OnRequest(v *httpclient.ClientAttributes) *httpclient.RequestAttributes {
	return v.Request
}

func OnTLSConfig(v *httpclient.ClientAttributes) *tls.Config {
	return v.TLSConfig
}
