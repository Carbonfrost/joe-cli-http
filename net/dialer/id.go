// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialer

import (
	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/pattern"
)

//go:generate go tool stringer -type=ID -trimprefix=ID

// ID identifies each of the flags and args registered by FlagsAndArgs
type ID int

const (
	IDDialTimeout ID = iota
	IDDialKeepAlive
	IDDisableDialKeepAlive
	IDBindAddress
	IDInterface
	IDListInterfaces
)

// IDAnnotation provides an action which annotates the identity of a flag or
// arg registered by FlagsAndArgs.  It is applied directly to the flag or arg
// at its point of registration, rather than from within the Uses pipeline or
// binding delegate of the corresponding SetX function.
func IDAnnotation(id ID) Action {
	return pattern.IDAnnotation(id)
}

// LookupID gets the identity of a flag or arg registered by FlagsAndArgs, if
// present.  target can be any value which exposes annotation data, such as
// *cli.Flag, *cli.Arg, or *cli.Context.
func LookupID(target any) (ID, bool) {
	return pattern.LookupID[ID](target)
}

// ContextKeyAnnotation provides an action which annotates a flag, arg, or
// command with the context key of the dialer that it configures.  The SetX
// actions and the Option actions resolve the dialer using this key rather
// than the default context key.  The annotation is found from the flag or arg
// itself or from any of its enclosing commands.
func ContextKeyAnnotation(contextKey any) Action {
	return pattern.ContextKeyAnnotation(contextKey)
}

// LookupContextKey gets the context key which the flag, arg, or command was
// annotated with by ContextKeyAnnotation, if present.  target can be any
// value which exposes annotation data, such as *cli.Flag, *cli.Arg, or
// *cli.Context.
func LookupContextKey(target any) (contextKey any, ok bool) {
	return pattern.LookupContextKey(target)
}

func idFlag(id ID, uses Action) *cli.Flag {
	return pattern.IdFlag(id, uses)
}

// UnmarshalText parses the string representation of the Id, as produced by
// String, back into an Id.
func (i *ID) UnmarshalText(text []byte) error {
	return pattern.Unmarshal(i, text, _ID_name, _ID_index[:])
}
