// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"strings"
)

// RedactHeaderField returns a copy of the header with the values
// redacted
func RedactHeaderField(key string, value []string, redactoropt ...HeaderFieldRedactor) []string {
	var redactor HeaderFieldRedactor = redactHeaderField
	if len(redactoropt) > 0 {
		redactor = composeRedactor(redactoropt...)
	}
	return redactor(key, value)
}

// HeaderFieldRedactor provides the logic of redacting the output of
// header fields before the http trace processes them with
// the WroteHeaderField method.  The main use case is removing secrets
type HeaderFieldRedactor func(name string, value []string) []string

func composeRedactor(fns ...HeaderFieldRedactor) HeaderFieldRedactor {
	return func(name string, value []string) []string {
		for _, fn := range fns {
			value = fn(name, value)
		}
		return value
	}
}

func (c *Client) redactHeader(name string, value []string) string {
	return strings.Join(redactHeaderField(name, value), ", ")
}

func redactHeaderField(name string, value []string) []string {
	result := make([]string, len(value))
	for i, s := range value {
		switch {
		case strings.EqualFold(name, "authorization"):
			result[i] = redactAuthorization(s)

		case strings.Contains(strings.ToLower(name), "-key") || strings.Contains(strings.ToLower(name), "token"):
			result[i] = redactGeneric(s)

		default:
			result[i] = s
		}
	}
	return result
}

func redactAuthorization(s string) string {
	h, r, _ := strings.Cut(s, " ")
	switch strings.ToLower(h) {
	case "api-key", "token", "basic", "bearer":
		return h + " " + redactGeneric(r)
	}
	return s
}

func redactGeneric(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	if len(s) <= 16 {
		return strings.Repeat("*", len(s)-2) + s[len(s)-2:]
	}
	return s[0:4] + strings.Repeat("*", len(s)-6) + s[len(s)-2:]
}
