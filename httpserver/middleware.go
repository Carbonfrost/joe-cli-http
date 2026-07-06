package httpserver

import (
	"io"
	"net/http"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

// NewRequestLoggerMiddleware provides handler middleware to write to access log
func NewRequestLoggerMiddleware(format string, out io.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if format == "" {
			format = defaultAccessLog
		}
		logFormat := expander.Compile(format).WithMeta("accessLog.default", metaDefaultAccessLog)
		return newRequestLoggerHandler(out, next, logFormat)
	}
}

// NewHeaderMiddleware provides handler middleware which simply adds the given
// header
func NewHeaderMiddleware(name, value string) func(http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(name, value)
			inner.ServeHTTP(w, r)
		})
	}
}
