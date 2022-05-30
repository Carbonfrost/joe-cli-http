package httpserver_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpserver"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {

	Describe("Addr", func() {

		DescribeTable("examples", func(fn func(*httpserver.Server), expected string) {
			s := httpserver.New()
			fn(s)

			Expect(s.Server.Addr).To(Equal(expected))
		},

			Entry("default", func(_ *httpserver.Server) {}, "localhost:8000"),
			Entry("host",
				func(s *httpserver.Server) {
					s.SetHostname("elvis.localhost")
				}, "elvis.localhost:8000"),
			Entry("port",
				func(s *httpserver.Server) {
					s.SetPort(1619)
				}, "localhost:1619"),
			Entry("host and port",
				func(s *httpserver.Server) {
					s.SetHostname("elvis.localhost")
					s.SetPort(1619)
				}, "elvis.localhost:1619"),
			Entry("addr",
				func(s *httpserver.Server) {
					s.SetAddr("elvis.localhost:8900")
				}, "elvis.localhost:8900"),
		)
	})
})
