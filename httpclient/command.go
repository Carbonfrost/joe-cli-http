package httpclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/cliutil"
)

// Options provides the options for how the HTTP client works.  This is the entry
// point for setting up a command or app to contain an HTTP client in the context.
type Options struct {
}

const (
	expectedOneArg = "expected 0 or 1 arg"

	dnsOptions      = "DNS options"
	networkOptions  = "Network interface options"
	requestOptions  = "Request options"
	responseOptions = "Response options"
	tlsOptions      = "TLS options"
)

func (o *Options) Execute(c *cli.Context) error {
	return c.Do(
		FlagsAndArgs(),
		cli.RegisterTemplate("HTTPTrace", outputTemplateText),
		cli.ContextValue(servicesKey, New()),
		Authenticators,
		PromptForCredentials(),
	)
}

func FetchAndPrint() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		response, err := Do(c)
		if err != nil {
			return err
		}

		output, err := FromContext(c).openDownload(response)
		if err != nil {
			return err
		}

		return response.CopyTo(output, FromContext(c).IncludeHeaders)
	})
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: SetMethod()},
			{Uses: SetHeader()},
			{Uses: SetBody()},
			{Uses: SetBodyContent()},
			{Uses: SetJSON()},
			{Uses: SetJSONContent()},
			{Uses: SetFollowRedirects()},
			{Uses: SetUserAgent()},
			{Uses: SetDialTimeout()},
			{Uses: SetIncludeResponseHeaders()},
			{Uses: SetOutputFile()},
			{Uses: SetDownload()},
			{Uses: SetTLSv1()},
			{Uses: SetTLSv1_0()},
			{Uses: SetTLSv1_1()},
			{Uses: SetTLSv1_2()},
			{Uses: SetTLSv1_3()},
			{Uses: SetInsecureSkipVerify()},
			{Uses: SetCiphers()},
			{Uses: ListCiphers()},
			{Uses: SetDNSInterface()},
			{Uses: SetPreferGo()},
			{Uses: SetDialKeepAlive()},
			{Uses: SetDisableDialKeepAlive()},
			{Uses: SetStrictErrorsDNS()},
			{Uses: SetInterface()},
			{Uses: ListInterfaces()},
			{Uses: SetVerbose()},
			{Uses: SetTraceLevel()},
		}...),

		cli.AddArg(&cli.Arg{
			Name:  "url",
			Value: new(URLValue),
			Uses:  cli.BindContext(FromContext, (*Client).SetURL),
		}),

		cli.AddFlags([]*cli.Flag{
			{Uses: ListAuthenticators()},
			{Uses: SetUser()},
			{Uses: SetAuth()},
			{Uses: SetBasicAuth()},
		}...),
	)
}

func SetMethod(s ...string) cli.Action {
	switch len(s) {
	case 0:
		return &cli.Prototype{
			Name:      "method",
			Aliases:   []string{"X", "request"},
			UsageText: "NAME",
			HelpText:  "Sets the request method to {NAME}",
			Setup: cli.Setup{
				Uses: cli.BindContext(FromContext, (*Client).SetMethod),
			},
			Options:  cli.ImpliedAction,
			Category: requestOptions,
		}
	case 1:
		return cli.BindContext(FromContext, (*Client).SetMethod, s[0])
	default:
		panic(expectedOneArg)
	}

}

func SetHeader(s ...*HeaderValue) cli.Action {
	switch len(s) {
	case 0:
		return &cli.Prototype{
			Name:     "header",
			Aliases:  []string{"H"},
			HelpText: "Sets header to {NAME} and {VALUE}",
			Setup: cli.Setup{
				Uses: cli.BindContext(FromContext, (*Client).SetHeader),
			},
			Options:  cli.EachOccurrence,
			Category: requestOptions,
		}
	case 1:
		return cli.BindContext(FromContext, (*Client).SetHeader, s[0])
	default:
		panic(expectedOneArg)
	}
}

func SetBody(s ...string) cli.Action {
	return createFlag((*Client).SetBody, s, &cli.Prototype{
		Name:     "body",
		HelpText: "Sets the raw content of the body of the request",
		Aliases:  []string{"data-raw"},
		Category: requestOptions,
		Options:  cli.AllowFileReference,
	}, cli.Implies("method", "POST"),
		cli.Implies("body-content", ContentTypeRaw.String()),
	)
}

func SetBodyContent(s ...*ContentType) cli.Action {
	return createFlag((*Client).setBodyContentHelper, s, &cli.Prototype{
		Name:     "body-content",
		HelpText: "Sets the type of the body of the request: form, raw, urlencoded, multipart, json",
		Options:  cli.ImpliedAction,
		Category: requestOptions,
	})
}

func SetJSON() cli.Action {
	return cli.Setup{
		Optional: true,
		Uses: &cli.Prototype{
			Name:     "json",
			HelpText: "Sets the Accept header to application/json",
			Value:    cli.Bool(),
			Category: requestOptions,
		},
		Action: setHTTPHeaderStatic("Accept", "application/json"),
	}
}

func SetJSONContent() cli.Action {
	return cli.Setup{
		Optional: true,
		Uses: &cli.Prototype{
			Name:     "json-content",
			HelpText: "Sets the Content-Type header to application/json",
			Value:    cli.Bool(),
			Category: requestOptions,
		},
		Action: setHTTPHeaderStatic("Content-Type", "application/json"),
	}
}

func SetFollowRedirects(s ...bool) cli.Action {
	return createFlag((*Client).SetFollowRedirects, s, &cli.Prototype{
		Name:     "follow-redirects",
		Aliases:  []string{"L", "location"},
		Options:  cli.No,
		HelpText: "Follow redirects in the Location header",
		Category: requestOptions,
	})
}

