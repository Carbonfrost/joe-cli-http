package httpclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

// Options provides the options for how the HTTP client works.  This is the entry
// point for setting up a command or app to contain HTTP client services in the context
type Options struct {
}

func (o *Options) Execute(c *cli.Context) error {
	return c.Do(
		FlagsAndArgs(),
		cli.ContextValue(servicesKey, newContextServices()),
	)
}

func FetchAndPrint() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		response, err := Do(c)
		if err != nil {
			return err
		}
		return response.CopyTo(c.Stdout)
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
				Name:     "method",
				Aliases:  []string{"X", "request"},
				HelpText: "Sets the request method",
				Value:    cli.String(),
				Action:   setHTTPMethod(),
				Category: requestOptions,
			},
			{
				Name:     "header",
				Aliases:  []string{"H"},
				HelpText: "Sets header to {NAME} and {VALUE}",
				Value:    &cli.NameValue{},
				Action:   setHTTPHeader(),
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
				Value:    cli.Bool(),
				Action:   followRedirects(),
				Category: requestOptions,
			},
			{
				Name:     "user-agent",
				Aliases:  []string{"A"},
				HelpText: "Send the specified user-agent {NAME} to server",
				Action:   setUserAgent(),
				Category: requestOptions,
			},
			{
				Name:     "include",
				Aliases:  []string{"i"},
				Value:    cli.Bool(),
				HelpText: "Include response headers in the output",
				Action:   setIncludeHeaders(),
				Category: requestOptions,
			},
			{
				Name:     "dial-timeout",
				Value:    cli.Duration(),
				HelpText: "maximum amount of time a dial will wait for a connect to complete",
				Action:   setDialTimeout(),
				Category: requestOptions,
			},
			{
				Name:     "tlsv1",
				HelpText: "Use TLSv1.0 or higher.  This is implied as this tool doesn't support SSLv3",
				Value:    cli.Bool(),
				Action:   setTLSVersion(tls.VersionTLS10, tls.VersionTLS13),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.0",
				HelpText: "Use TLSv1.0",
				Value:    cli.Bool(),
				Action:   setTLSVersion(tls.VersionTLS10, tls.VersionTLS10),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.1",
				HelpText: "Use TLSv1.1",
				Value:    cli.Bool(),
				Action:   setTLSVersion(tls.VersionTLS11, tls.VersionTLS11),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.2",
				HelpText: "Use TLSv1.2",
				Value:    cli.Bool(),
				Action:   setTLSVersion(tls.VersionTLS12, tls.VersionTLS12),
				Category: tlsOptions,
			},
			{
				Name:     "tlsv1.3",
				HelpText: "Use TLSv1.3",
				Value:    cli.Bool(),
				Action:   setTLSVersion(tls.VersionTLS13, tls.VersionTLS13),
				Category: tlsOptions,
			},
			{
				Name:     "insecure-skip-verify",
				Aliases:  []string{"k"},
				Value:    cli.Bool(),
				HelpText: "Whether to verify the server's certificate chain and host name.",
				Action:   insecureSkipVerify(),
				Category: tlsOptions,
			},
			{
				Name:     "ciphers",
				Value:    &CipherSuites{},
				HelpText: "List of SSL ciphers to use.  Not applicable to TLS 1.3",
				Action:   setCiphers(),
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
				Value:    cli.String(),
				HelpText: "Use network {INTERFACE} by name or address for DNS requests",
				Action:   setDNSInterface(),
				Category: dnsOptions,
			},
			{
				Name:     "prefer-go",
				Value:    cli.Bool(),
				HelpText: "Whether Go's built-in DNS resolver is preferred",
				Action:   setPreferGoDialer(),
				Category: dnsOptions,
			},
			{
				Name:     "dial-keep-alive",
				Value:    cli.Duration(),
				HelpText: "Specifies the interval between keep-alive probes for an active network connection.",
				Action:   setDialKeepAlive(),
				Category: dnsOptions,
			},
			{
				Name:     "disable-dial-keep-alive",
				Value:    cli.Bool(),
				HelpText: "Disable dialer keep-alive probes",
				Action:   disableDialKeepAlive(),
				Category: dnsOptions,
			},
			{
				Name:     "strict-errors",
				Value:    cli.Bool(),
				HelpText: "When set, returns errors instead of partial results with the Go built-in DNS resolver.",
				Action:   setStrictErrorsDNS(),
				Category: dnsOptions,
			},
			{
				Name:     "interface",
				Value:    cli.String(),
				HelpText: "Use network {INTERFACE} by name or address to connect",
				Action:   setInterface(),
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
			Name:   "url",
			Value:  cli.URL(),
			Action: setURL(),
		}),
	)
}

