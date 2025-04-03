package httpclient

import (
	"bytes"
	"encoding"
	"net/http"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
)

// Expr provides the expression used within the "write out" flag
type Expr string

func (e Expr) Compile() *expr.Pattern {
	return expr.CompilePattern(string(e), "%(", ")")
}

func (e *Expr) UnmarshalText(b []byte) error {
	*e = Expr(string(b))
	return nil
}

func ExpandResponse(r *Response) expr.Expander {
	return expr.ComposeExpanders(func(s string) any {
		switch s {
		case "status":
			return r.Status // "200 OK"
		case "statusCode":
			return httpStatus(r.StatusCode)
		case "http.version":
			return strings.TrimPrefix(r.Proto, "HTTP/")
		case "http.proto":
			return r.Proto
		case "http.protoMajor":
			return r.ProtoMajor
		case "http.protoMinor":
			return r.ProtoMinor
		case "contentLength":
			return r.ContentLength
		case "header":
			var buf bytes.Buffer
			r.Header.Write(&buf)
			return buf.String()
		}
		return nil
	}, expr.Prefix("header", ExpandHeader(r.Header)))
}

func ExpandHeader(h http.Header) expr.Expander {
	return func(s string) any {
		return h.Get(headerCanonicalName(s))
	}
}

func headerCanonicalName(s string) string {
	if strings.Contains(s, "-") {
		return s
	}

	// Convert Pascal and camel case to canonical names
	var buf bytes.Buffer
	pat := regexp.MustCompile("(^[a-z]|[A-Z])[^A-Z]*")

	submatchall := pat.FindAllString(s, -1)
	for i, element := range submatchall {
		if i > 0 {
			buf.WriteString("-")
		}
		buf.WriteString(element)
	}
	return buf.String()
}

var _ encoding.TextUnmarshaler = (*Expr)(nil)
