package httpserver // intentional

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WrapResponseWriter", func() {

	It("remembers wrote header when flushed", func() {
		f := &httpFancyWriter{basicWriter: basicWriter{ResponseWriter: httptest.NewRecorder()}}
		f.Flush()

		Expect(f.wroteHeader).To(BeTrue(), "want Flush to have set wroteHeader=true")
	})

	It("HTTP/2 remembers wrote header when flushed", func() {
		f := &http2FancyWriter{basicWriter{ResponseWriter: httptest.NewRecorder()}}
		f.Flush()

		Expect(f.wroteHeader).To(BeTrue(), "want Flush to have set wroteHeader=true")
	})

})
