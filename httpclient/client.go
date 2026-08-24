// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"bytes"
	"context"
	gotls "crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli-http/internal/pattern"
	"github.com/Carbonfrost/joe-cli-http/tls"
	joetls "github.com/Carbonfrost/joe-cli-http/tls"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
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
// The client is configured exclusively with Options, either passed to New or
// applied later using Apply.  Because an Option is also an Action, options can
// likewise be used within the Uses or Before pipeline, where they apply to the
// client which is in the context.
//
// The cmd/wig package provides wig, which is a command line utility
// very similar to this.
//
// If you only want to add the Client to the context (typically in
// advanced scenarios where you are deeply customizing the behavior),
// you only use the action httpclient.ContextValue() with the client
// you want to add instead of add the client to the pipeline directly.
type Client struct {
	cli.Action

	CheckRedirect          func(*http.Request, []*http.Request) error
	Request                *http.Request
	IncludeResponseHeaders bool
	BodyContent            Content
	UserInfo               *UserInfo
	LocationResolver       LocationResolver

	// FailFast causes no response output in the case of a failure
	FailFast bool

	downloader           Downloader
	downloaderMiddleware []DownloaderMiddleware

	transport  cacheable[http.RoundTripper]
	traceLevel TraceLevel

	tls               cacheable[*gotls.Config]
	interfaceResolver cacheable[InterfaceResolver]
	dialer            *net.Dialer
	dnsDialer         *net.Dialer
	auth              Authenticator
	authMiddleware    []AuthenticatorMiddleware

	bodyForm     []*cli.NameValue
	queryString  url.Values
	middleware   []Middleware
	writeOutExpr Expr
	writeErrExpr Expr

	// These are values that are ready after the first call to Do
	exprHandlingCache *exprHandling
	logger            TraceLogger
}

// AuthenticatorMiddleware provides middleware to the authenticator
type AuthenticatorMiddleware func(context.Context, Authenticator) Authenticator

// Option is an option to configure the client.
// Option can be used as an Action, typically within the Uses or Before pipeline.
type Option interface {
	cli.Action
	apply(*Client)
}

type option[T any] struct {
	val T
	fn  func(*Client, T) error
}

func (o option[_]) Execute(ctx context.Context) error {
	o.apply(FromContext(ctx))
	return nil
}

func (o option[_]) apply(c *Client) {
	o.fn(c, o.val)
}

type optionFunc func(*Client) error

func (f optionFunc) Execute(ctx context.Context) error {
	return f(FromContext(ctx))
}

func (f optionFunc) apply(c *Client) {
	f(c)
}

type cacheable[T comparable] = pattern.Cacheable[T]

type exprHandling struct {
	outExpr   *expander.Pattern
	errExpr   *expander.Pattern
	outRender io.Writer
	errRender io.Writer
}

var (
	impliedOptions = []Option{
		WithDefaultAction(),
		WithUserAgent(defaultUserAgent()),
		WithDefaultInterfaceResolver(),
		WithDefaultTransportFactory(),
	}

	// These don't have values within redirects
	noResponseExpander = expander.Map(map[string]any{
		"status":          "",
		"statusCode":      "",
		"http.version":    "",
		"http.proto":      "",
		"http.protoMajor": "",
		"http.protoMinor": "",
		"contentLength":   "",
		"header":          "",
	})
	noHeaderExpander = expander.Prefix("header", expander.Func(func(_ string) any {
		return ""
	}))
)

// New creates a new client with the given option.
func New(options ...Option) *Client {
	h := &Client{
		dnsDialer:   &net.Dialer{},
		queryString: url.Values{},
		Request: &http.Request{
			Method: "GET",
		},
	}
	h.dialer = &net.Dialer{
		Resolver: &net.Resolver{
			Dial: h.dnsDialer.DialContext,
		},
	}

	h.Apply(append(impliedOptions, options...)...)
	return h
}

// Apply applies the given options to the client
func (c *Client) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(c)
	}
}

func (c *Client) Pipeline() cli.Action {
	return c.Action
}

// WithAction sets the action
func WithAction(a cli.Action) Option {
	return withAdapter((*Client).setAction, a)
}

