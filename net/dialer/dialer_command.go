// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialer

import (
	"fmt"
	"net"
	"reflect"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

const (
	networkOptions = "Network interface options"
)

var (
	tagged  = cli.Data(SourceAnnotation())
	pkgPath = reflect.TypeFor[Dialer]().PkgPath()
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// Action provides a context action that affects the dialer
type Action = cli.Action

func SetDialTimeout(s ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "dial-timeout",
			HelpText: "maximum amount of time a dial will wait for a connect to complete",
			Category: networkOptions,
		},
		bind.Action(WithDialTimeout, bind.Exact(s...)),
		tagged,
	)
}

func SetDialKeepAlive(v ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "dial-keep-alive",
			HelpText: "Specifies the interval between keep-alive probes for an active network connection.",
			Category: networkOptions,
		},
		bind.Action(WithDialKeepAlive, bind.Exact(v...)),
		tagged,
	)
}

func SetDisableDialKeepAlive() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "disable-dial-keep-alive",
			HelpText: "Disable dialer keep-alive probes",
			Category: networkOptions,
			Value:    new(bool),
		},
		cli.At(cli.ActionTiming, WithDisableDialKeepAlive(true)),
		tagged,
	)
}

func SetBindAddress(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "bind-address",
			UsageText: "HOSTNAME|IP",
			HelpText:  "Bind client TCP/IP connections to ADDRESS on the local machine",
			Category:  networkOptions,
		},
		bind.Action(WithBindAddress, bind.Exact(v...)),
		tagged,
	)
}

func SetInterface(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       "interface",
			HelpText:   "Use network {INTERFACE} by name or address to connect",
			Category:   networkOptions,
			Completion: completeInterfaces(),
		},
		bind.Action(WithInterface, bind.Exact(v...)),
		tagged,
	)
}

func ListInterfaces() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-interfaces",
			Value:    cli.Bool(),
			Options:  cli.Exits,
			HelpText: "List network interfaces and then exit",
			Category: networkOptions,
		},
		listInterfaces(),
		tagged,
	)
}

// FlagsAndArgs provides the flags and args for configuring the dialer.  By
// default, these configure the dialer registered under the default context
// key.  To configure a dialer registered with WithContextKey, specify the
// context key; the flags are then annotated with ContextKeyAnnotation.  At
// most one context key can be specified.
func FlagsAndArgs(contextKey ...any) Action {
	flag := func(id ID, uses Action) *cli.Flag {
		if len(contextKey) > 0 {
			uses = cli.Pipeline(uses, ContextKeyAnnotation(contextKey[0]))
		}
		return idFlag(id, uses)
	}

	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			flag(IDDialTimeout, SetDialTimeout()),
			flag(IDDialKeepAlive, SetDialKeepAlive()),
			flag(IDDisableDialKeepAlive, SetDisableDialKeepAlive()),
			flag(IDBindAddress, SetBindAddress()),
			flag(IDInterface, SetInterface()),
			flag(IDListInterfaces, ListInterfaces()),
		}...))
}

func listInterfaces() Action {
	return cli.At(cli.ActionTiming, cli.ActionOf(func(c *cli.Context) error {
		eths, _ := net.Interfaces()
		for _, s := range eths {
			addrs, err := s.Addrs()
			if err != nil {
				fmt.Fprintf(c.Stdout, "%s\t%v\n", s.Name, err)
				continue
			}
			fmt.Fprint(c.Stdout, s.Name)
			for i, a := range addrs {
				if i > 0 {
					fmt.Fprintln(c.Stdout)
				}
				fmt.Fprintf(c.Stdout, "\t%s\t%s", a.Network(), a.String())
			}
			fmt.Fprintln(c.Stdout)
		}
		return nil
	}))
}

func completeInterfaces() cli.CompletionFunc {
	return func(cc *cli.Context) []cli.CompletionItem {
		values := []string{}
		eths, _ := net.Interfaces()
		for _, s := range eths {
			values = append(values, s.Name)

			addrs, err := s.Addrs()
			if err != nil {
				continue
			}
			for _, a := range addrs {
				values = append(values, a.String())
			}
		}
		return cli.ValueCompletion(values...).Complete(cc)
	}
}
