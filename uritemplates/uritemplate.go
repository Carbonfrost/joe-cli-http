// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates

import (
	"encoding"
	"flag"

	"github.com/Carbonfrost/joe-cli-http/internal/uritemplates"
)

// URITemplate is a parsed representation of a URI template, which can
// also be used as a flag Value
type URITemplate struct {
	u *uritemplates.URITemplate
}

func Parse(text string) (*URITemplate, error) {
	u, err := uritemplates.Parse(text)
	if err != nil {
		return nil, err
	}
	return &URITemplate{u}, err
}

func (u *URITemplate) Expand(value any) (string, error) {
	s, err := u.u.Expand(ensureValues(value))
	return s, err
}

func (u *URITemplate) PartialExpand(value any) (string, error) {
	s, err := u.u.PartialExpand(ensureValues(value))
	return s, err
}

func (u *URITemplate) Names() []string {
	return u.u.Names()
}

func (u *URITemplate) Set(arg string) error {
	if u.u != nil {
		arg = u.u.String() + arg
	}
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

func ensureValues(value any) any {
	if val, ok := value.(Vars); ok {
		return map[string]any(val)
	}
	return value
}

var (
	_ flag.Value               = (*URITemplate)(nil)
	_ encoding.TextMarshaler   = (*URITemplate)(nil)
	_ encoding.TextUnmarshaler = (*URITemplate)(nil)
)
