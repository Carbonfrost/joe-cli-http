// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package weave

import (
	"context"
	"fmt"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/color"
)

func Run(args []string) {
	NewApp().Run(args)
}

func NewApp() *cli.App {
	return &cli.App{
		Name:     "weave",
		HelpText: "Provides access to a simple Go HTTP server for files and proxy handling",
		Uses: cli.Pipeline(
			httpserver.DefaultServer(),
			httpserver.WithShutdownFunc(func(context.Context) {
				fmt.Fprintf(os.Stderr, "Goodbye!\n")
			}),
			&color.Options{},
			httpserver.RunServer(),
			httpserver.HandlerRegistry,
			cli.Sorted,
		),
		Version: build.Version,
		Flags: []*cli.Flag{
			{
				Name:     "chdir",
				HelpText: "Change directory into the specified working {DIRECTORY}",
				Value:    &cli.File{Name: "."},
				Options:  cli.MustExist | cli.WorkingDirectory,
			},
			{
				Uses: httpserver.SetHandler(),
			},
			{
				Uses: httpserver.ListHandlers(),
			},
		},
		Args: []*cli.Arg{
			{
				Name:     "directories",
				HelpText: "Specifies static directories to serve",
				NArg:     cli.TakeUntilNextFlag,
				Uses:     httpserver.SetFileServerHandler(),
			},
		},
	}
}
