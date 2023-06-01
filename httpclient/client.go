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
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
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

	// FailFast causes no response output in the case of a failure
	FailFast bool

	downloader         Downloader
	downloadMiddleware []func(Downloader) Downloader

	transport           http.RoundTripper
	transportMiddleware []TransportMiddleware
	traceLevel          TraceLevel

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

// TransportMiddleware provides middleware to the roundtripper
type TransportMiddleware func(context.Context, http.RoundTripper) http.RoundTripper

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
	h.Client = &http.Client{}

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

// WithTransportMiddleware adds middleware to the transport
func WithTransportMiddleware(m TransportMiddleware) Option {
	return func(c *Client) {
		c.AddTransportMiddleware(m)
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

// WithTransport sets the default transport
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) {
		c.transport = t
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

func (c *Client) AddTransportMiddleware(m TransportMiddleware) {
	c.transportMiddleware = append(c.transportMiddleware, m)
}

func (c *Client) SetTraceLevel(v TraceLevel) error {
	c.traceLevel = v
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

	c.Client = &http.Client{
		Transport: c.actualTransport(ctx),
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

func (c *Client) actualTransport(ctx context.Context) http.RoundTripper {
	t := c.transport
	if t == nil {
		defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
		defaultTransport.DialContext = c.dialer.DialContext
		defaultTransport.Proxy = http.ProxyFromEnvironment
		t = defaultTransport
	}
	for _, m := range c.generateTransportMiddleware() {
		t = m(ctx, t)
	}
	return t
}

func (c *Client) generateMiddleware() []Middleware {
	return append([]Middleware{
		setupBodyContent(c),
		setupQueryString(c),
		processAuth(c),
	}, c.middleware...)
}

func (c *Client) generateTransportMiddleware() []TransportMiddleware {
	return append([]TransportMiddleware{
		c.setupTraceLevelTransport,
	}, c.transportMiddleware...)
}

func (c *Client) setupTraceLevelTransport(ctx context.Context, t http.RoundTripper) http.RoundTripper {
	var logger TraceLogger
	if c.traceLevel == TraceOff {
		logger = nopTraceLogger{}
	} else {
		logger = &defaultTraceLogger{
			template: traceTemplate(ctx),
			out:      os.Stderr,
			flags:    c.traceLevel,
		}
	}
	return &traceableTransport{
		logger:    logger,
		Transport: t,
	}
}

func (c *Client) doOne(ctx context.Context, l Location) (*Response, error) {
	rctx, u, err := l.URL(ctx)
	if err != nil {
		return nil, err
	}
	c.Request.URL = u
	c.Request.Host = u.Host
	c.Request = c.Request.WithContext(rctx)

	for _, m := range c.generateMiddleware() {
		err := m.Handle(c.Request)
		if err != nil {
			return nil, err
		}
	}

	netResp, err := c.Client.Do(c.Request)
	if err != nil {
		return nil, err
	}
	resp := &Response{
		Response: netResp,
	}

	err = c.handleDownload(cli.FromContext(ctx), resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) handleDownload(ctx *cli.Context, response *Response) error {
	// Note that errRender always writes to stderr even if %(stdout) expr
	// is present
	outRender := expr.NewRenderer(ctx.Stdout, ctx.Stderr)
	errRender := expr.NewRenderer(ctx.Stderr, ctx.Stderr)

	outExpr := c.writeOutExpr.Compile()
	errExpr := c.writeErrExpr.Compile()

	if c.FailFast && !response.Success() {
		return fmt.Errorf("request failed (%s): %s %s", response.Status, response.Request.Method, response.Request.URL)
	}

	output, err := c.openDownload(ctx, response)
	if err != nil {
		return err
	}

	if c.IncludeResponseHeaders {
		err = response.CopyHeadersTo(output)
		fmt.Fprintln(output)
	}
	if err != nil {
		return err
	}

	err = response.CopyTo(output)
	if err != nil {
		return err
	}

	err = output.Close()
	if err != nil {
		return err
	}

	expander := expr.ComposeExpanders(
		expr.ExpandGlobals,
		expr.Prefix("color", expr.ExpandColors),
		ExpandResponse(response),
		expr.Unknown,
	)

	expr.Fprint(outRender, outExpr, expander)
	expr.Fprint(errRender, errExpr, expander)
	return nil
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
		if defaultTransport, ok := c.transport.(*http.Transport); ok {
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
	c.SetDownloadFile(NewFileDownloader(f))
	return nil
}

func (c *Client) SetNoOutput(b bool) error {
	if b {
		c.downloader = NewDownloaderTo(io.Discard)
		return nil
	}
	c.downloader = nil
	return nil
}

func (c *Client) SetIntegrity(i Integrity) error {
	c.UseDownloadMiddleware(func(downloader Downloader) Downloader {
		return NewIntegrityDownloader(i, downloader)
	})
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

func (c *Client) SetBindAddress(value string) error {
	addr, err := net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return err
	}
	c.Dialer().LocalAddr = addr
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
	return downloader.OpenDownload(resp)
}

func (c *Client) actualDownloader(ctx *cli.Context) Downloader {
	downloader := c.downloader
	if c.downloader == nil {
		downloader = NewDownloaderTo(ctx.Stdout)
	}
	for _, d := range c.downloadMiddleware {
		downloader = d(downloader)
	}
	return downloader
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

func (c *Client) UseDownloadMiddleware(fn func(Downloader) Downloader) {
	c.downloadMiddleware = append(c.downloadMiddleware, fn)
}

func (c *Client) SetStripComponents(count int) error {
	c.SetDownloadFile(PreserveRequestPath)
	c.UseDownloadMiddleware(func(d Downloader) Downloader {
		return d.(DownloadMode).WithStripComponents(count)
	})
	return nil
}

func (c *Client) SetFailFast(v bool) error {
	c.FailFast = v
	return nil
}

func NewIntegrityDownloadMiddleware(i Integrity) func(Downloader) Downloader {
	return func(d Downloader) Downloader {
		return NewIntegrityDownloader(i, d)
	}
}

func (o Option) Execute(c context.Context) error {
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
