// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"bytes"
	"encoding"
	"net/http"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

// Expr provides the expression used within the "write out" flag
type Expr string

func (e Expr) Compile() *expander.Pattern {
	return expander.CompilePattern(string(e), "%(", ")")
}

func (e *Expr) UnmarshalText(b []byte) error {
	*e = Expr(string(b))
	return nil
}

// Expander converts the given string key into its variable expansion
type Expander = expander.Interface

// ExpandRequest provides an expander that provides variables from a request
// in the context of a client.
func ExpandRequest(r *http.Request) Expander {
	if r == nil {
		return expander.Func(func(s string) any {
			if s == "method" || s == "protocol" || s == "location" || s == "url" || s == "header" {
				return ""
			}
			if strings.HasPrefix(s, "location.") || strings.HasPrefix(s, "url.") || strings.HasPrefix(s, "header.") {
				return ""
			}
			return nil
		})
	}

	return expander.Compose(expander.Func(func(s string) any {
		switch s {
		case "method":
			return httpMethod(r.Method)
		case "protocol":
			return r.Proto
		case "location", "url":
			// Note - technically, we encourage and have documented %(request.url)
			// and %(redirect.location), but either is acceptable in either context
			// (i.e. %(request.location) and %(redirect.url) also work)
			return r.URL
		case "header":
			var buf bytes.Buffer
			r.Header.Write(&buf)
			return buf.String()
		}
		return nil
	}),
		expander.Prefix("location", expr.ExpandURL(r.URL)),
		expander.Prefix("url", expr.ExpandURL(r.URL)),
		expander.Prefix("header", ExpandHeader(r.Header)))
}

func ExpandResponse(r *http.Response) Expander {
	return expander.Compose(expander.Func(func(s string) any {
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
		case "transferEncoding":
			return strings.Join(r.TransferEncoding, ",")
		case "close":
			return r.Close
		case "uncompressed":
			return r.Uncompressed
		case "header":
			var buf bytes.Buffer
			r.Header.Write(&buf)
			return buf.String()
		}
		return nil
	}), expander.Prefix("header", ExpandHeader(r.Header)))
}

func ExpandHeader(h http.Header) Expander {
	return expander.Func(func(s string) any {
		return h.Get(headerCanonicalName(s))
	})
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
