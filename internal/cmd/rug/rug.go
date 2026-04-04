// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rug

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/joe-cli/extensions/color"
)

func Run(args []string) {
	NewApp().Run(args)
}

func NewApp() *cli.App {
	return &cli.App{
		Name:     "rug",
		HelpText: "Expands RFC 6570 (level 4) URI templates",
		Uses: cli.Pipeline(
			&color.Options{},
			uritemplates.New(),
			cli.Sorted,
		),
		Version: build.Version,
		Action: cli.Pipeline(
			cli.IfMatch(
				cli.HasSeen("template"),
				nil,
				cli.Pipeline(
					cli.DisplayHelpScreen(),
					cli.Exit(2),
				),
			),
			uritemplates.ExpandAndPrint(),
		),
	}
}
