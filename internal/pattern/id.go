// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pattern

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
)

const idDataKey = "ID"

type hasLookupData interface {
	LookupData(any) (any, bool)
}

// IDAnnotation provides an action which annotates a flag or arg with the
// identity id. The annotation is placed early and only to flags and args
// directly defined by the package.
func IDAnnotation[T any](id T) cli.Action {
	return cli.Data(idDataKey, id)
}

// LookupID gets the identity annotation of a flag or arg, if present.
func LookupID[T any](target any) (T, bool) {
	var zero T
	data, ok := target.(hasLookupData)
	if !ok {
		return zero, false
	}
	v, ok := data.LookupData(idDataKey)
	if !ok {
		return zero, false
	}
	id, ok := v.(T)
	return id, ok
}

// IdFlag creates a flag which uses the given action and carries the
// identity annotation id.
func IdFlag[T any](id T, uses cli.Action) *cli.Flag {
	return &cli.Flag{
		Uses: cli.Pipeline(uses, IDAnnotation(id)),
	}
}

// Unmarshal parses as a text unmarshaler
func Unmarshal[T ~int, Index uint8 | uint16](id *T, data []byte, names string, index []Index) error {
	for i := 0; i < len(index)-1; i++ {
		if names[index[i]:index[i+1]] == string(data) {
			*id = T(i)
			return nil
		}
	}
	return fmt.Errorf("unexpected value %q", string(data))
}
