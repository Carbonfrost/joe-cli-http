// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"os"

	"github.com/Carbonfrost/joe-cli-http/internal/cmd/weave"
)

func main() {
	weave.Run(os.Args)
}
