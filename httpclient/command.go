// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/cliutil"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

const (
	expectedOneArg = "expected 0 or 1 arg"

	dnsOptions      = "DNS options"
	networkOptions  = "Network interface options"
	requestOptions  = "Request options"
	responseOptions = "Response options"
	tlsOptions      = "TLS options"
)

var (
	tagged = cli.Data(SourceAnnotation())
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", "joe-cli-http/httpclient"
}

func (c *Client) Execute(ctx context.Context) error {
	return cli.Do(
		ctx,
		cli.Pipeline(
			FlagsAndArgs(),
			cli.Before(cli.Pipeline(
				registerFallbackFuncs(),
				cli.RegisterTemplate("HTTPTrace", outputTemplateText),
			)),
			ContextValue(c),
			Authenticators,
			PromptForCredentials(),
		),
	)
}

func ContextValue(c *Client) cli.Action {
	return cli.ContextValue(servicesKey, c)
}

func FetchAndPrint() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		_, err := Do(c)
		return err
	})
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: SetMethod()},
			{Uses: SetHeader()},
			{Uses: SetBody()},
			{Uses: SetBaseURL()},
			{Uses: SetURITemplateVar()},
			{Uses: SetURITemplateVars()},
			{Uses: SetBodyContent()},
			{Uses: SetFillValue()},
			{Uses: SetJSON()},
			{Uses: SetJSONContent()},
			{Uses: SetFollowRedirects()},
			{Uses: SetUserAgent()},
			{Uses: SetDialTimeout()},
			{Uses: SetIncludeResponseHeaders()},
			{Uses: SetOutputFile()},
			{Uses: SetNoOutput()},
			{Uses: SetIntegrity()},
			{Uses: SetDownload()},

			// TLS options
			{Uses: SetTLSv1()},
			{Uses: SetTLSv1_0()},
			{Uses: SetTLSv1_1()},
			{Uses: SetTLSv1_2()},
			{Uses: SetTLSv1_3()},
			{Uses: SetInsecureSkipVerify()},
			{Uses: SetCiphers()},
			{Uses: ListCiphers()},
			{Uses: SetCurves()},
			{Uses: ListCurves()},
			{Uses: SetClientCertFile()},
			{Uses: SetKeyFile()},
			{Uses: SetCACertFile()},
			{Uses: SetCACertPath()},
			{Uses: SetTime()},

			// DNS options
			{Uses: SetDNSInterface()},
			{Uses: SetPreferGo()},
			{Uses: SetStrictErrorsDNS()},

			{Uses: SetDialKeepAlive()},
			{Uses: SetDisableDialKeepAlive()},

			// Network interface options
			{Uses: SetBindAddress()},
			{Uses: SetInterface()},
			{Uses: ListInterfaces()},

			{Uses: SetVerbose()},
			{Uses: SetTraceLevel()},
			{Uses: SetServerName()},
			{Uses: SetNextProtos()},
			{Uses: SetRequestID()},
			{Uses: SetQueryString()},
			{Uses: SetWriteOut()},
			{Uses: SetWriteErr()},
			{Uses: SetStripComponents()},
			{Uses: SetFailFast()},

			// Auth
			{Uses: ListAuthenticators()},
			{Uses: SetUser()},
			{Uses: SetAuth()},
			{Uses: SetBasicAuth()},
		}...),

		cli.AddArg(&cli.Arg{
			Uses: SetURLValue(),
		}),
	)
}

func SetMethod(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       "method",
			Aliases:    []string{"X", "request"},
			UsageText:  "NAME",
			HelpText:   "Sets the request method to {NAME}",
			Options:    cli.ImpliedAction,
			Category:   requestOptions,
			Completion: cli.CompletionValues("GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE"),
		},
		withBinding((*Client).SetMethod, s),
		tagged,
	)
}

func SetHeader(s ...*HeaderValue) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "header",
			Aliases:  []string{"H"},
			HelpText: "Sets header to {NAME} and {VALUE}",
			Options:  cli.EachOccurrence,
			Category: requestOptions,
		},
		withBinding((*Client).SetHeader, s),
		tagged,
	)
}

func SetBody(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "body",
			HelpText: "Sets the raw content of the body of the request",
			Aliases:  []string{"data-raw"},
			Category: requestOptions,
			Options:  cli.AllowFileReference,
			Setup: cli.Setup{
				Uses: cli.Pipeline(
					cli.Implies("method", "POST"),
					cli.Implies("body-content", ContentTypeRaw.String()),
				),
			},
		},
		withBinding((*Client).SetBody, s),
		tagged,
	)
}

