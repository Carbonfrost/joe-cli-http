// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tls

import (
	gotls "crypto/tls"
	"reflect"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

const (
	tlsOptions = "TLS options"
)

var (
	tagged  = cli.Data(SourceAnnotation())
	pkgPath = reflect.TypeFor[Config]().PkgPath()
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

func tlsVersionFlag(minVersion, maxVersion uint16, proto *cli.Prototype) cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Uses: cli.Pipeline(
				&cli.Prototype{
					Value: new(bool),
				},
				proto,
			),
			Action: func(c *cli.Context) error {
				s := FromContext(c)
				if c.Bool("") {
					s.MinVersion = minVersion
					s.MaxVersion = maxVersion
				}
				return nil
			},
		},
		tagged,
	)
}

func SetCACertFile(path ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "cacert",
			HelpText:  "CA certificate to verify peer against (PEM format)",
			UsageText: "PATH",
			Options:   cli.EachOccurrence,
			Category:  tlsOptions,
		},
		bind.Action(AddRootCACertFile, bind.Exact(path...)),
		tagged,
	)
}

func SetCACertPath(path ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "capath",
			HelpText:  "CA directory to verify peer against",
			UsageText: "DIRECTORY",
			Options:   cli.EachOccurrence,
			Category:  tlsOptions,
		},
		bind.Action(AddRootCACertPath, bind.Exact(path...)),
		tagged,
	)
}

func SetClientCertFile(path ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "cert",
			Aliases:   []string{"E"},
			HelpText:  "Client certificate file (PEM format)",
			UsageText: "PATH",
			Category:  tlsOptions,
			Uses:      cli.Requires("key"),
		},
		bind.Action2(
			AddX509KeyPair, bind.File("cert").Name(), bind.File("key").Name(),
		),
		tagged,
	)
}

func SetKeyFile(path ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "key",
			HelpText:  "Private key file (PEM format)",
			UsageText: "PATH",
			Value:     new(cli.File),
			Category:  tlsOptions,
			Uses:      cli.Requires("cert"),
		},
		// Provides no action - it is provided by above
		tagged,
	)
}

func SetTime(s ...*cli.File) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "time",
			HelpText:  "Specifies a {FILE} whose mtime is used to represent the current time in TLS configuration",
			UsageText: "PATH",
			Category:  tlsOptions,
		},
		bind.Action(WithTimeHelper, bind.Exact(s...)),
		tagged,
	)
}

func SetServerName(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "server-name",
			HelpText: "Used to verify the {HOSTNAME} on certificates unless verification is being skipped",
			Category: tlsOptions,
		},
		bind.Action(WithServerName, bind.Exact(s...)),
		tagged,
	)
}

func SetNextProtos(s ...[]string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "next-protos",
			HelpText: "List of ALPN supported application level protocols, in order of preference.",
			Category: tlsOptions,
		},
		bind.Action(WithNextProtos, bind.Exact(s...)),
		tagged,
	)
}

func SetTLSv1() cli.Action {
	return tlsVersionFlag(gotls.VersionTLS10, gotls.VersionTLS13, &cli.Prototype{
		Name:     "tlsv1",
		HelpText: "Use TLSv1.0 or higher.  This is implied as this tool doesn't support SSLv3",
		Category: tlsOptions,
	})
}

func SetTLSv1_0() cli.Action {
	return tlsVersionFlag(gotls.VersionTLS10, gotls.VersionTLS10, &cli.Prototype{
		Name:     "tlsv1.0",
		HelpText: "Use TLSv1.0",
		Category: tlsOptions,
	})
}

func SetTLSv1_1() cli.Action {
	return tlsVersionFlag(gotls.VersionTLS11, gotls.VersionTLS11, &cli.Prototype{
		Name:     "tlsv1.1",
		HelpText: "Use TLSv1.1",
		Category: tlsOptions,
	})
}

func SetTLSv1_2() cli.Action {
	return tlsVersionFlag(gotls.VersionTLS12, gotls.VersionTLS12, &cli.Prototype{
		Name:     "tlsv1.2",
		HelpText: "Use TLSv1.2",
		Category: tlsOptions,
	})
}

func SetTLSv1_3() cli.Action {
	return tlsVersionFlag(gotls.VersionTLS13, gotls.VersionTLS13, &cli.Prototype{
		Name:     "tlsv1.3",
		HelpText: "Use TLSv1.3",
		Category: tlsOptions,
	})
}

func SetInsecureSkipVerify(v ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "insecure-skip-verify",
			Aliases:  []string{"k", "insecure"},
			HelpText: "Whether to verify the server's certificate chain and host name.",
			Category: tlsOptions,
			EnvVars:  []string{"INSECURE_SKIP_VERIFY"},
			Options:  cli.ImpliedAction,
		},
		bind.Action(WithInsecureSkipVerify, bind.Exact(v...)),
		tagged,
	)
}

func SetCiphers(v ...CipherSuites) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "ciphers",
			HelpText: "List of SSL ciphers to use.  Not applicable to TLS 1.3",
			Category: tlsOptions,
		},
		bind.Action(WithCiphers, bind.Exact(v...)),
		tagged,
	)
}

func SetCurves(v ...CurveIDs) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "curves",
			HelpText: "TLS key exchange algorithms to request",
			Category: tlsOptions,
		},
		bind.Action(WithCurves, bind.Exact(v...)),
		tagged,
	)
}

func ListCiphers() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-ciphers",
			Value:    cli.Bool(),
			Options:  cli.Exits,
			HelpText: "List the cipher suites available and exit",
			Category: tlsOptions,
		},
		cli.At(cli.ActionTiming, cli.ActionOf(doListCiphers)),
		tagged,
	)
}

func ListCurves() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-curves",
			Value:    cli.Bool(),
			Options:  cli.Exits,
			HelpText: "List the key exchange algorithms and exit",
			Category: tlsOptions,
		},
		cli.At(cli.ActionTiming, cli.ActionOf(doListCurves)),
		tagged,
	)
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: ListCiphers()},
			{Uses: ListCurves()},
			{Uses: SetCACertFile()},
			{Uses: SetCACertPath()},
			{Uses: SetCiphers()},
			{Uses: SetClientCertFile()},
			{Uses: SetCurves()},
			{Uses: SetInsecureSkipVerify()},
			{Uses: SetKeyFile()},
			{Uses: SetNextProtos()},
			{Uses: SetServerName()},
			{Uses: SetTime()},
			{Uses: SetTLSv1()},
			{Uses: SetTLSv1_0()},
			{Uses: SetTLSv1_1()},
			{Uses: SetTLSv1_2()},
			{Uses: SetTLSv1_3()},
		}...))
}

func withBinding[V any](binder func(*Config, V) error, args []V) cli.Action {
	return bind.Call2(binder, bind.FromContext(FromContext), bind.Exact(args...))
}
