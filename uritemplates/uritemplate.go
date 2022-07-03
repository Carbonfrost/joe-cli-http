package uritemplates

import (
	"encoding"
	"flag"

	"github.com/Carbonfrost/joe-cli-http/internal/uritemplates"
)

// URITemplate is a parsed representation of a URI template, which can
// also be used as a flag Value
type URITemplate struct {
	u *uritemplates.UriTemplate
}

func Parse(text string) (*URITemplate, error) {
	u, err := uritemplates.Parse(text)
	if err != nil {
		return nil, err
	}
	return &URITemplate{u}, err
}

func (u *URITemplate) Expand(value interface{}) (string, error) {
	if val, ok := value.(Vars); ok {
		value = map[string]interface{}(val)
	}

	s, err := u.u.Expand(value)
	return s, err
}

func (u *URITemplate) Names() []string {
	return u.u.Names()
}

func (u *URITemplate) Set(arg string) error {
	uri, err := uritemplates.Parse(arg)
	u.u = uri
	return err
}

func (u *URITemplate) String() string {
	return u.u.String()
}

// MarshalText provides the textual representation
func (u *URITemplate) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText converts the textual representation
func (u *URITemplate) UnmarshalText(b []byte) error {
	res, err := Parse(string(b))
	if err != nil {
		return err
	}
	*u = *res
	return nil
}

var (
	_ flag.Value               = (*URITemplate)(nil)
	_ encoding.TextMarshaler   = (*URITemplate)(nil)
	_ encoding.TextUnmarshaler = (*URITemplate)(nil)
)