func SetBodyContent(s ...*ContentType) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "body-content",
			HelpText: "Sets the type of the body of the request: form, raw, urlencoded, multipart, json",
			Options:  cli.ImpliedAction,
			Category: requestOptions,
		},
		cli.Implies("method", "POST"),
		withBinding((*Client).setBodyContentHelper, s),
		tagged,
	)
}

func SetFillValue(s ...*cli.NameValue) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "fill",
			HelpText: "Fills a value in the body of the request or the query string",
			Aliases:  []string{"F"},
			Category: requestOptions,
			Options:  cli.EachOccurrence,
		},
		cli.Implies("method", "POST"),
		withBinding((*Client).SetFillValue, s),
		tagged,
	)
}

func SetJSON() cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Optional: true,
			Uses: &cli.Prototype{
				Name:     "json",
				HelpText: "Sets the Accept header to application/json",
				Value:    cli.Bool(),
				Category: requestOptions,
			},
			Action: setHTTPHeaderStatic("Accept", "application/json"),
		},
		tagged,
	)
}

func SetJSONContent() cli.Action {
	c := ContentTypeJSON
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "json-content",
			HelpText: "Sets the Content-Type header to application/json",
			Value:    cli.Bool(),
			Category: requestOptions,
		},
		SetBodyContent(&c),
		tagged,
	)
}

func SetFollowRedirects(s ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "follow-redirects",
			Aliases:  []string{"L", "location"},
			Options:  cli.No,
			HelpText: "Follow redirects in the Location header",
			Category: requestOptions,
		},
		withBinding((*Client).SetFollowRedirects, s),
		tagged,
	)
}

func SetUserAgent(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "user-agent",
			Aliases:  []string{"A"},
			HelpText: "Send the specified user-agent {NAME} to server",
			Category: requestOptions,
		},
		withBinding((*Client).SetUserAgent, s),
		tagged,
	)
}

func SetDialTimeout(s ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "dial-timeout",
			HelpText: "maximum amount of time a dial will wait for a connect to complete",
			Category: requestOptions,
		},
		withBinding((*Client).SetDialTimeout, s),
		tagged,
	)
}

func SetIncludeResponseHeaders(s ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "include",
			Aliases:  []string{"i"},
			HelpText: "Include response headers in the output",
			Category: responseOptions,
		},
		withBinding((*Client).SetIncludeResponseHeaders, s),
		tagged,
	)
}

func SetOutputFile(f ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "output",
			HelpText: "Download file to {FILE} instead of writing to stdout",
			Aliases:  []string{"o"},
			Category: responseOptions,
		},
		withBinding((*Client).SetOutputFile, f),
		tagged,
	)
}

func SetNoOutput(b ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "no-output",
			HelpText: "Don't write the response output to stdout",
			Category: responseOptions,
		},
		withBinding((*Client).SetNoOutput, b),
		tagged,
	)
}

func SetIntegrity(i ...Integrity) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "integrity",
			UsageText: "hash:digest",
			HelpText:  "Validate the integrity of the download",
			Category:  responseOptions,
		},
		withBinding((*Client).SetIntegrity, i),
		tagged,
	)
}

func SetDownload() cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Optional: true,
			Uses: &cli.Prototype{
				Name:     "download",
				HelpText: "Download file using the same name as the request path.  If specified a second time, also preserves the path structure",
				Aliases:  []string{"O", "remote-name"},
				Value:    new(bool),
				Category: responseOptions,
			},
			Action: func(c *cli.Context) error {
				switch c.Occurrences("") {
				case 1:
					FromContext(c).SetDownloadFile(PreserveRequestFile)
				case 2:
					FromContext(c).SetDownloadFile(PreserveRequestPath)
				default:
					return fmt.Errorf("too many occurrences of -O flag")
				}
				return nil
			},
		},
		tagged,
	)
}

func SetStripComponents(i ...int) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "strip-components",
			HelpText: "Remove the specified number of leading path elements when downloading files",
			Category: responseOptions,
		},
		withBinding((*Client).SetStripComponents, i),
		tagged,
	)
}

func SetFailFast(i ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "fail",
			Aliases:  []string{"f"},
			HelpText: "Fail fast with no output on HTTP errors",
			Category: responseOptions,
		},
		withBinding((*Client).SetFailFast, i),
		tagged,
	)
}

func SetURLValue(i ...*URLValue) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "url",
			HelpText: "Set the request URL",
			NArg:     cli.TakeUntilNextFlag,
			Options:  cli.EachOccurrence,
			Category: requestOptions,
		},
		withBinding((*Client).SetURLValue, i),
		tagged,
	)
}

func SetTLSv1() cli.Action {
	return tlsVersionFlag(tls.VersionTLS10, tls.VersionTLS13, &cli.Prototype{
		Name:     "tlsv1",
		HelpText: "Use TLSv1.0 or higher.  This is implied as this tool doesn't support SSLv3",
		Category: tlsOptions,
	})
}

