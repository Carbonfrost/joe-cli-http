package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {

	Describe("SetHeader", func() {
		It("aggregates header values", func() {
			s := httpclient.New()
			s.SetHeader(&httpclient.HeaderValue{"Link", "Something"})
			s.SetHeader(&httpclient.HeaderValue{"Link", "SomethingElse"})

			Expect(s.Request.Header).To(HaveKeyWithValue("Link", []string{"Something", "SomethingElse"}))
		})
	})

	Describe("SetBody", func() {
		It("sets raw body value", func() {
			s := httpclient.New()
			s.SetBody("raw content")

			body, _ := io.ReadAll(s.BodyContent.Read())
			Expect(string(body)).To(Equal("raw content"))
		})
	})

	Describe("New", func() {
		It("sets up the default user agent string", func() {
			s := httpclient.New()
			expected := "Go-http-client/1.1 (joe-cli-http/development, +https://github.com/Carbonfrost/joe-cli-http)"
			Expect(s.Request.Header).To(HaveKeyWithValue("User-Agent", []string{expected}))
		})
	})
})
