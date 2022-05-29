package httpclient

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

type contextKey string

const servicesKey contextKey = "httpclient_services"

type Client struct {
	Client            *http.Client
	Request           *http.Request
	IncludeHeaders    bool
	InterfaceResolver InterfaceResolver
	downloader        Downloader

	dialer    *net.Dialer
	dnsDialer *net.Dialer
	tlsConfig *tls.Config
}

func New() *Client {
	h := &Client{
		InterfaceResolver: &defaultResolver{},
		dnsDialer:         &net.Dialer{},
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
				Proxy:           http.ProxyFromEnvironment,
			},
		},
	}
	return h
}

func Do(c *cli.Context) (*Response, error) {
	return FromContext(c).Do(c)
}

// FromContext obtains the client stored in the context
func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
}

func (c *Client) SetTraceLevel(v TraceLevel) error {
	c.Client.Transport.(*traceableTransport).level = v
	return nil
}

func (c *Client) setTraceLevelHelper(v *TraceLevel) error {
	return c.SetTraceLevel(*v)
}

func (c *Client) Do(ctx context.Context) (*Response, error) {
	c.Request = c.Request.WithContext(ctx)

	resp, err := c.Client.Do(c.Request)
	if err != nil {
		return nil, err
	}
	return &Response{
		Response: resp,
	}, nil
}

func (c *Client) TLSConfig() *tls.Config {
	return c.tlsConfig
}

func (c *Client) Dialer() *net.Dialer {
	return c.dialer
}

func (c *Client) DNSDialer() *net.Dialer {
	return c.dnsDialer
}

func (c *Client) SetMethod(s string) error {
	c.Request.Method = s
	return nil
}

func (c *Client) SetFollowRedirects(value bool) error {
	if value {
		c.Client.CheckRedirect = nil // default policy to follow 10 times
		return nil
	}

	// Follow no redirects
	c.Client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return nil
}

func (c *Client) SetUserAgent(value string) error {
	c.Request.Header.Set("User-Agent", value)
	return nil
}

func (c *Client) SetURL(u *URLValue) error {
	c.Request.URL = &u.URL
	c.Request.Host = u.Host
	return nil
}

func (c *Client) SetIncludeHeaders(v bool) error {
	c.IncludeHeaders = v
	return nil
}

func (c *Client) SetInsecureSkipVerify(v bool) error {
	c.TLSConfig().InsecureSkipVerify = v
	return nil
}

func (c *Client) SetCiphers(ids *CipherSuites) error {
	c.TLSConfig().CipherSuites = []uint16(*ids)
	return nil
}

func (c *Client) SetPreferGoDialer(v bool) error {
	c.Dialer().Resolver.PreferGo = v
	return nil
}

func (c *Client) SetStrictErrorsDNS(v bool) error {
	c.Dialer().Resolver.StrictErrors = v
	return nil
}

func (c *Client) SetDisableDialKeepAlive(v bool) error {
	if v {
		c.Dialer().KeepAlive = time.Duration(-1)
	}
	return nil
}

func (c *Client) SetHeader(n *cli.NameValue) error {
	name, value := n.Name, n.Value
	// If a colon was used, then assume the syntax Header:Value was used.
	if strings.Contains(name, ":") && value == "true" {
		args := strings.SplitN(name, ":", 2)
		name = args[0]
		value = args[1]
	}
	c.Request.Header.Set(name, value)
	return nil
}

func (c *Client) SetInterface(value string) error {
	addr, err := c.resolveInterface(value)
	if err != nil {
		return err
	}
	c.Dialer().LocalAddr = addr
	return nil
}

func (c *Client) SetDNSInterface(value string) error {
	if value == "" {
		return nil
	}
	addr, err := c.resolveInterface(value)
	if err != nil {
		return err
	}
	c.DNSDialer().LocalAddr = addr
	return nil
}

func (c *Client) SetDialTimeout(v time.Duration) error {
	c.Dialer().Timeout = v
	return nil
}

func (c *Client) SetDialKeepAlive(v time.Duration) error {
	c.Dialer().KeepAlive = v
	return nil
}

func (c *Client) SetDownloadFile(v Downloader) error {
	c.downloader = v
	return nil
}

func (c *Client) resolveInterface(v string) (*net.TCPAddr, error) {
	return c.InterfaceResolver.Resolve(v)
}

func (c *Client) openDownload(resp *Response) (io.Writer, error) {
	if c.downloader == nil {
		return os.Stdout, nil
	}
	return c.downloader.OpenDownload(resp)
}
