package expr

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// HTTPStatus provides a representation of an HTTP
// status code that supports terminal formatting
type HTTPStatus int

// HTTPMethod provides a representation of an HTTP
// method that supports terminal formatting
type HTTPMethod string

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

func (s HTTPStatus) Message() string {
	return http.StatusText(int(s))
}

func (s HTTPStatus) Code() int {
	return int(s)
}

func (s HTTPStatus) Format(f fmt.State, verb rune) {
	if verb == 'C' {
		writeFormatted(f, s)
		return
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), int(s))
}

func (s HTTPStatus) String() string {
	return strconv.Itoa(int(s)) + " " + s.Message()
}

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

func (m HTTPMethod) Format(f fmt.State, verb rune) {
	if verb == 'C' {
		writeFormatted(f, m)
		return
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), string(m))
}

func (m HTTPMethod) String() string {
	return string(m)
}

func colorString(s string) string {
	return ExpandColors(s).(string)
}

func writeFormatted(f io.Writer, a formattable) {
	f.Write([]byte(colorString("reverse")))
	f.Write([]byte(colorString(strings.ToLower(a.Color()))))
	f.Write([]byte(a.String()))
	f.Write([]byte(colorString("reset")))
}

type formattable interface {
	fmt.Stringer
	Color() string
}
