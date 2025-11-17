// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates

import (
	"context"
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/value"
)

type contextKey string

var (
	tagged                    = cli.Data(SourceAnnotation())
	varsContextKey contextKey = "uritemplates_vars"
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", "joe-cli-http/uritemplates"
}

func Expand() cli.Action {
	return cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) error {
		if !c.Seen("template") {
			return c.Do(cli.DisplayHelpScreen())
		}

		tpl := c.Value("template").(*URITemplate)

		if c.Bool("partial") {
			rr, err := tpl.PartialExpand(fromContext(c))
			fmt.Fprintln(c.Stdout, rr)
			return err
		}

		rr, err := tpl.Expand(fromContext(c))
		fmt.Fprintln(c.Stdout, rr)
		return err
	}))
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.ContextValue(varsContextKey, Vars{}),
		cli.AddFlags([]*cli.Flag{
			{Uses: SetURITemplateVar()},
			{Uses: SetURITemplateVars()},
			{Uses: SetPartialExpand()},
		}...),

		cli.AddArg(&cli.Arg{
			Name:  "template",
			NArg:  cli.TakeUntilNextFlag,
			Value: new(URITemplate),
		}),
	)
}

func SetURITemplateVar(v ...*Var) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "param",
			Aliases:  []string{"T"},
			HelpText: "Specify a value used to fill the template",
			Value:    new(Var),
			Options:  cli.EachOccurrence,
		},
		cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) error {
			v := c.Value("").(*Var)
			fromContext(c).Add(v)
			return nil
		})),
		tagged,
	)
}

func SetURITemplateVars(v ...*Vars) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "params",
			Aliases:   []string{"t"},
			UsageText: "expr|@file",
			HelpText:  "Specify a template parameters using abbreviated syntax or from a JSON file",
			Value:     value.JSON(&Vars{}),
			Options:   cli.EachOccurrence | cli.AllowFileReference,
		},
		cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) error {
			v := c.Value("").(*Vars)
			return fromContext(c).Update(*v)
		})),
		tagged,
	)
}

func SetPartialExpand(b ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "partial",
			Aliases:  []string{"P"},
			Value:    cli.Bool(),
			HelpText: "When set, partially expand the template by preserving missing variables",
		},
		tagged,
	)
}

func fromContext(c context.Context) Vars {
	return c.Value(varsContextKey).(Vars)
}
