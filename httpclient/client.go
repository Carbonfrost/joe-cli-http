package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/fs"
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
const joeURL = "https://github.com/Carbonfrost/joe-cli-http"

// Client provides an HTTP client that can be accessed from commands,
// flags, and args within Joe applications.  When you register the Client
// within a Uses pipeline, it also registers flags, templates, and
// other handlers to enable its configuration from the command line.
// The simplest action to use is FetchAndPrint(), which executes the
// request(s) and prints (or downloads) the results:
//
//	&cli.App{
//	   Name: "gocurl",
//	   Uses: &httpclient.New(),
//	   Action: httpclient.FetchAndPrint(),
//	}
//
// This simple app has numerous flags and its simplest invocation
// could be something like
//
//	gocurl https://example.com/
//
// The cmd/wig package provides wig, which is a command line utility
// very similar to this.
//
// If you only want to add the Client to the context (typically in
// advanced scenarios where you are deeply customizing the behavior),
// you only use the action httpclient.ContextValue() with the client
// you want to add instead of add the client to the pipeline directly.
type Client struct {
	Client                 *http.Client
	Request                *http.Request
	IncludeResponseHeaders bool
	InterfaceResolver      InterfaceResolver
	BodyContent            Content
	UserInfo               *UserInfo
	LocationResolver       LocationResolver

	downloader     Downloader
	integrity      *Integrity
	dialer         *net.Dialer
	dnsDialer      *net.Dialer
	tlsConfig      *tls.Config
	auth           Authenticator
	authMiddleware []func(Authenticator) Authenticator
	bodyForm       []*cli.NameValue
	queryString    url.Values
	certFile       string
	keyFile        string
	rootCAs        []string
	middleware     []Middleware
	writeOutExpr   Expr
	writeErrExpr   Expr
}

type RoundTripperFunc func(req *http.Request) *http.Response

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
		queryString:       url.Values{},
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
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.DialContext = h.dialer.DialContext
	defaultTransport.Proxy = http.ProxyFromEnvironment
	h.Client = &http.Client{
		Transport: &traceableTransport{
			Transport: defaultTransport,
		},
	}

	for _, o := range append(impliedOptions, options...) {
		o(h)
	}
	return h
}

func WithDefaultUserAgent(s string) Option {
	return func(c *Client) {
		ensureHeader(c.Request).Set("User-Agent", s)
	}
}

func WithLocationResolver(r LocationResolver) Option {
	return func(c *Client) {
		c.LocationResolver = r
	}
}

// WithMiddleware adds a middleware function that will execute before
// the client request
func WithMiddleware(m Middleware) Option {
	return func(c *Client) {
		c.AddMiddleware(m)
	}
}

// WithRequestID provides middleware to the client that adds a header
// X-Request-ID to the request.  The optional argument defines how to generate
// the ID.  When specified, it must be one of these types:
//
//   - string
//   - func()string
//   - func(context.Context)(string, error)
//
// When unspecified, a cryptographically random string is generated for
// request IDs.
func WithRequestID(v ...any) Option {
	mw := NewRequestIDMiddleware(v...)
	return WithMiddleware(mw)
}

func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) {
		c.Client.Transport.(*traceableTransport).Transport = t
	}
}

func Do(c *cli.Context) ([]*Response, error) {
	return FromContext(c).Do(c)
}

// FromContext obtains the client stored in the context
func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
}

func (c *Client) AddMiddleware(m Middleware) {
	c.middleware = append(c.middleware, m)
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
		r, err := c.doOne(ctx, u)
		if err != nil {
			return rsp, err
		}
		rsp = append(rsp, r)
	}
	return rsp, nil
}

func (c *Client) generateMiddleware() []Middleware {
	return append([]Middleware{
		setupBodyContent(c),
		setupQueryString(c),
		processAuth(c),
	}, c.middleware...)
}

