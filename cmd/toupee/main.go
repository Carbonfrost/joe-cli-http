package main

import (
	"fmt"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

func main() {
	createApp().Run(os.Args)
}

func createApp() *cli.App {
	return &cli.App{
		Name:     "toupee",
		HelpText: "Exapnds RFC 6570 (level 4) URI templates",
		Version:  build.Version,
		Args: []*cli.Arg{
			{
				Name:  "template",
				Value: cli.String(),
			},
		},
		Action: func(c *cli.Context) error {
			tpl, err := uritemplates.Parse(c.String("template"))
			if err != nil {
				return err
			}
			vars := map[string]interface{}{}
			rr, err := tpl.Expand(vars)
			fmt.Println(rr)
			return err
		},
	}
}
