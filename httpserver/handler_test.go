package httpserver_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewPingHandler", func() {

	It("writes out ping message", func() {
		recorder := httptest.NewRecorder()
		p := httpserver.NewPingHandler()
		p.ServeHTTP(recorder, nil)

		Expect(string(recorder.Body.Bytes())).To(Equal("ping\n"))
	})
})