func newContextServices() *ContextServices {
	h := &ContextServices{
		DNSDialer: &net.Dialer{},
		Request: &http.Request{
			Method: "GET",
			Header: make(http.Header),
		},
	}
	h.Dialer = &net.Dialer{
		Resolver: &net.Resolver{
			Dial: h.DNSDialer.DialContext,
		},
	}
	h.Client = &http.Client{
		Transport: &traceableTransport{
			Transport: &http.Transport{
				DialContext:     h.Dialer.DialContext,
				DialTLSContext:  h.Dialer.DialContext,
				TLSClientConfig: &tls.Config{},
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

func bindBoolean(fn func(*ContextServices, bool) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		return fn(Services(c), c.Bool(""))
	}
}

func bindDuration(fn func(*ContextServices, time.Duration) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		return fn(Services(c), c.Duration(""))
	}
}

func bindString(fn func(*ContextServices, string) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		return fn(Services(c), c.String(""))
	}
}

func bindNameValue(fn func(*ContextServices, string, string) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		nv := c.Value("").(*cli.NameValue)
		return fn(Services(c), nv.Name, nv.Value)
	}
}

func setHTTPHeader() cli.Action {
	return bindNameValue(func(s *ContextServices, name string, value string) error {
		// If a colon was used, then assume the syntax Header:Value was used.
		if strings.Contains(name, ":") && value == "true" {
			args := strings.SplitN(name, ":", 2)
			name = args[0]
			value = args[1]
		}
		s.Request.Header.Set(name, value)
		return nil
	})
}

func setHTTPHeaderStatic(name, value string) cli.Action {
	return bind(func(s *ContextServices) error {
		s.Request.Header.Set(name, value)
		return nil
	})
}

func setUserAgent() cli.Action {
	return bindString(func(s *ContextServices, value string) error {
		s.Request.Header.Set("User-Agent", value)
		return nil
	})
}

func setURL() cli.ActionFunc {
	return func(c *cli.Context) error {
		u := c.URL("")
		Services(c).Request.URL = u
		Services(c).Request.Host = u.Host
		return nil
	}
}

func setHTTPMethod() cli.Action {
	return bindString(func(s *ContextServices, v string) error {
		s.Request.Method = v
		return nil
	})
}

func followRedirects() cli.Action {
	return bindBoolean(func(s *ContextServices, value bool) error {
		if value {
			s.Client.CheckRedirect = nil // default policy to follow 10 times
			return nil
		}

		// Follow no redirects
		s.Client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	})
}

func setTLSVersion(min, max uint16) cli.Action {
	return bindBoolean(func(s *ContextServices, v bool) error {
		if v {
			s.tlsConfig().MinVersion = min
			s.tlsConfig().MaxVersion = max
		}
		return nil
	})
}

func setIncludeHeaders() cli.Action {
	return bindBoolean(func(s *ContextServices, v bool) error {
		s.IncludeHeaders = v
		return nil
	})
}

func insecureSkipVerify() cli.Action {
	return bindBoolean(func(s *ContextServices, v bool) error {
		s.tlsConfig().InsecureSkipVerify = v
		return nil
	})
}

func setCiphers() cli.ActionFunc {
	return func(c *cli.Context) error {
		ids := c.Value("").(*CipherSuites)
		Services(c).tlsConfig().CipherSuites = []uint16(*ids)
		return nil
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

func setInterface() cli.Action {
	return bindString(func(s *ContextServices, value string) error {
		addr, err := resolveInterface(value)
		if err != nil {
			return err
		}
		s.Dialer.LocalAddr = addr
		return nil
	})
}

func setDNSInterface() cli.Action {
	return bindString(func(s *ContextServices, value string) error {
		addr, err := resolveInterface(value)
		if err != nil {
			return err
		}
		s.DNSDialer.LocalAddr = addr
		return nil
	})
}

func setPreferGoDialer() cli.Action {
	return bindBoolean(func(s *ContextServices, v bool) error {
		s.Dialer.Resolver.PreferGo = v
		return nil
	})
}

func setStrictErrorsDNS() cli.Action {
	return bindBoolean(func(s *ContextServices, v bool) error {
		s.Dialer.Resolver.StrictErrors = v
		return nil
	})
}

func disableDialKeepAlive() cli.Action {
	return bindBoolean(func(s *ContextServices, v bool) error {
		if v {
			s.Dialer.KeepAlive = time.Duration(-1)
		}
		return nil
	})
}

func setDialTimeout() cli.Action {
	return bindDuration(func(s *ContextServices, v time.Duration) error {
		s.Dialer.Timeout = v
		return nil
	})
}

func setDialKeepAlive() cli.Action {
	return bindDuration(func(s *ContextServices, v time.Duration) error {
		s.Dialer.KeepAlive = v
		return nil
	})
}