func SetTLSv1_0() cli.Action {
	return tlsVersionFlag(tls.VersionTLS10, tls.VersionTLS10, &cli.Prototype{
		Name:     "tlsv1.0",
		HelpText: "Use TLSv1.0",
		Category: tlsOptions,
	})
}

func SetTLSv1_1() cli.Action {
	return tlsVersionFlag(tls.VersionTLS11, tls.VersionTLS11, &cli.Prototype{
		Name:     "tlsv1.1",
		HelpText: "Use TLSv1.1",
		Category: tlsOptions,
	})
}

func SetTLSv1_2() cli.Action {
	return tlsVersionFlag(tls.VersionTLS12, tls.VersionTLS12, &cli.Prototype{
		Name:     "tlsv1.2",
		HelpText: "Use TLSv1.2",
		Category: tlsOptions,
	})
}

func SetTLSv1_3() cli.Action {
	return tlsVersionFlag(tls.VersionTLS13, tls.VersionTLS13, &cli.Prototype{
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
		withBinding((*Client).SetInsecureSkipVerify, v),
		tagged,
	)
}

func SetCiphers(v ...*CipherSuites) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "ciphers",
			HelpText: "List of SSL ciphers to use.  Not applicable to TLS 1.3",
			Category: tlsOptions,
		},
		withBinding((*Client).SetCiphers, v),
		tagged,
	)
}

func SetCurves(v ...*CurveIDs) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "curves",
			HelpText: "TLS key exchange algorithms to request",
			Category: tlsOptions,
		},
		withBinding((*Client).SetCurves, v),
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

func SetDNSInterface(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       "dns-interface",
			HelpText:   "Use network {INTERFACE} by name or address for DNS requests",
			Category:   dnsOptions,
			Completion: completeInterfaces(),
		},
		withBinding((*Client).SetDNSInterface, s),
		tagged,
	)
}

func SetPreferGo() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "prefer-go",
			HelpText: "Whether Go's built-in DNS resolver is preferred",
			Setup:    dualSetup(cli.BindContext(FromContext, (*Client).SetPreferGoDialer)),
			Category: dnsOptions,
		},
		tagged,
	)
}

func SetDialKeepAlive(v ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "dial-keep-alive",
			HelpText: "Specifies the interval between keep-alive probes for an active network connection.",
			Category: dnsOptions,
		},
		withBinding((*Client).SetDialKeepAlive, v),
		tagged,
	)
}

func SetDisableDialKeepAlive() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "disable-dial-keep-alive",
			HelpText: "Disable dialer keep-alive probes",
			Setup:    dualSetup(cli.BindContext(FromContext, (*Client).SetDisableDialKeepAlive)),
			Category: dnsOptions,
		},
		tagged,
	)
}

func SetStrictErrorsDNS() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "strict-errors",
			HelpText: "When set, returns errors instead of partial results with the Go built-in DNS resolver.",
			Setup:    dualSetup(cli.BindContext(FromContext, (*Client).SetStrictErrorsDNS)),
			Category: dnsOptions,
		},
		tagged,
	)
}

func SetBindAddress(v ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "bind-address",
			UsageText: "HOSTNAME|IP",
			HelpText:  "Bind client TCP/IP connections to ADDRESS on the local machine",
			Category:  networkOptions,
		},
		withBinding((*Client).SetBindAddress, v),
		tagged,
	)
}

func SetInterface(v ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       "interface",
			HelpText:   "Use network {INTERFACE} by name or address to connect",
			Category:   networkOptions,
			Completion: completeInterfaces(),
		},
		withBinding((*Client).SetInterface, v),
		tagged,
	)
}

func ListInterfaces() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-interfaces",
			Value:    cli.Bool(),
			Options:  cli.Exits,
			HelpText: "List network interfaces and then exit",
			Setup: cli.Setup{
				Action: listInterfaces(),
			},
			Category: networkOptions,
		},
		tagged,
	)
}

func SetVerbose() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "verbose",
			Aliases:  []string{"v"},
			Value:    new(bool),
			HelpText: "Display verbose output; can be used multiple times to increase detail",
			Setup: cli.Setup{
				Action: func(c *cli.Context) {
					switch c.Occurrences("") {
					case 0:
					case 1:
						FromContext(c).SetTraceLevel(TraceOn)
					case 2:
						FromContext(c).SetTraceLevel(TraceVerbose)
					case 3:
						fallthrough
					default:
						FromContext(c).SetTraceLevel(TraceDebug)
					}
				},
			},
		},
		tagged,
	)
}

