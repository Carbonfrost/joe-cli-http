package httpclient_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resolver", func() {

	DescribeTable("examples", func(locations []string, expected []string) {
		r := httpclient.NewDefaultLocationResolver()
		for _, loc := range locations {
			r.Add(loc)
		}

		actual, err := r.Resolve(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(urisToStrings(actual)).To(Equal(expected))
	},
		Entry(
			"fixup host scheme",
			[]string{"example.com"},
			[]string{"http://example.com"},
		),
		Entry(
			"resolve relative",
			[]string{"https://example.com", "hello"},
			[]string{"https://example.com", "https://example.com/hello"},
		),
	)
})

func urisToStrings(results []httpclient.Location) []string {
	res := make([]string, len(results))
	for i, item := range results {
		_, u, _ := item.URL(context.Background())
		res[i] = u.String()
	}
	return res
}