func (c *Client) doOne(ctx context.Context, u *url.URL) (*Response, error) {
	c.Request.URL = u
	c.Request.Host = u.Host
	c.Request = c.Request.WithContext(ctx)

	for _, m := range c.generateMiddleware() {
		err := m.Handle(c.Request)
		if err != nil {
			return nil, err
		}
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
	auth := c.Authenticator()
	for _, a := range c.authMiddleware {
		auth = a(auth)
	}
	return auth.Authenticate(c.Request, c.UserInfo)
}

func (c *Client) loadClientTLSCreds() error {
	if c.certFile != "" || c.keyFile != "" {
		t := c.Client.Transport.(*traceableTransport)
		if defaultTransport, ok := t.Transport.(*http.Transport); ok {
			defaultTransport.TLSClientConfig = c.tlsConfig
		}

		cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
		cfg := c.TLSConfig()
		cfg.Certificates = append(cfg.Certificates, cert)
		if err != nil {
			return err
		}
	}

	caCertPool := c.TLSConfig().RootCAs
	if caCertPool == nil {
		caCertPool = x509.NewCertPool()
		c.TLSConfig().RootCAs = caCertPool
	}

	for _, path := range c.rootCAs {
		cert, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_ = caCertPool.AppendCertsFromPEM(cert)
	}

	return nil
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

func (c *Client) SetIncludeResponseHeaders(v bool) error {
	c.IncludeResponseHeaders = v
	return nil
}

func (c *Client) setOutputFileHelper(f *cli.File) error {
	c.SetDownloadFile(&directAdapter{f})
	return nil
}

func (c *Client) SetIntegrity(i Integrity) error {
	c.integrity = &i
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

func (c *Client) SetCurves(ids *CurveIDs) error {
	c.TLSConfig().CurvePreferences = []tls.CurveID(*ids)
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
	if c.BodyContent == nil {
		c.BodyContent = NewContent(*name)

	} else {
		var err error
		c.BodyContent, err = convertContent(c.BodyContent, *name)
		if err != nil {
			return err
		}
	}

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

func (c *Client) openDownload(ctx *cli.Context, resp *Response) (io.WriteCloser, error) {
	downloader := c.actualDownloader(ctx)
	if c.integrity != nil {
		downloader = NewIntegrityDownloader(*c.integrity, downloader)
	}
	return downloader.OpenDownload(resp)
}

func (c *Client) actualDownloader(ctx *cli.Context) Downloader {
	if c.downloader == nil {
		return NewDownloaderTo(ctx.Stdout)
	}
	return c.downloader
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

func (c *Client) Authenticator() Authenticator {
	if c.auth == nil {
		c.auth = NoAuth
	}
	return c.auth
}

func (c *Client) UseAuthMiddleware(fn func(Authenticator) Authenticator) {
	c.authMiddleware = append(c.authMiddleware, fn)
}

func (c *Client) SetCACertFile(path string) error {
	c.rootCAs = append(c.rootCAs, path)
	return nil
}

func (c *Client) SetCACertPath(path string) error {
	paths, err := fs.Glob(os.DirFS("."), "*.pem")
	if err != nil {
		return err
	}
	c.rootCAs = append(c.rootCAs, paths...)
	return nil
}

func (c *Client) SetClientCertFile(path string) error {
	c.certFile = path
	return nil
}

func (c *Client) SetKeyFile(path string) error {
	c.keyFile = path
	return nil
}

func (c *Client) SetServerName(s string) error {
	c.TLSConfig().ServerName = s
	return nil
}

func (c *Client) setTimeHelper(f *cli.File) error {
	s, err := f.Stat()
	if err != nil {
		return err
	}
	c.TLSConfig().Time = s.ModTime
	return nil
}

func (c *Client) SetNextProtos(s []string) error {
	c.TLSConfig().NextProtos = s
	return nil
}

func (c *Client) SetRequestID(s string) error {
	if s == "" {
		WithRequestID()(c)
		return nil
	}
	WithRequestID(s)(c)
	return nil
}

func (c *Client) SetQueryString(n *cli.NameValue) error {
	c.queryString.Add(n.Name, n.Value)
	return nil
}

func (c *Client) SetWriteOut(w Expr) error {
	c.writeOutExpr = w
	return nil
}

func (c *Client) SetWriteErr(w Expr) error {
	c.writeErrExpr = w
	return nil
}

func (o Option) Execute(c *cli.Context) error {
	o(FromContext(c))
	return nil
}

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func defaultUserAgent() string {
	version := build.Version
	if len(version) == 0 {
		version = "development"
	}
	return fmt.Sprintf("Go-http-client/1.1 (joe-cli-http/%s, +%s)", version, joeURL)
}

func ensureHeader(r *http.Request) http.Header {
	if r.Header == nil {
		r.Header = http.Header{}
	}
	return r.Header
}
