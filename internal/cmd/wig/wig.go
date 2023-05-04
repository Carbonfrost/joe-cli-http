package wig

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/color"

	// These hash algorithms need to be available for --integrity to work
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
)

const wigURL = "https://github.com/Carbonfrost/joe-cli-http/cmd/wig"

func defaultUserAgent() string {
	version := build.Version
	if len(version) == 0 {
		version = "development"
	}
	return fmt.Sprintf("Go-http-client/1.1 (wig/%s, +%s)", version, wigURL)
}

func Run(args []string) {
	NewApp().Run(args)
}

func NewApp() *cli.App {
	return &cli.App{
		Name:     "wig",
		HelpText: "Provides access to the Go HTTP client with some cURL compatibility",
		Uses: cli.Pipeline(
			httpclient.New(
				httpclient.WithDefaultUserAgent(defaultUserAgent()),
			),
			&color.Options{},
			cli.Sorted,
		),
		Action:  httpclient.FetchAndPrint(),
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
