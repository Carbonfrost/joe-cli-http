// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

var expandColors = expander.Colors().Expand

// HTTPStatus provides a representation of an HTTP
// status code that supports terminal formatting
type HTTPStatus int

// HTTPMethod provides a representation of an HTTP
// method that supports terminal formatting
type HTTPMethod string

// Color gets the color string that applies to the status
func (s HTTPStatus) Color() string {
	switch 100 * (s / 100) {
	case 100:
		return "Magenta"
	case 200:
		return "Green"
	case 300:
		return "Yellow"
	case 400, 500:
		fallthrough
	default:
		return "Red"
	}
}

// Message converts the status in to the canonical status text message
func (s HTTPStatus) Message() string {
	return http.StatusText(int(s))
}

// Code converts the value into a value
func (s HTTPStatus) Code() int {
	return int(s)
}

// Format implements [fmt.Formatter]
func (s HTTPStatus) Format(f fmt.State, verb rune) {
	if verb == 'C' {
		writeFormatted(f, s)
		return
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), int(s))
}

// String provides the string representation of the status
func (s HTTPStatus) String() string {
	return strconv.Itoa(int(s)) + " " + s.Message()
}

// Color gets the color string that applies to the method
func (m HTTPMethod) Color() string {
	switch m {
	case "DELETE":
		return "Red"
	case "GET":
		return "Blue"
	default:
		return "Magenta"
	}
}

// Format implements [fmt.Formatter]
func (m HTTPMethod) Format(f fmt.State, verb rune) {
	if verb == 'C' {
		writeFormatted(f, m)
		return
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), string(m))
}

// String provides the string representation of the method
func (m HTTPMethod) String() string {
	return string(m)
}

func colorString(s string) string {
	return expandColors(s).(string)
}

func writeFormatted(f io.Writer, a formattable) {
	f.Write([]byte(colorString("reverse")))
	f.Write([]byte(colorString(strings.ToLower(a.Color()))))
	f.Write([]byte(a.String()))
	f.Write([]byte(colorString("reset")))
}

type formattable interface {
	fmt.Stringer
	fmt.Formatter
	Color() string
}

var (
	_ formattable = HTTPStatus(0)
	_ formattable = HTTPMethod("")
)
