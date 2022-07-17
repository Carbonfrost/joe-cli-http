package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

type contextKey string

const servicesKey contextKey = "httpclient_services"
const wigURL = "https://github.com/Carbonfrost/joe-cli-http/cmd/wig"

type Client struct {
	Client            *http.Client
	Request           *http.Request
	IncludeHeaders    bool
	InterfaceResolver InterfaceResolver
	BodyContent       Content
	UserInfo          *UserInfo
	LocationResolver  LocationResolver

	downloader     Downloader
	dialer         *net.Dialer
	dnsDialer      *net.Dialer
	tlsConfig      *tls.Config
	auth           Authenticator
	authMiddleware []func(Authenticator) Authenticator
	bodyForm       []*cli.NameValue
}

var (
	impliedOptions = []Option{
		WithDefaultUserAgent(defaultUserAgent()),
	}
)

// Option is an option to configure the client
type Option func(*Client)

func New(options ...Option) *Client {
	h := &Client{
		InterfaceResolver: &defaultResolver{},
		dnsDialer:         &net.Dialer{},
		Request: &http.Request{
			Method: "GET",
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
	for _, o := range append(impliedOptions, options...) {
		o(h)
	}
	return h
}

func WithDefaultUserAgent(s string) Option {
	return func(c *Client) {
		if c.Request.Header == nil {
			c.Request.Header = http.Header{}
		}
		c.Request.Header.Set("User-Agent", s)
	}
}

func WithLocationResolver(r LocationResolver) Option {
	return func(c *Client) {
		c.LocationResolver = r
	}
}

func Do(c *cli.Context) ([]*Response, error) {
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

func (c *Client) Do(ctx context.Context) ([]*Response, error) {
	urls, err := c.ensureLocationResolver().Resolve(ctx)
	if err != nil {
		return nil, err
	}

	rsp := make([]*Response, 0, len(urls))
	for _, u := range urls {
		r, err := c.doOne(u, ctx)
		if err != nil {
			return rsp, err
		}
		rsp = append(rsp, r)
	}
	return rsp, nil
}

func (c *Client) doOne(u *url.URL, ctx context.Context) (*Response, error) {
	c.Request.URL = u
	c.Request.Host = u.Host
	c.Request = c.Request.WithContext(ctx)

	// Apply additional setup to request
	if len(c.bodyForm) > 0 {
		c.ensureBodyContent()
	}
	if c.BodyContent != nil {
		for _, k := range c.bodyForm {
			err := c.BodyContent.Set(k.Name, k.Name)
			if err != nil {
				return nil, err
			}
		}
		if c.Request.Header.Get("Content-Type") == "" {
			if ct := c.BodyContent.ContentType(); ct != "" {
				c.Request.Header.Set("Content-Type", ct)
			}
		}
		c.Request.Body = wrapReader(c.BodyContent.Read())
	}
	err := c.applyAuth()
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

func (c *Client) applyAuth() error {
	auth := c.Auth()
	for _, a := range c.authMiddleware {
		auth = a(auth)
	}
	return auth.Authenticate(c.Request, c.UserInfo)
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
	c.Request.Method = strings.ToUpper(s)
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

func (c *Client) SetBaseURL(u *URLValue) error {
	uu, err := u.URL()
	if err != nil {
		return err
	}
	return c.ensureLocationResolver().SetBase(uu)
}

func (c *Client) SetURL(u *url.URL) error {
	return c.ensureLocationResolver().Add(u.String())
}

func (c *Client) SetURLValue(u *URLValue) error {
	return c.ensureLocationResolver().Add(u.String())
}

func (c *Client) SetURITemplateVar(v *uritemplates.Var) error {
	return c.ensureLocationResolver().AddVar(v)
}

func (c *Client) SetURITemplateVars(v uritemplates.Vars) error {
	for _, item := range v.Items() {
		err := c.ensureLocationResolver().AddVar(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ensureLocationResolver() LocationResolver {
	if c.LocationResolver == nil {
		c.LocationResolver = NewDefaultLocationResolver()
	}
	return c.LocationResolver
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

func (c *Client) ensureBodyContent() Content {
	if c.BodyContent == nil {
		c.BodyContent = &FormDataContent{}
	}
	return c.BodyContent
}

func (c *Client) SetBodyContent(bodyContent Content) error {
	c.BodyContent = bodyContent
	return nil
}

func (c *Client) SetFillValue(v *cli.NameValue) error {
	c.bodyForm = append(c.bodyForm, v)
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

func (o Option) Execute(c *cli.Context) error {
	o(FromContext(c))
	return nil
}

func defaultUserAgent() string {
	version := build.Version
	if len(version) == 0 {
		version = "development"
	}
	return fmt.Sprintf("Go-http-client/1.1 (wig/%s, +%s)", version, wigURL)
}
