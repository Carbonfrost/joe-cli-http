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

const someJson = `{"error":"Character not found"}`

var _ = Describe("FetchAndPrint", func() {
	DescribeTable("examples", func(command string, expected Fields) {
		var actual *http.Request

		app := &cli.App{
			Uses: httpclient.New(
				httpclient.WithTransport(httpclient.RoundTripperFunc(func(r *http.Request) *http.Response {
					actual = r
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       io.NopCloser(strings.NewReader(someJson)),
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
						Body:       io.NopCloser(strings.NewReader(someJson)),
					}
				})),
			),
			Action: httpclient.FetchAndPrint(),
			Stdout: &out,
		}

		args, _ := cli.Split("_ https://example.com")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(Equal(someJson))
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
			OnClient,
			Fields{"BaseURL": Equal("https://example.com")},
		),
		Entry(
			"SetBody",
			httpclient.SetBody(),
			"app -a RawContent",
			OnClient,
			Fields{"BodyContentString": Equal("RawContent")},
		),
		Entry(
			"SetBodyContent",
			httpclient.SetBodyContent(),
			"app -a FORM_DATA",
			OnClient,
			Fields{"BodyContent": BeAssignableToTypeOf(&httpclient.FormDataContent{})},
		),
	)

})

func OnClient(v *httpclient.ClientAttributes) *httpclient.ClientAttributes {
	return v
}
func OnTLSConfig(v *httpclient.ClientAttributes) *tls.Config {
	return v.TLSConfig
}