// WithDefaultAction sets the action to the default.  This option is applied
// automatically by New.
func WithDefaultAction() Option {
	return optionFunc(func(c *Client) error {
		c.Action = cli.Pipeline(
			FlagsAndArgs(),
			cli.Before(cli.Pipeline(
				cli.RegisterTemplateFunc("RedactHeader", c.redactHeader),
				registerFallbackFuncs(),
				cli.RegisterTemplate("HTTPTrace", outputTemplateText),
			)),
			ContextValue(c),
			Authenticators,
			PromptForCredentials(),
			joetls.New(),
			WithDefaultTLSConfigFactory(),
		)
		return nil
	})
}

// WithUserAgent sets the User-Agent header on the request
func WithUserAgent(s string) Option {
	return withAdapter((*Client).setUserAgent, s)
}

// WithLocationResolver sets the resolver which obtains the request locations
func WithLocationResolver(r LocationResolver) Option {
	return withAdapter((*Client).setLocationResolver, r)
}

// WithTLSConfig sets the TLS config for use on the client
func WithTLSConfig(t *gotls.Config) Option {
	return withAdapter((*Client).setTLSConfig, t)
}

// WithTLSConfigFactory provides a factory for obtaining TLS config
func WithTLSConfigFactory(fn func(context.Context) (*gotls.Config, error)) Option {
	return withAdapter((*Client).setTLSConfigFactory, fn)
}

// WithDefaultTLSConfigFactory provides the default factory, which provides
// TLS from the context
func WithDefaultTLSConfigFactory() Option {
	return WithTLSConfigFactory(func(ctx context.Context) (*gotls.Config, error) {
		return tls.FromContext(ctx).Config, nil
	})
}

// WithInterfaceResolver sets the interface resolver for use on the client
func WithInterfaceResolver(r InterfaceResolver) Option {
	return withAdapter((*Client).setInterfaceResolver, r)
}

// WithInterfaceResolverFactory provides a factory for obtaining the interface resolver
func WithInterfaceResolverFactory(fn func(context.Context) (InterfaceResolver, error)) Option {
	return withAdapter((*Client).setInterfaceResolverFactory, fn)
}

// WithDefaultInterfaceResolver sets up the default interface resolver.  This
// option is applied automatically by New.
func WithDefaultInterfaceResolver() Option {
	return WithInterfaceResolver(DefaultInterfaceResolver)
}

// WithMiddleware adds a middleware function that will execute before
// the client request
func WithMiddleware(m Middleware) Option {
	return withAdapter((*Client).addMiddleware, m)
}

// WithTransportMiddleware adds middleware to the transport
func WithTransportMiddleware(m TransportMiddleware) Option {
	return withAdapter((*Client).addTransportMiddleware, m)
}

// WithDownloaderMiddleware adds downloader middleware
func WithDownloaderMiddleware(d DownloaderMiddleware) Option {
	return withAdapter((*Client).addDownloaderMiddleware, d)
}

