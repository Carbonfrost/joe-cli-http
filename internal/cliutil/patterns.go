package cliutil

import (
	"github.com/Carbonfrost/joe-cli"
)

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
