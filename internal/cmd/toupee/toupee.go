package toupee

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
		Name:     "toupee",
		HelpText: "Expands RFC 6570 (level 4) URI templates",
		Uses: cli.Pipeline(
			&color.Options{},
			uritemplates.FlagsAndArgs(),
			cli.Sorted,
		),
		Version: build.Version,
		Action:  uritemplates.Expand(),
	}
}
