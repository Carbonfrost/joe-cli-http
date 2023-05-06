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

		Expect(recorder.Body.String()).To(Equal("ping\n"))
	})
})

var _ = Describe("NewHeaderMiddleware", func() {

	It("sets up header with name", func() {
		recorder := httptest.NewRecorder()

		p := httpserver.NewHeaderMiddleware("Server", "Albatross")
		handler := p(httpserver.NewPingHandler())
		handler.ServeHTTP(recorder, nil)

		Expect(recorder.Header()).To(HaveKeyWithValue("Server", []string{"Albatross"}))
	})

	It("adds additional headers", func() {
		recorder := httptest.NewRecorder()

		handler := httpserver.NewPingHandler()
		handler = httpserver.NewHeaderMiddleware("Server", "A")(handler)
		handler = httpserver.NewHeaderMiddleware("Server", "B")(handler)
		handler.ServeHTTP(recorder, nil)

		Expect(recorder.Header()).To(HaveKeyWithValue("Server", []string{"B", "A"}))
	})
})
