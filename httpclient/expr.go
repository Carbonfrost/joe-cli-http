package httpclient

import (
	"bytes"
	"encoding"
	"net/http"
	"regexp"
	"strconv"
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
	return func(s string) any {
		return expandToken(r, s)
	}
}

func ExpandHeader(h http.Header) expr.Expander {
	return func(s string) any {
		return expandHeader(h, s)
	}
}

func expandToken(r *Response, tok string) any {
	if strings.HasPrefix(tok, "header") {
		return expandHeader(r.Header, tok)
	}
	switch tok {
	case "status":
		return r.Status // "200 OK"
	case "statusCode":
		return strconv.Itoa(r.StatusCode)
	case "http.version":
		return strings.TrimPrefix(r.Proto, "HTTP/")
	case "http.proto":
		return r.Proto
	case "http.protoMajor":
		return strconv.Itoa(r.ProtoMajor)
	case "http.protoMinor":
		return strconv.Itoa(r.ProtoMinor)
	case "contentLength":
		return strconv.FormatInt(r.ContentLength, 10)
	}
	return nil
}

func expandHeader(h http.Header, tok string) any {
	switch tok {
	case "header":
		var buf bytes.Buffer
		h.Write(&buf)
		return buf.String()
	default:
		if name, ok := strings.CutPrefix(tok, "header."); ok {
			return h.Get(headerCanonicalName(name))
		}
		return nil
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
