package httpclient_test

import (
	"encoding/json"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	DescribeTable("examples", func(name string, v interface{}, requiresUserInfo bool) {
		actual, err := httpclient.NewAuthenticator(name, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(v))
		Expect(actual.RequiresUserInfo()).To(Equal(requiresUserInfo))
	},
		Entry("basic", "basic", httpclient.BasicAuth, true),
		Entry("BASIC", "BASIC", httpclient.BasicAuth, true),
		Entry("blank", "", httpclient.NoAuth, false),
	)
})
