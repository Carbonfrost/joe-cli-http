package httpclient

import (
	"bytes"
	"encoding"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Expr provides the expression used within the "write out" flag
type Expr string

var patternRegexp = regexp.MustCompile(`%\((.+?)\)`)

func (w *Expr) UnmarshalText(b []byte) error {
	*w = Expr(string(b))
	return nil
}

func (w Expr) Expand(r *Response) string {
	str := string(w)

	// Handle escape sequences
	if s, err := strconv.Unquote(`"` + str + `"`); err == nil {
		str = s
	}
	content := []byte(str)
	allIndexes := patternRegexp.FindAllSubmatchIndex(content, -1)
	var buf bytes.Buffer

	var index int
	for _, loc := range allIndexes {
		if index < loc[0] {
			buf.Write(content[index:loc[0]])
		}

		// Expressions
		key := content[loc[2]:loc[3]]
		buf.WriteString(expandToken(r, string(key)))
		index = loc[1]
	}
	if index < len(content) {
		buf.Write(content[index:])
	}

	return buf.String()
}

func expandToken(r *Response, tok string) string {
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
		return fmt.Sprintf("%%!(unknown token: %s)", tok)
	}
}

var _ encoding.TextUnmarshaler = (*Expr)(nil)