package httpclient

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

type contextKey string

const servicesKey contextKey = "httpclient_services"

var (
	defaultUserAgent string = "Go-http-client/1.1 wig/" + build.Version
)

type Client struct {
	Client            *http.Client
	Request           *http.Request
	IncludeHeaders    bool
	InterfaceResolver InterfaceResolver
	BodyContent       Content
	UserInfo          *UserInfo
	downloader        Downloader

	dialer         *net.Dialer
	dnsDialer      *net.Dialer
	tlsConfig      *tls.Config
	auth           Authenticator
	authMiddleware []func(Authenticator) Authenticator
}

// Option is an option to configure the client
type Option func(*Client)

func New(options ...Option) *Client {
	h := &Client{
		InterfaceResolver: &defaultResolver{},
		dnsDialer:         &net.Dialer{},
		Request: &http.Request{
			Method: "GET",
			Header: http.Header{
				"User-Agent": []string{defaultUserAgent},
			},
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

func wrapReader(r io.Reader) io.ReadCloser {
	if c, ok := r.(io.ReadCloser); ok {
		return c
	}
	return io.NopCloser(r)
}

func (c *Client) Do(ctx context.Context) (*Response, error) {
	c.Request = c.Request.WithContext(ctx)

	// Apply additional setup to request
	if c.BodyContent != nil {
		c.Request.Body = wrapReader(c.BodyContent.Read())
	}
	auth := c.Auth()
	for _, a := range c.authMiddleware {
		auth = a(auth)
	}
	err := auth.Authenticate(c.Request, c.UserInfo)

	if err != nil {
		return nil, err
	}

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

func (c *Client) SetURL(u *url.URL) error {
	c.Request.URL = u
	c.Request.Host = u.Host
	return nil
}

func (c *Client) SetURLValue(u *URLValue) error {
	return c.SetURL(&u.URL)
}

func (c *Client) SetIncludeHeaders(v bool) error {
	c.IncludeHeaders = v
	return nil
}

func (c *Client) setOutputFileHelper(f *cli.File) error {
	c.SetDownloadFile(&directAdapter{f})
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

func (c *Client) SetHeader(n *HeaderValue) error {
	c.Request.Header.Add(n.Name, n.Value)
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

func (c *Client) SetBody(body string) error {
	c.BodyContent = NewRawContent([]byte(body))
	return nil
}

func (c *Client) setBodyContentHelper(name *ContentType) error {
	c.BodyContent = NewContent(*name)
	return nil
}

func (c *Client) SetBodyContent(bodyContent Content) error {
	c.BodyContent = bodyContent
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

func (c *Client) SetAuth(auth Authenticator) error {
	c.auth = auth
	return nil
}

func (c *Client) setAuthenticatorHelper(a *provider.Value) error {
	args := a.Args.(*map[string]string)
	res, err := NewAuthenticator(a.Name, *args)
	if err != nil {
		return err
	}
	c.auth = res
	return nil
}

func (c *Client) setAuthModeHelper(auth AuthMode) error {
	return c.SetAuth(auth)
}

func (c *Client) SetUser(user *UserInfo) error {
	c.UserInfo = user
	return nil
}

func (c *Client) Auth() Authenticator {
	if c.auth == nil {
		c.auth = NoAuth
	}
	return c.auth
}

func (c *Client) UseAuthMiddleware(fn func(Authenticator) Authenticator) {
	c.authMiddleware = append(c.authMiddleware, fn)
}