// WithAuthenticatorMiddleware adds middleware for the authenticator
func WithAuthenticatorMiddleware(fn AuthenticatorMiddleware) Option {
	return withAdapter((*Client).addAuthenticatorMiddleware, fn)
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

// WithRequestMethod sets the method of the request
func WithRequestMethod(s string) Option {
	return withAdapter((*Client).setMethod, s)
}

// AddRequestHeader appends the given header name and value to the request
func AddRequestHeader(v *HeaderValue) Option {
	return withAdapter((*Client).addHeader, v)
}

// WithFollowRedirects sets whether redirects in the Location header are
// followed.  When enabled, the default policy of following up to 10 redirects
// applies unless CheckRedirect is set explicitly.
func WithFollowRedirects(v bool) Option {
	return withAdapter((*Client).setFollowRedirects, v)
}

// WithBodyString sets up the body on the request to the given string
func WithBodyString(s string) Option {
	return WithBody(io.NopCloser(bytes.NewReader([]byte(s))))
}

// WithBody sets up the body on the request to the given reader
func WithBody(b io.ReadCloser) Option {
	return requestOption(func(r *http.Request) error {
		r.Body = b
		return nil
	})
}

// WithBodyContent sets the content of the body of the request
func WithBodyContent(bodyContent Content) Option {
	return withAdapter((*Client).setBodyContent, bodyContent)
}

// WithBodyContentString sets the raw content of the body of the request
func WithBodyContentString(s string) Option {
	return withAdapter((*Client).setBodyContentString, s)
}

// WithFillValue adds a value which fills the body of the request or the
// query string
func WithFillValue(v *cli.NameValue) Option {
	return withAdapter((*Client).addFillValue, v)
}

// WithQueryString adds a name and value to the query string
func WithQueryString(v *cli.NameValue) Option {
	return withAdapter((*Client).addQueryString, v)
}

// WithBaseURL sets the base URL used to resolve relative request locations
func WithBaseURL(u *URLValue) Option {
	return withAdapter((*Client).setBaseURL, u)
}

// WithURL adds the given URL to the request locations
func WithURL(u *url.URL) Option {
	return withAdapter((*Client).addURL, u)
}

// WithURLValue adds the given URL to the request locations
func WithURLValue(u *URLValue) Option {
	return withAdapter((*Client).addURLValue, u)
}

// WithURITemplateVar adds a value used to fill an RFC 6570 URI template
func WithURITemplateVar(v *uritemplates.Var) Option {
	return withAdapter((*Client).addURITemplateVar, v)
}

// WithURITemplateVars adds values used to fill an RFC 6570 URI template
func WithURITemplateVars(v *uritemplates.Vars) Option {
	return withAdapter((*Client).addURITemplateVars, v)
}

// WithIncludeResponseHeaders sets whether response headers are copied to
// the output
func WithIncludeResponseHeaders(v bool) Option {
	return withAdapter((*Client).setIncludeResponseHeaders, v)
}

// WithOutputFile downloads the response to the given file instead of
// writing it to stdout
func WithOutputFile(f string) Option {
	return withAdapter((*Client).setOutputFile, f)
}

// WithNoOutput sets whether the response output is discarded
func WithNoOutput(v bool) Option {
	return withAdapter((*Client).setNoOutput, v)
}

// WithDownloadFile sets the downloader which handles the response
func WithDownloadFile(d Downloader) Option {
	return withAdapter((*Client).setDownloadFile, d)
}

// WithIntegrity validates the integrity of the download
func WithIntegrity(i Integrity) Option {
	return withAdapter((*Client).setIntegrity, i)
}

// WithStripComponents removes the specified number of leading path elements
// when downloading files
func WithStripComponents(count int) Option {
	return withAdapter((*Client).setStripComponents, count)
}

// WithFailFast sets whether to fail with no output on HTTP errors
func WithFailFast(v bool) Option {
	return withAdapter((*Client).setFailFast, v)
}

// WithPreferGoDialer sets whether Go's built-in DNS resolver is preferred
func WithPreferGoDialer(v bool) Option {
	return withAdapter((*Client).setPreferGoDialer, v)
}

// WithStrictErrorsDNS sets whether the Go built-in DNS resolver returns
// errors instead of partial results
func WithStrictErrorsDNS(v bool) Option {
	return withAdapter((*Client).setStrictErrorsDNS, v)
}

// WithDisableDialKeepAlive disables dialer keep-alive probes
func WithDisableDialKeepAlive(v bool) Option {
	return withAdapter((*Client).setDisableDialKeepAlive, v)
}

// WithDialKeepAlive sets the interval between keep-alive probes for an
// active network connection
func WithDialKeepAlive(v time.Duration) Option {
	return withAdapter((*Client).setDialKeepAlive, v)
}

// WithDialTimeout sets the maximum amount of time a dial waits for a
// connect to complete
func WithDialTimeout(v time.Duration) Option {
	return withAdapter((*Client).setDialTimeout, v)
}

// WithBindAddress binds client TCP/IP connections to the given address on
// the local machine
func WithBindAddress(v string) Option {
	return withAdapter((*Client).setBindAddress, v)
}

// WithInterface uses the network interface, named by name or address, to connect
func WithInterface(v string) Option {
	return withAdapter((*Client).setInterface, v)
}

// WithDNSInterface uses the network interface, named by name or address, for
// DNS requests
func WithDNSInterface(v string) Option {
	return withAdapter((*Client).setDNSInterface, v)
}

// WithAuth sets the authenticator used on the request
func WithAuth(auth Authenticator) Option {
	return withAdapter((*Client).setAuth, auth)
}

// WithUser sets the user and password used in authentication
func WithUser(user *UserInfo) Option {
	return withAdapter((*Client).setUser, user)
}

// WithTraceLevel sets which client operations are traced
func WithTraceLevel(v TraceLevel) Option {
	return withAdapter((*Client).setTraceLevel, v)
}

// WithWriteOut sets the expression which is evaluated and printed to stdout
func WithWriteOut(w Expr) Option {
	return withAdapter((*Client).setWriteOut, w)
}

// WithWriteErr sets the expression which is evaluated and printed to stderr
func WithWriteErr(w Expr) Option {
	return withAdapter((*Client).setWriteErr, w)
}

// withBodyContentType sets or converts the body content to the given content type
func withBodyContentType(name *ContentType) Option {
	return withAdapter((*Client).setBodyContentType, name)
}

// withRequestID sets or generates the X-Request-ID header.  An empty value
// causes the ID to be generated.
func withRequestID(s string) Option {
	if s == "" {
		return WithRequestID()
	}
	return WithRequestID(s)
}

func requestOption(fn func(r *http.Request) error) Option {
	return optionFunc(func(c *Client) error {
		return fn(c.Request)
	})
}

func withAdapter[T any](fn func(*Client, T) error, value T) Option {
	return option[T]{value, fn}
}

// Do invokes the context client to generate corresponding responses
func Do(c context.Context) ([]*Response, error) {
	return FromContext(c).Do(c)
}

// FromContext obtains the client stored in the context
func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
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

// DoLocation invokes the request for the specified location
func (c *Client) DoLocation(ctx context.Context, l Location) (*Response, error) {
	return c.doOne(ctx, l)
}

func (c *Client) ensureExprHandling(ctx *cli.Context) {
	// Note that errRender always writes to stderr even if %(stdout) expr
	// is present
	c.exprHandlingCache = &exprHandling{
		outRender: expander.NewRenderer(ctx.Stdout, ctx.Stderr),
		errRender: expander.NewRenderer(ctx.Stderr, ctx.Stderr),
		outExpr:   c.writeOutExpr.Compile(),
		errExpr:   c.writeErrExpr.Compile(),
	}
}

func (c *Client) ensureClient(ctx context.Context) *http.Client {
	return &http.Client{
		Transport:     c.actualTransport(ctx),
		CheckRedirect: c.actualCheckRedirect(),
	}
}

func (c *Client) actualCheckRedirect() func(*http.Request, []*http.Request) error {
	redirect := c.CheckRedirect
	if redirect == nil {
		redirect = defaultCheckRedirect
	}

	// Wrap with support from the logger
	return func(req *http.Request, via []*http.Request) error {
		err := redirect(req, via)
		c.exprHandlingCache.eval(nil, req, nil)
		c.logger.Redirected(req, via, err)
		return err
	}
}

func defaultCheckRedirect(_ *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	return nil
}

func (c *Client) generateMiddleware(l Location) Middleware {
	mw, _ := l.(Middleware)
	return ComposeMiddleware(append([]Middleware{
		mw,
		setupBodyContent(c),
		setupQueryString(c),
		processAuth(c),
	}, c.middleware...)...)
}

func (c *Client) doOne(ctx context.Context, l Location) (*Response, error) {
	client := c.ensureClient(ctx)
	c.ensureExprHandling(cli.FromContext(ctx))

	rctx, u, err := l.URL(ctx)
	if err != nil {
		return nil, err
	}
	c.Request.URL = u
	c.Request.Host = u.Host
	c.Request = c.Request.WithContext(rctx)
	c.generateMiddleware(l).Handle(c.Request, nil)

	netResp, err := client.Do(c.Request)
	if err != nil {
		return nil, err
	}
	resp := &Response{
		Response: netResp,
	}

	c.exprHandlingCache.eval(c.Request, nil, netResp)
	err = c.handleDownload(ctx, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) handleDownload(ctx context.Context, response *Response) error {
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

	return nil
}

func (c *Client) applyAuth(ctx context.Context) error {
	auth := c.Authenticator()
	for _, a := range c.authMiddleware {
		auth = a(ctx, auth)
	}
	return auth.Authenticate(c.Request, c.UserInfo)
}

func (c *Client) Dialer() *net.Dialer {
	return c.dialer
}

func (c *Client) DNSDialer() *net.Dialer {
	return c.dnsDialer
}

// NewInterfaceResolver creates the interface resolver
func (c *Client) NewInterfaceResolver(ctx context.Context) (InterfaceResolver, error) {
	return c.interfaceResolver.New(ctx)
}

// NewTLSConfig creates the TLS config
func (c *Client) NewTLSConfig(ctx context.Context) (*gotls.Config, error) {
	return c.tls.New(ctx)
}

// Authenticator obtains the authenticator which has been configured
func (c *Client) Authenticator() Authenticator {
	if c.auth == nil {
		return NoAuth
	}
	return c.auth
}

func (c *Client) setAction(value cli.Action) error {
	c.Action = value
	return nil
}

func (c *Client) setMethod(s string) error {
	c.Request.Method = strings.ToUpper(s)
	return nil
}

func (c *Client) setFollowRedirects(value bool) error {
	if value {
		c.CheckRedirect = nil // default policy to follow 10 times
		return nil
	}

	// Follow no redirects
	c.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return nil
}

func (c *Client) setUserAgent(value string) error {
	ensureHeader(c.Request).Set("User-Agent", value)
	return nil
}

func (c *Client) setLocationResolver(r LocationResolver) error {
	c.LocationResolver = r
	return nil
}

func (c *Client) setTLSConfig(t *gotls.Config) error {
	c.tls.SetDiscrete(t)
	return nil
}

func (c *Client) setTLSConfigFactory(fn func(context.Context) (*gotls.Config, error)) error {
	c.tls.SetFactory(fn)
	return nil
}

func (c *Client) setInterfaceResolver(r InterfaceResolver) error {
	c.interfaceResolver.SetDiscrete(r)
	return nil
}

func (c *Client) setInterfaceResolverFactory(fn func(context.Context) (InterfaceResolver, error)) error {
	c.interfaceResolver.SetFactory(fn)
	return nil
}

func (c *Client) setBaseURL(u *URLValue) error {
	uu, err := u.URL()
	if err != nil {
		return err
	}
	return c.ensureLocationResolver().SetBaseURL(uu)
}

func (c *Client) addURL(u *url.URL) error {
	return c.ensureLocationResolver().Add(u.String())
}

func (c *Client) addURLValue(u *URLValue) error {
	return c.ensureLocationResolver().Add(u.String())
}

func (c *Client) addURITemplateVar(v *uritemplates.Var) error {
	return c.ensureLocationResolver().AddVar(v.Name, v.Value)
}

func (c *Client) addURITemplateVars(v *uritemplates.Vars) error {
	for _, item := range v.Items() {
		err := c.ensureLocationResolver().AddVar(item.Name, item.Value)
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

func (c *Client) setIncludeResponseHeaders(v bool) error {
	c.IncludeResponseHeaders = v
	return nil
}

func (c *Client) setOutputFile(f string) error {
	return c.setDownloadFile(NewFileDownloader(f, nil))
}

func (c *Client) setNoOutput(b bool) error {
	if b {
		c.downloader = NewDownloaderTo(io.Discard)
		return nil
	}
	c.downloader = nil
	return nil
}

func (c *Client) setIntegrity(i Integrity) error {
	return c.addDownloaderMiddleware(func(_ context.Context, downloader Downloader) Downloader {
		return NewIntegrityDownloader(i, downloader)
	})
}

func (c *Client) setPreferGoDialer(v bool) error {
	c.Dialer().Resolver.PreferGo = v
	return nil
}

func (c *Client) setStrictErrorsDNS(v bool) error {
	c.Dialer().Resolver.StrictErrors = v
	return nil
}

func (c *Client) setDisableDialKeepAlive(v bool) error {
	if v {
		c.Dialer().KeepAlive = time.Duration(-1)
	}
	return nil
}

func (c *Client) addHeader(n *HeaderValue) error {
	ensureHeader(c.Request).Add(n.Name, n.Value)
	return nil
}

func (c *Client) setBindAddress(value string) error {
	addr, err := net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return err
	}
	c.Dialer().LocalAddr = addr
	return nil
}

func (c *Client) setInterface(value string) error {
	addr, err := c.resolveInterface(value)
	if err != nil {
		return err
	}
	c.Dialer().LocalAddr = addr
	return nil
}

func (c *Client) setDNSInterface(value string) error {
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

func (c *Client) setDialTimeout(v time.Duration) error {
	c.Dialer().Timeout = v
	return nil
}

func (c *Client) setDialKeepAlive(v time.Duration) error {
	c.Dialer().KeepAlive = v
	return nil
}

func (c *Client) setDownloadFile(v Downloader) error {
	c.downloader = v
	return nil
}

func (c *Client) setBodyContentString(body string) error {
	c.BodyContent = NewRawContent([]byte(body))
	return nil
}

func (c *Client) setBodyContentType(name *ContentType) error {
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

func (c *Client) setBodyContent(bodyContent Content) error {
	c.BodyContent = bodyContent
	return nil
}

func (c *Client) addFillValue(v *cli.NameValue) error {
	c.bodyForm = append(c.bodyForm, v)
	return nil
}

func (c *Client) resolveInterface(v string) (*net.TCPAddr, error) {
	resolver, err := c.NewInterfaceResolver(context.Background())
	if err != nil {
		return nil, err
	}
	return resolver.Resolve(context.Background(), v)
}

func (c *Client) openDownload(ctx context.Context, resp *Response) (io.WriteCloser, error) {
	downloader := c.actualDownloader(ctx)
	return downloader.OpenDownload(ctx, resp)
}

func (c *Client) actualDownloader(ctx context.Context) Downloader {
	downloader := c.downloader
	if c.downloader == nil {
		downloader = NewDownloaderTo(cli.FromContext(ctx).Stdout)
	}
	for _, d := range c.downloaderMiddleware {
		downloader = d(ctx, downloader)
	}
	return downloader
}

func (c *Client) setAuth(auth Authenticator) error {
	c.auth = auth
	return nil
}

func (c *Client) setUser(user *UserInfo) error {
	c.UserInfo = user
	return nil
}

func (c *Client) addMiddleware(m Middleware) error {
	c.middleware = append(c.middleware, m)
	return nil
}

func (c *Client) addAuthenticatorMiddleware(fn AuthenticatorMiddleware) error {
	c.authMiddleware = append(c.authMiddleware, fn)
	return nil
}

func (c *Client) addDownloaderMiddleware(fn DownloaderMiddleware) error {
	c.downloaderMiddleware = append(c.downloaderMiddleware, fn)
	return nil
}

func (c *Client) addQueryString(n *cli.NameValue) error {
	c.queryString.Add(n.Name, n.Value)
	return nil
}

func (c *Client) setTraceLevel(v TraceLevel) error {
	c.traceLevel = v
	return nil
}

func (c *Client) setWriteOut(w Expr) error {
	c.writeOutExpr = w
	return nil
}

func (c *Client) setWriteErr(w Expr) error {
	c.writeErrExpr = w
	return nil
}

func (c *Client) setStripComponents(count int) error {
	if err := c.setDownloadFile(PreserveRequestPath); err != nil {
		return err
	}
	return c.addDownloaderMiddleware(func(_ context.Context, d Downloader) Downloader {
		return d.(DownloadMode).WithStripComponents(count)
	})
}

func (c *Client) setFailFast(v bool) error {
	c.FailFast = v
	return nil
}

func (e *exprHandling) eval(initial, req *http.Request, resp *http.Response) {
	expanders := []expander.Interface{
		expr.ExpandGlobals(),
		expander.Prefix("color", expander.Colors()),
		expander.Prefix("redirect", ExpandRequest(req)),
		expander.Prefix("request", ExpandRequest(initial)),
	}

	if resp == nil {
		expanders = append(expanders, noResponseExpander, noHeaderExpander)
	} else {
		expanders = append(expanders, ExpandResponse(resp))
	}
	expanders = append(expanders, expander.Unknown())
	exp := expander.Compose(expanders...)

	e.outExpr.Fprint(e.outRender, exp)
	e.errExpr.Fprint(e.errRender, exp)
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
