package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("URLValue", func() {

	DescribeTable("examples", func(value string, expected string) {
		u := new(httpclient.URLValue)
		err := u.Set(value)

		Expect(err).NotTo(HaveOccurred())

		uu, _ := u.URL()
		Expect(uu.String()).To(Equal(expected))
	},

		Entry("localhost", "localhost", "http://localhost"),
		Entry("example", "example.com", "http://example.com"),
		Entry("port", ":8080", "http://localhost:8080"),
		Entry("rooted", "/root", "/root"),
		Entry("empty", "", ""),
	)
})

var _ = Describe("UserInfo", func() {

	DescribeTable("examples", func(value string, expected string) {
		u := new(httpclient.UserInfo)
		err := u.Set(value)

		Expect(err).NotTo(HaveOccurred())
		Expect(u.String()).To(Equal(expected))
	},

		Entry("user only", "user", "user"),
		Entry("user and password", "user:go", "user:go"),
		Entry("empty password", "hello:", "hello:"),
	)
})

var _ = Describe("HeaderValue", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected *httpclient.HeaderValue) {
				actual := &httpclient.HeaderValue{}
				for _, a := range args {
					err := actual.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(actual).To(Equal(expected))
			},
			Entry(
				"nominal",
				[]string{"name=value"},
				&httpclient.HeaderValue{"name", "value"},
			),
			Entry(
				"escaped equal sign",
				[]string{"name\\=value=value"},
				&httpclient.HeaderValue{"name=value", "value"},
			),
			Entry(
				"separated by spaces",
				[]string{"name", "value"},
				&httpclient.HeaderValue{"name", "value"},
			),
			Entry(
				"key only",
				[]string{"name="},
				&httpclient.HeaderValue{"name", ""},
			),
			Entry(
				"colon",
				[]string{"name:value"},
				&httpclient.HeaderValue{"name", "value"},
			),
			Entry(
				"colon and space",
				[]string{"name: value"},
				&httpclient.HeaderValue{"name", "value"},
			),
		)
	})
})

var _ = Describe("HeaderCounter", func() {

	var (
		newCounter = func() cli.ArgCounter {
			return new(httpclient.HeaderValue).NewCounter()
		}
	)

	DescribeTable("examples",
		func(args []string) {
			actual := newCounter()
			for _, a := range args {
				err := actual.Take(a, true)
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(actual.Done()).NotTo(HaveOccurred())
		},
		Entry(
			"nominal",
			[]string{"name=value"},
		),
		Entry(
			"separated by spaces",
			[]string{"name", "value"},
		),
		Entry(
			"key only",
			[]string{"name="},
		),
		Entry(
			"colon",
			[]string{"name:value"},
		),
		Entry(
			"colon and space",
			[]string{"name: value"},
		),
	)

	DescribeTable("errors",
		func(args []string, expected string) {
			actual := newCounter()
			for _, a := range args {
				err := actual.Take(a, true)
				Expect(err).NotTo(HaveOccurred())
			}

			err := actual.Done()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		},
		Entry(
			"missing both",
			[]string{},
			"missing name and value",
		),
	)

})
