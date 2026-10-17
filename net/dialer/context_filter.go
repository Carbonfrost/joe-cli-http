// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialer

import (
	"context"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
)

// ContextFilter provides the context filters in this package
type ContextFilter int

const (
	// Defines provides a context filter for flags and args defined by this package
	Defines ContextFilter = iota
)

var (
	definesImpl = cli.HasData(SourceAnnotation())

	contextFilterLabels = map[ContextFilter][2]string{
		Defines: {"defined in joe-cli-http/net/dialer pkg", "dialer.DEFINES"},
	}
)

// Matches the context filter
func (f ContextFilter) Matches(ctx context.Context) bool {
	switch f {
	case Defines:
		return definesImpl.Matches(ctx)
	}
	return false
}

// String produces a textual representation of the context filter
func (f ContextFilter) String() string {
	return contextFilterLabels[f][1]
}

// MarshalText provides the textual representation
func (f ContextFilter) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText converts the textual representation
func (f *ContextFilter) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, v := range contextFilterLabels {
		if v[1] == token {
			*f = k
			return nil
		}
	}
	return nil
}

// Describe produces a representation of the context filter suitable for use in messages
func (f ContextFilter) Describe() string {
	return contextFilterLabels[f][0]
}
