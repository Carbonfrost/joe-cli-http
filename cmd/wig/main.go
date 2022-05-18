package main

import (
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/color"
)

func main() {
	createApp().Run(os.Args)
}

func createApp() *cli.App {
	return &cli.App{
		Name:     "wig",
		HelpText: "Provides access to the Go HTTP client with some cURL compatibility",
		Uses: cli.Pipeline(
			&httpclient.Options{},
			&color.Options{},
		),
		Action:  httpclient.FetchAndPrint(),
		Version: build.Version,
	}
}
