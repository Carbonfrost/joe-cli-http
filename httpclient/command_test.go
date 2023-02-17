package httpclient_test

import (
	"bytes"
	"context"
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
