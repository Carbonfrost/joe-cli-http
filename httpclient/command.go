package httpclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/Carbonfrost/joe-cli"
)

// Options provides the options for how the HTTP client works.  This is the entry
// point for setting up a command or app to contain HTTP client services in the context
type Options struct {
}

func (o *Options) Execute(c *cli.Context) error {
	return c.Do(
		FlagsAndArgs(),
		cli.RegisterTemplate("HTTPTrace", outputTemplateText),
		cli.ContextValue(servicesKey, newContextServices()),
	)
}

func FetchAndPrint() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		response, err := Do(c)
		if err != nil {
			return err
		}
		return response.CopyTo(c.Stdout, Services(c).IncludeHeaders)
	})
}

func FlagsAndArgs() cli.Action {
	const (
		dnsOptions     = "DNS options"
		networkOptions = "Network interface options"
		requestOptions = "Request options"
		tlsOptions     = "TLS options"
	)
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{
				Name:      "method",
				Aliases:   []string{"X", "request"},
				UsageText: "NAME",
				HelpText:  "Sets the request method to {NAME}",
				Uses:      cli.BindContext(Services, (*ContextServices).SetMethod),
				Category:  requestOptions,
			},
			{
				Name:     "header",
				Aliases:  []string{"H"},
				HelpText: "Sets header to {NAME} and {VALUE}",
				Uses:     cli.BindContext(Services, (*ContextServices).SetHeader),
				Category: requestOptions,
			},
			{
				Name:     "json",
				HelpText: "Sets the Accept header to application/json",
				Value:    cli.Bool(),
				Action:   setHTTPHeaderStatic("Accept", "application/json"),
				Category: requestOptions,
			},
			{
				Name:     "json-content",
				HelpText: "Sets the Content-Type header to application/json",
				Value:    cli.Bool(),
				Action:   setHTTPHeaderStatic("Content-Type", "application/json"),
				Category: requestOptions,
			},
			{
				Name:     "follow-redirects",
				Aliases:  []string{"L", "location"},
				Options:  cli.No,
				HelpText: "Follow redirects in the Location header",
				Uses:     cli.BindContext(Services, (*ContextServices).SetFollowRedirects),
				Category: requestOptions,
			},
			{
				Name:     "user-agent",
				Aliases:  []string{"A"},
				HelpText: "Send the specified user-agent {NAME} to server",
				Uses:     cli.BindContext(Services, (*ContextServices).SetUserAgent),
				Category: requestOptions,
			},
			{
				Name:     "include",
				Aliases:  []string{"i"},
				HelpText: "Include response headers in the output",
				Uses:     cli.BindContext(Services, (*ContextServices).SetIncludeHeaders),
				Category: requestOptions,
			},
			{
				Name:     "dial-timeout",
				HelpText: "maximum amount of time a dial will wait for a connect to complete",
				Uses:     cli.BindContext(Services, (*ContextServices).SetDialTimeout),
				Category: requestOptions,
			},
			{
				Name:     "tlsv1",
				HelpText: "Use TLSv1.0 or higher.  This is implied as this tool doesn't support SSLv3",
				Uses:     tlsVersionFlag(tls.VersionTLS10, tls.VersionTLS13),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.0",
				HelpText: "Use TLSv1.0",
				Uses:     tlsVersionFlag(tls.VersionTLS10, tls.VersionTLS10),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.1",
				HelpText: "Use TLSv1.1",
				Uses:     tlsVersionFlag(tls.VersionTLS11, tls.VersionTLS11),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.2",
				HelpText: "Use TLSv1.2",
				Uses:     tlsVersionFlag(tls.VersionTLS12, tls.VersionTLS12),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.3",
				HelpText: "Use TLSv1.3",
				Uses:     tlsVersionFlag(tls.VersionTLS13, tls.VersionTLS13),
				Category: tlsOptions,
			},
			{
				Name:     "insecure-skip-verify",
				Aliases:  []string{"k"},
				HelpText: "Whether to verify the server's certificate chain and host name.",
				Uses:     cli.BindContext(Services, (*ContextServices).SetInsecureSkipVerify),
				Category: tlsOptions,
			},
			{
				Name:     "ciphers",
				HelpText: "List of SSL ciphers to use.  Not applicable to TLS 1.3",
				Uses:     cli.BindContext(Services, (*ContextServices).SetCiphers),
				Category: tlsOptions,
			},
			{
				Name:     "list-ciphers",
				Value:    cli.Bool(),
				Options:  cli.Exits,
				HelpText: "List the cipher suites available and exit",
				Action:   doListCiphers,
				Category: tlsOptions,
			},
			{
				Name:     "dns-interface",
				HelpText: "Use network {INTERFACE} by name or address for DNS requests",
				Uses:     cli.BindContext(Services, (*ContextServices).SetDNSInterface),
				Category: dnsOptions,
			},
			{
				Name:     "prefer-go",
				HelpText: "Whether Go's built-in DNS resolver is preferred",
				Action:   cli.BindContext(Services, (*ContextServices).SetPreferGoDialer),
				Category: dnsOptions,
			},
			{
				Name:     "dial-keep-alive",
				HelpText: "Specifies the interval between keep-alive probes for an active network connection.",
				Uses:     cli.BindContext(Services, (*ContextServices).SetDialKeepAlive),
				Category: dnsOptions,
			},
			{
				Name:     "disable-dial-keep-alive",
				HelpText: "Disable dialer keep-alive probes",
				Uses:     cli.BindContext(Services, (*ContextServices).SetDisableDialKeepAlive),
				Category: dnsOptions,
			},
			{
				Name:     "strict-errors",
				HelpText: "When set, returns errors instead of partial results with the Go built-in DNS resolver.",
				Uses:     cli.BindContext(Services, (*ContextServices).SetStrictErrorsDNS),
				Category: dnsOptions,
			},
			{
				Name:     "interface",
				HelpText: "Use network {INTERFACE} by name or address to connect",
				Uses:     cli.BindContext(Services, (*ContextServices).SetInterface),
				Category: networkOptions,
			},
			{
				Name:     "list-interfaces",
				Value:    cli.Bool(),
				Options:  cli.Exits,
				HelpText: "List network interfaces and then exit",
				Action:   listInterfaces(),
				Category: networkOptions,
			},
			{
				Name:     "verbose",
				Aliases:  []string{"v"},
				Value:    new(bool),
				HelpText: "Display verbose output; can be used multiple times",
				Action: func(c *cli.Context) {
					switch c.Occurrences("") {
					case 0:
					case 1:
						Services(c).SetTraceLevel(TraceOn)
					case 2:
						Services(c).SetTraceLevel(TraceVerbose)
					case 3:
						fallthrough
					default:
						Services(c).SetTraceLevel(TraceDebug)
					}
				},
			},
		}...),

		cli.AddArg(&cli.Arg{
			Name:  "url",
			Value: new(URLValue),
			Uses:  cli.BindContext(Services, (*ContextServices).SetURL),
		}),
	)
}

