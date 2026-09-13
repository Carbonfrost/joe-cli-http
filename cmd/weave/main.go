// Copyright 2022, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !json_info

// Package main provides the entry point for weave
package main

import (
	"os"

	"github.com/Carbonfrost/joe-cli-http/internal/cmd/weave"
)

func main() {
	weave.Run(os.Args)
}
