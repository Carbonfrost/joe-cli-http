// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pattern

import (
	"github.com/Carbonfrost/joe-cli"
)

const contextKeyDataKey = "ContextKey"

func ContextKeyAnnotation(contextKey any) cli.Action {
	return cli.Data(contextKeyDataKey, contextKey)
}

func LookupContextKey(target any) (any, bool) {
	return lookupData(target, contextKeyDataKey)
}