func newContextServices() *ContextServices {
	h := &ContextServices{
		dnsDialer: &net.Dialer{},
		Request: &http.Request{
			Method: "GET",
			Header: make(http.Header),
		},
	}
	h.dialer = &net.Dialer{
		Resolver: &net.Resolver{
			Dial: h.dnsDialer.DialContext,
		},
	}
	h.tlsConfig = &tls.Config{}
	h.Client = &http.Client{
		Transport: &traceableTransport{
			Transport: &http.Transport{
				DialContext:     h.dialer.DialContext,
				DialTLSContext:  h.dialer.DialContext,
				TLSClientConfig: h.tlsConfig,
			},
		},
	}
	return h
}

func bind(fn func(*ContextServices) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		return fn(Services(c))
	}
}

func setHTTPHeaderStatic(name, value string) cli.Action {
	return bind(func(s *ContextServices) error {
		s.Request.Header.Set(name, value)
		return nil
	})
}

func tlsVersionFlag(min, max uint16) cli.Action {
	return cli.Prototype{
		Value: new(bool),
		Setup: cli.Setup{
			Action: func(c *cli.Context) error {
				s := Services(c)
				if c.Bool("") {
					s.TLSConfig().MinVersion = min
					s.TLSConfig().MaxVersion = max
				}
				return nil
			},
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

func resolveInterface(v string) (*net.TCPAddr, error) {
	var resolveTCP = func(value string) (*net.TCPAddr, error) {
		return net.ResolveTCPAddr("tcp", net.JoinHostPort(value, "0"))
	}
	var resolveIPNet = func(value string) (*net.TCPAddr, error) {
		ip, _, err := net.ParseCIDR(value)
		if err != nil {
			return nil, err
		}
		return resolveTCP(ip.String())
	}

	ip := net.ParseIP(v)
	if ip != nil {
		return resolveTCP(ip.String())
	}
	eth, err := net.InterfaceByName(v)
	if err != nil {
		return nil, err
	}
	addrs, err := eth.Addrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		if a.Network() == "ip+net" {
			return resolveIPNet(a.String())
		}
	}
	return nil, errors.New("failed to resolve " + v)
}
