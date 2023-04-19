package httpclient

import (
	"bytes"
	"encoding"
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

func (e Expr) Expand(r *Response) string {
	c := e.Compile()
	return c.Expand(func(s string) any {
		return expandToken(r, s)
	})
}

func expandToken(r *Response, tok string) any {
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
	case "header":
		var buf bytes.Buffer
		r.Header.Write(&buf)
		return buf.String()
	default:
		if name, ok := strings.CutPrefix(tok, "header."); ok {
			return r.Header.Get(headerCanonicalName(name))
		}
		return expr.UnknownToken(tok)
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
