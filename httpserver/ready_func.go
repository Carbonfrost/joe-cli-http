// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver

import (
	"fmt"
	"context"
	"os"

	"github.com/Carbonfrost/joe-cli"
)

// ReadyFunc provides a function for when the server has started or stopped
type ReadyFunc func(context.Context)


// DefaultReadyFunc provides the default behavior when the server starts
var (
	DefaultReadyFunc = ComposeReadyFuncs(
		ReportListening(),
	)

)

// ComposeReadyFuncs provides a ReadyFunc that combines a sequence
func ComposeReadyFuncs(v ...ReadyFunc) ReadyFunc {
	return func(c context.Context) {
		for _, f := range v {
			if f == nil {
				continue
			}
			f(c)
		}
	}
}

// OpenInBrowser is a function to open the server in the browser.  This
// function is passed as a value to WithReadyFunc
func OpenInBrowser(path ...string) ReadyFunc {
	return func(c context.Context) {
		err := FromContext(c).OpenInBrowser(path...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// ReportListening is a ready func that prints a message to stderr that the
// server is listening
func ReportListening() ReadyFunc {
	return func(c context.Context) {
		s := FromContext(c)
		s.ReportListening()
	}
}

func (r ReadyFunc) Execute(c context.Context) error {
	FromContext(c).Apply(AddReadyFunc(r))
	return nil
}

var _ cli.Action = (ReadyFunc)(nil)
