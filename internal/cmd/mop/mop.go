// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mop provides the mop app, which is a WebSocket client
package mop

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli-http/websocket"
	"github.com/Carbonfrost/joe-cli/extensions/color"
)

const mopURL = "https://github.com/Carbonfrost/joe-cli-http/cmd/mop"

func defaultUserAgent() string {
	version := build.Version
	if len(version) == 0 {
		version = "development"
	}
	return fmt.Sprintf("Go-http-client/1.1 (mop/%s, +%s)", version, mopURL)
}

// Run runs the mop app
func Run(args []string) {
	NewApp().Run(args)
}

// NewApp creates the app for mop
func NewApp() *cli.App {
	return &cli.App{
		Name:     "mop",
		HelpText: "Provides access to a WebSocket client for sending and receiving messages",
		Uses: cli.Pipeline(
			websocket.New(
				websocket.WithHeader("User-Agent", defaultUserAgent()),
			),
			&color.Options{},
			cli.Sorted,
		),
		Action: cli.Pipeline(
			displayHelpOnNoArgs,
			websocket.ConnectAndPrint(),
		),
		Version: build.Version,
		Flags: []*cli.Flag{
			{
				Name:     "chdir",
				HelpText: "Change directory into the specified working {DIRECTORY}",
				Value:    &cli.File{Name: "."},
				Options:  cli.MustExist | cli.WorkingDirectory,
			},
		},
	}
}

func displayHelpOnNoArgs(c *cli.Context) error {
	if !c.Seen("url") {
		return c.Do(cli.DisplayHelpScreen("mop"))
	}
	return nil
}