func SetBaseURL(name ...*URLValue) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "base",
			Aliases:  []string{"a"},
			HelpText: "Specify a base URL.  Can be used multiple times",
			Category: requestOptions,
		},
		withBinding((*Client).SetBaseURL, name),
		tagged,
	)
}

func SetURITemplateVar(v ...*uritemplates.Var) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "param",
			Aliases:  []string{"T"},
			HelpText: "Specify a value used to fill an RFC 6570 Level 4 URI template and parse the input URL as having template expressions",
			Value:    new(uritemplates.Var),
			Category: requestOptions,
			Options:  cli.EachOccurrence,
		},
		withBinding((*Client).SetURITemplateVar, v),
		tagged,
	)
}

func SetURITemplateVars(v ...uritemplates.Vars) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "params",
			Aliases:   []string{"t"},
			UsageText: "expr|@file",
			HelpText:  "Specify a template parameters using abbreviated syntax or from a JSON file",
			Value:     &uritemplates.Vars{},
			Options:   cli.EachOccurrence | cli.AllowFileReference,
		},
		withBinding((*Client).SetURITemplateVars, v),
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
		withBinding((*Client).SetCACertFile, path),
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
		withBinding((*Client).SetCACertPath, path),
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
		},
		withBinding((*Client).SetClientCertFile, path),
		tagged,
	)
}

func SetKeyFile(path ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "key",
			HelpText:  "Private key file (PEM format)",
			UsageText: "PATH",
			Category:  tlsOptions,
		},
		withBinding((*Client).SetKeyFile, path),
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
		withBinding((*Client).setTimeHelper, s),
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
		withBinding((*Client).SetServerName, s),
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
		withBinding((*Client).SetNextProtos, s),
		tagged,
	)
}

func SetRequestID(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "request-id",
			HelpText: "Sets or generates X-Request-ID header with optional {VALUE}",
			Options:  cli.Optional,
			Category: requestOptions,
		},
		withBinding((*Client).SetRequestID, s),
		tagged,
	)
}

func SetQueryString(s ...*cli.NameValue) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "query",
			Aliases:  []string{"Q"},
			HelpText: "Specify a {NAME} and {VALUE} to add to the query string",
			Category: requestOptions,
			Options:  cli.EachOccurrence,
		},
		withBinding((*Client).SetQueryString, s),
		tagged,
	)
}

func SetWriteOut(w ...Expr) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "write-out",
			HelpText: "Evaluate the expression and print out the result",
			Aliases:  []string{"w"},
			Category: requestOptions,
		},
		cli.BindContext(FromContext, (*Client).SetWriteOut, w...),
		tagged,
	)
}

func SetWriteErr(w ...Expr) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "write-err",
			HelpText: "Evaluate the expression and print out the result to stderr",
			Aliases:  []string{"W"},
			Category: requestOptions,
		},
		cli.BindContext(FromContext, (*Client).SetWriteErr, w...),
		tagged,
	)
}

func bind(fn func(*Client) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		return fn(FromContext(c))
	}
}

func setHTTPHeaderStatic(name, value string) cli.Action {
	return bind(func(s *Client) error {
		s.Request.Header.Set(name, value)
		return nil
	})
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
					s.TLSConfig().MinVersion = minVersion
					s.TLSConfig().MaxVersion = maxVersion
				}
				return nil
			},
		},
		tagged,
	)
}

func listInterfaces() cli.ActionFunc {
	return func(_ *cli.Context) error {
		eths, _ := net.Interfaces()
		for _, s := range eths {
			addrs, err := s.Addrs()
			if err != nil {
				fmt.Printf("%s\t%v\n", s.Name, err)
				continue
			}
			fmt.Print(s.Name)
			for i, a := range addrs {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("\t%s\t%s", a.Network(), a.String())
			}
			fmt.Println()
		}
		return nil
	}
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
		return cli.CompletionValues(values...).Complete(cc)
	}
}

func dualSetup(a cli.Action) cli.Setup {
	return cliutil.DualSetup(a)
}

func withBinding[V any](binder func(*Client, V) error, args []V) cli.Action {
	return cli.BindContext(FromContext, binder, args...)
}

func registerFallbackFuncs() cli.ActionFunc {
	return func(c *cli.Context) error {
		// Certain function names that control color need to be stubbed
		// if they have not been registered already by the color extension.
		// Build up a template and execute it to make sure all names are present.
		var sb strings.Builder
		for k := range funcs {
			fmt.Fprintf(&sb, "{{ %s }}\n", k)
		}

		if err := c.RegisterTemplate("_CheckForFunctions", sb.String()); err != nil {
			// The error occurs if functions are not present on registration
			for k, v := range funcs {
				c.RegisterTemplateFunc(k, v)
			}
		}
		return nil
	}
}
