// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"

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

			Expect(newRequest(s).Header).To(HaveKeyWithValue("Link", []string{"Something", "SomethingElse"}))
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
			Expect(newRequest(s).Header).To(HaveKeyWithValue("User-Agent", []string{expected}))
		})
	})

	Describe("NewRequest", func() {
		It("applies the configured method to the request", func() {
			s := httpclient.New(httpclient.WithRequestMethod("patch"))

			Expect(newRequest(s).Method).To(Equal("PATCH"))
		})

		It("caches the request", func() {
			s := httpclient.New()

			Expect(newRequest(s)).To(BeIdenticalTo(newRequest(s)))
		})

		It("applies the configured values to the request from WithRequest", func() {
			s := httpclient.New(
				httpclient.WithRequest(&http.Request{Method: http.MethodPut, Close: true}),
				httpclient.AddRequestHeader(&httpclient.HeaderValue{"Link", "Something"}),
			)

			Expect(newRequest(s)).To(HaveField("Method", Equal(http.MethodPut)))
			Expect(newRequest(s)).To(HaveField("Close", BeTrue()))
			Expect(newRequest(s).Header).To(HaveKeyWithValue("Link", []string{"Something"}))
		})
	})

	Describe("NewDownloader", func() {
		It("downloads to the context stdout by default", func() {
			var buf bytes.Buffer
			s := httpclient.New()

			output, err := newDownloader(s, &buf).OpenDownload(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())

			fmt.Fprint(output, "hello")
			Expect(buf.String()).To(Equal("hello"))
		})

		It("caches the downloader", func() {
			var buf bytes.Buffer
			s := httpclient.New()

			Expect(newDownloader(s, &buf)).To(BeIdenticalTo(newDownloader(s, &buf)))
		})

		It("uses the downloader from WithDownloaderFactory", func() {
			var buf bytes.Buffer
			expected := httpclient.NewDownloaderTo(io.Discard)
			s := httpclient.New(httpclient.WithDownloaderFactory(
				func(context.Context) (httpclient.Downloader, error) {
					return expected, nil
				}))

			Expect(newDownloader(s, &buf)).To(BeIdenticalTo(expected))
		})

		It("prefers the downloader from WithDownloadFile over the factory", func() {
			var buf bytes.Buffer
			expected := httpclient.NewDownloaderTo(io.Discard)
			s := httpclient.New(
				httpclient.WithDownloaderFactory(func(context.Context) (httpclient.Downloader, error) {
					return httpclient.NewDownloaderTo(&buf), nil
				}),
				httpclient.WithDownloadFile(expected),
			)

			Expect(newDownloader(s, &buf)).To(BeIdenticalTo(expected))
		})

		It("applies middleware to the downloader", func() {
			var actual httpclient.Downloader
			expected := httpclient.NewDownloaderTo(io.Discard)
			var buf bytes.Buffer

			s := httpclient.New(
				httpclient.WithDownloadFile(expected),
				httpclient.WithDownloaderMiddleware(func(_ context.Context, d httpclient.Downloader) httpclient.Downloader {
					actual = d
					return httpclient.NewDownloaderTo(&buf)
				}),
			)

			Expect(newDownloader(s, &buf)).NotTo(BeIdenticalTo(expected))
			Expect(actual).To(BeIdenticalTo(expected))
		})

		It("reports the error from the factory", func() {
			var buf bytes.Buffer
			s := httpclient.New(httpclient.WithDownloaderFactory(
				func(context.Context) (httpclient.Downloader, error) {
					return nil, errors.New("not today")
				}))

			_, err := s.NewDownloader(&cli.Context{Stdout: cli.NewWriter(&buf)})
			Expect(err).To(MatchError("not today"))
		})
	})

	Describe("NewAuthenticator", func() {
		It("defaults to NoAuth", func() {
			s := httpclient.New()
			auth, err := s.NewAuthenticator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(auth).To(Equal(httpclient.NoAuth))
		})

		It("uses the authenticator from WithAuth", func() {
			s := httpclient.New(httpclient.WithAuth(httpclient.BasicAuth))
			auth, _ := s.NewAuthenticator(context.Background())

			Expect(auth).To(Equal(httpclient.BasicAuth))
		})

		It("uses the authenticator from WithAuthenticatorFactory", func() {
			expected := httpclient.NewBearerTokenAuthenticator("TOKEN")
			s := httpclient.New(httpclient.WithAuthenticatorFactory(func(context.Context) (httpclient.Authenticator, error) {
				return expected, nil
			}))
			auth, _ := s.NewAuthenticator(context.Background())

			Expect(auth).To(BeIdenticalTo(expected))
		})

		It("applies the authenticator middleware and caches the result", func() {
			var calls int
			s := httpclient.New(
				httpclient.WithAuth(httpclient.BasicAuth),
				httpclient.WithAuthenticatorMiddleware(func(ctx context.Context, a httpclient.Authenticator) httpclient.Authenticator {
					calls++
					return httpclient.WithPromptForCredentials(ctx, a)
				}),
			)
			first, _ := s.NewAuthenticator(context.Background())
			second, _ := s.NewAuthenticator(context.Background())

			Expect(first).NotTo(Equal(httpclient.BasicAuth))
			Expect(first).To(BeIdenticalTo(second))
			Expect(calls).To(Equal(1))
		})
	})
})

var _ = Describe("Do", func() {

	It("processes middleware on a copy of the request per location", func() {
		var actual []*http.Request

		u, _ := url.Parse("https://example.com/a")
		v, _ := url.Parse("https://example.com/b")

		client := httpclient.New(
			httpclient.WithTransport(httpclient.RoundTripperFunc(func(r *http.Request) *http.Response {
				actual = append(actual, r)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
				}
			})),
			httpclient.WithURL(u),
			httpclient.WithURL(v),
			httpclient.WithQueryString(&cli.NameValue{Name: "q", Value: "1"}),
			httpclient.WithRequestID(),
		)
		app := &cli.App{
			Uses:   client,
			Action: httpclient.FetchAndPrint(),
			Stdout: io.Discard,
		}

		err := app.RunContext(context.Background(), "_")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(HaveLen(2))

		// Each location has its own URL and query string rather than
		// accumulating the values from the previous one
		Expect(actual[0].URL.String()).To(Equal("https://example.com/a?q=1"))
		Expect(actual[1].URL.String()).To(Equal("https://example.com/b?q=1"))

		// Middleware runs for each request, so each has its own request ID
		Expect(actual[0].Header.Get("X-Request-Id")).NotTo(BeEmpty())
		Expect(actual[1].Header.Get("X-Request-Id")).NotTo(Equal(actual[0].Header.Get("X-Request-Id")))
	})
})

func newRequest(c *httpclient.Client) *http.Request {
	r, err := c.NewRequest(context.Background())
	Expect(err).NotTo(HaveOccurred())
	return r
}

// newDownloader obtains the downloader, using a context whose stdout is the
// given buffer so that the default downloader is observable
func newDownloader(c *httpclient.Client, out io.Writer) httpclient.Downloader {
	d, err := c.NewDownloader(&cli.Context{Stdout: cli.NewWriter(out)})
	Expect(err).NotTo(HaveOccurred())
	return d
}
