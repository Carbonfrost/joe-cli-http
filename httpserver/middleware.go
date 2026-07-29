package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

// NewRequestLoggerMiddleware provides handler middleware to write to access log
func NewRequestLoggerMiddleware(format string, out io.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if format == "" {
			format = defaultAccessLog
		}
		logFormat := expander.Compile(format, expander.WithMeta("accessLog.default", metaDefaultAccessLog))
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

// NewRecoveryMiddleware is middleware that recovers from panics, logs the panic and
// backtrace, and writes out a HTTP 500 (Internal Server Error) status if
// appropriate.
func NewRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}
				printDebugStack()

				if r.Header.Get("Connection") != "Upgrade" {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
func printDebugStack() {
	debugStack := string(debug.Stack())
	fmt.Fprintln(os.Stderr, debugStack)
}
