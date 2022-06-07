package cliutil

import (
	"context"
	"github.com/Carbonfrost/joe-cli"
)

const (
	expectedOneArg = "expected 0 or 1 arg"
)

// FlagBinding creates a flag which either takes a specific value to set
// or takes its value from the flag.  args is the optional value to set
func FlagBinding[T, V any](
	fromContext func(context.Context) *T,
	binder func(*T, V) error,
	args []V,
	proto *cli.Prototype,
	otherUses ...cli.Action,
) cli.Action {
	switch len(args) {
	case 0:
		proto.Setup = cli.Setup{
			Uses: cli.Pipeline(cli.BindContext(fromContext, binder)).Append(otherUses...),
		}
		return proto
	case 1:
		return cli.BindContext(fromContext, binder, args[0])
	default:
		panic(expectedOneArg)
	}
}

// DualSetup sets up optional setup that applies to both Uses and Action timing.
func DualSetup(a cli.Action) cli.Setup {
	return cli.Setup{
		Optional: true,
		Uses:     cli.Pipeline(a, cli.Data("_DidDualSetupUses", true)),
		Action: func(c *cli.Context) error {
			if _, ok := c.LookupData("_DidDualSetupUses"); ok {
				return nil
			}
			return c.Do(a)
		},
	}
}
