package uritemplates

import (
	"context"
	"fmt"

	"github.com/Carbonfrost/joe-cli"
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
	return cli.ActionFunc(func(c *cli.Context) error {
		tpl := c.Value("template").(*URITemplate)
		rr, err := tpl.Expand(fromContext(c))
		fmt.Fprintln(c.Stdout, rr)
		return err
	})
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.ContextValue(varsContextKey, Vars{}),
		cli.AddFlags([]*cli.Flag{
			{Uses: SetURITemplateVar()},
			{Uses: SetURITemplateVars()},
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
		cli.AtTiming(cli.ActionFunc(func(c *cli.Context) error {
			v := c.Value("").(*Var)
			fromContext(c).Add(v)
			return nil
		}), cli.ActionTiming),
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
			Value:     &Vars{},
			Options:   cli.EachOccurrence | cli.AllowFileReference,
		},
		cli.AtTiming(cli.ActionFunc(func(c *cli.Context) error {
			v := c.Value("").(*Vars)
			return fromContext(c).Update(*v)
		}), cli.ActionTiming),
		tagged,
	)
}

func fromContext(c context.Context) Vars {
	return c.Value(varsContextKey).(Vars)
}