func SetUserAgent(s ...string) cli.Action {
	return createFlag((*Client).SetUserAgent, s, &cli.Prototype{
		Name:     "user-agent",
		Aliases:  []string{"A"},
		HelpText: "Send the specified user-agent {NAME} to server",
		Category: requestOptions,
	})
}

func SetDialTimeout(s ...time.Duration) cli.Action {
	return createFlag((*Client).SetDialTimeout, s, &cli.Prototype{
		Name:     "dial-timeout",
		HelpText: "maximum amount of time a dial will wait for a connect to complete",
		Category: requestOptions,
	})
}

func SetIncludeResponseHeaders(s ...bool) cli.Action {
	return createFlag((*Client).SetIncludeHeaders, s, &cli.Prototype{
		Name:     "include",
		Aliases:  []string{"i"},
		HelpText: "Include response headers in the output",
		Category: responseOptions,
	})
}

func SetOutputFile(f ...*cli.File) cli.Action {
	return createFlag((*Client).setOutputFileHelper, f, &cli.Prototype{
		Name:     "output",
		HelpText: "Download file to {FILE} instead of writing to stdout",
		Aliases:  []string{"o"},
		Value:    new(cli.File),
		Category: responseOptions,
	})
}

func SetDownload() cli.Action {
	return cli.Setup{
		Optional: true,
		Uses: &cli.Prototype{
			Name:     "download",
			HelpText: "Download file using the same name as the request path.  If specified a second time, also preserves the path structure",
			Aliases:  []string{"O", "remote-name"},
			Value:    new(bool),
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
	}
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
	return createFlag((*Client).SetInsecureSkipVerify, v, &cli.Prototype{
		Name:     "insecure-skip-verify",
		Aliases:  []string{"k", "insecure"},
		HelpText: "Whether to verify the server's certificate chain and host name.",
		Category: tlsOptions,
	})
}

func SetCiphers(v ...*CipherSuites) cli.Action {
	return createFlag((*Client).SetCiphers, v, &cli.Prototype{
		Name:     "ciphers",
		HelpText: "List of SSL ciphers to use.  Not applicable to TLS 1.3",
		Category: tlsOptions,
	})
}

func ListCiphers() cli.Action {
	return &cli.Prototype{
		Name:     "list-ciphers",
		Value:    cli.Bool(),
		Options:  cli.Exits,
		HelpText: "List the cipher suites available and exit",
		Category: tlsOptions,
		Setup: cli.Setup{
			Action: doListCiphers,
		},
	}
}

func SetDNSInterface(s ...string) cli.Action {
	return createFlag((*Client).SetDNSInterface, s, &cli.Prototype{
		Name:     "dns-interface",
		HelpText: "Use network {INTERFACE} by name or address for DNS requests",
		Category: dnsOptions,
	})
}

func SetPreferGo() cli.Action {
	return &cli.Prototype{
		Name:     "prefer-go",
		HelpText: "Whether Go's built-in DNS resolver is preferred",
		Setup:    dualSetup(cli.BindContext(FromContext, (*Client).SetPreferGoDialer)),
		Category: dnsOptions,
	}
}

func SetDialKeepAlive(v ...time.Duration) cli.Action {
	return createFlag((*Client).SetDialKeepAlive, v, &cli.Prototype{
		Name:     "dial-keep-alive",
		HelpText: "Specifies the interval between keep-alive probes for an active network connection.",
		Category: dnsOptions,
	})
}

func SetDisableDialKeepAlive() cli.Action {
	return &cli.Prototype{
		Name:     "disable-dial-keep-alive",
		HelpText: "Disable dialer keep-alive probes",
		Setup:    dualSetup(cli.BindContext(FromContext, (*Client).SetDisableDialKeepAlive)),
		Category: dnsOptions,
	}
}

func SetStrictErrorsDNS() cli.Action {
	return &cli.Prototype{
		Name:     "strict-errors",
		HelpText: "When set, returns errors instead of partial results with the Go built-in DNS resolver.",
		Setup:    dualSetup(cli.BindContext(FromContext, (*Client).SetStrictErrorsDNS)),
		Category: dnsOptions,
	}
}

func SetInterface(v ...string) cli.Action {
	return createFlag((*Client).SetInterface, v, &cli.Prototype{
		Name:     "interface",
		HelpText: "Use network {INTERFACE} by name or address to connect",
		Category: networkOptions,
	})
}

func ListInterfaces() cli.Action {
	return &cli.Prototype{
		Name:     "list-interfaces",
		Value:    cli.Bool(),
		Options:  cli.Exits,
		HelpText: "List network interfaces and then exit",
		Setup: cli.Setup{
			Action: listInterfaces(),
		},
		Category: networkOptions,
	}
}

func SetVerbose() cli.Action {
	return &cli.Prototype{
		Name:     "verbose",
		Aliases:  []string{"v"},
		Value:    new(bool),
		HelpText: "Display verbose output; can be used multiple times",
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
	}
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

func tlsVersionFlag(min, max uint16, proto *cli.Prototype) cli.Action {
	return cli.Setup{
		Uses: cli.Pipeline(
			&cli.Prototype{
				Value: new(bool),
			},
			proto,
		),
		Action: func(c *cli.Context) error {
			s := FromContext(c)
			if c.Bool("") {
				s.TLSConfig().MinVersion = min
				s.TLSConfig().MaxVersion = max
			}
			return nil
		},
	}
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

func createFlag[V any](binder func(*Client, V) error, args []V, proto *cli.Prototype, uses ...cli.Action) cli.Action {
	return cliutil.FlagBinding(FromContext, binder, args, proto, uses...)
}

func dualSetup(a cli.Action) cli.Setup {
	return cliutil.DualSetup(a)
}
