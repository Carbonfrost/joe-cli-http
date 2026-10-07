// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uritemplates

import (
	"encoding"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/pattern"
)

//go:generate go tool stringer -type=ID -trimprefix=ID

// ID identifies each of the flags and args registered by FlagsAndArgs
type ID int

// Various identifiers for flags and args registered by the package
const (
	_ ID = iota
	IDPartialExpand
	IDTemplate
	IDURITemplateVar
	IDURITemplateVars
)

func IDAnnotation(id ID) Action {
	return pattern.IDAnnotation(id)
}

// LookupID gets the identity of a flag or arg registered by FlagsAndArgs, if
// present.
func LookupID(target any) (ID, bool) {
	return pattern.LookupID[ID](target)
}

// MarshalText provides the implementation of [encoding.TextMarshaler]
func (i ID) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText provides the implementation of [encoding.TextUnarshaler]
func (i *ID) UnmarshalText(data []byte) error {
	return pattern.Unmarshal(i, data, _ID_name, _ID_index[:])
}

func idFlag(id ID, uses Action) *cli.Flag {
	return pattern.IdFlag(id, uses)
}

var (
	_ encoding.TextMarshaler   = ID(0)
	_ encoding.TextUnmarshaler = new(ID(0))
)
