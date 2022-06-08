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
})
