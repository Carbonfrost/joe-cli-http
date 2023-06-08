package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"strconv"
	"strings"
	"text/template"

	"github.com/Carbonfrost/joe-cli"
)

// TraceLevel indicates the amount of client tracing to generate
type TraceLevel int

type httpStatus int

func (s httpStatus) Color() string {
	switch 100 * (s / 100) {
	case 100:
		return "Magenta"
	case 200:
		return "Green"
	case 300:
		return "Yellow"
	case 400, 500:
		fallthrough
	default:
		return "Red"
	}
}

func (s httpStatus) Message() string {
	return http.StatusText(int(s))
}

func (s httpStatus) Code() int {
	return int(s)
}

func (s httpStatus) String() string {
	return strconv.Itoa(int(s)) + " " + s.Message()
}

// TraceLogger provides delegates from ClientTrace
type TraceLogger interface {
	ConnectDone(network, addr string, err error)
	ConnectStart(network, addr string)
	DNSDone(httptrace.DNSDoneInfo)
	DNSStart(httptrace.DNSStartInfo)
	GetConn(hostPort string)
	Got1xxResponse(code int, header textproto.MIMEHeader) error
	GotConn(httptrace.GotConnInfo)
	TLSHandshakeDone(tls.ConnectionState, error)
	TLSHandshakeStart()
	Wait100Continue()
	WroteHeaderField(key string, value []string)
	WroteRequest(httptrace.WroteRequestInfo)
	StartRequest(req *http.Request)
	ResponseDone(resp *http.Response, err error)
	Redirected(req *http.Request, via []*http.Request, err error)
}

type nopTraceLogger struct{}

type defaultTraceLogger struct {
	flags    TraceLevel
	out      io.Writer
	template *template.Template
}

type traceableTransport struct {
	logger    TraceLogger
	Transport http.RoundTripper
}

// Trace level components, which enumerates the various parts of the roundtrip to trace
const (
	TraceConnections = TraceLevel(1 << iota)
	TraceRequestHeaders
	TraceDNS
	TraceTLS
	TraceHTTP1XX
	TraceRequestBody
	TraceResponseStatus
	TraceResponseHeaders
	TraceRedirects

	// TraceOff causes all tracing to be switched off
	TraceOff TraceLevel = 0

	// TraceOn enables tracing of when connections are made and header
	TraceOn = TraceRequestHeaders | TraceResponseStatus | TraceResponseHeaders

	// TraceVerbose enables tracing of DNS, TLS, HTTP 1xx responses
	TraceVerbose = TraceOn | TraceConnections | TraceDNS | TraceTLS | TraceHTTP1XX | TraceRedirects
	TraceDebug   = TraceVerbose | TraceRequestBody
)

var (
	traceString = [...]string{
		"debug",
		"verbose",
		"on",
		"connections",
		"requestHeaders",
		"dns",
		"tls",
		"http1xx",
		"requestBody",
		"responseStatus",
		"responseHeaders",
		"redirects",
		"off",
	}
	traceEnum = [...]TraceLevel{
		TraceDebug,
		TraceVerbose,
		TraceOn,
		TraceConnections,
		TraceRequestHeaders,
		TraceDNS,
		TraceTLS,
		TraceHTTP1XX,
		TraceRequestBody,
		TraceResponseStatus,
		TraceResponseHeaders,
		TraceRedirects,
		TraceOff,
	}
)

var (
	printColor = func(a ...any) string {
		return fmt.Sprint(a[1:])
	}
	funcs = template.FuncMap{

		// Stub color functions when used outside of Joe-cli
		"Gray":       fmt.Sprint,
		"Red":        fmt.Sprint,
		"Magenta":    fmt.Sprint,
		"Blue":       fmt.Sprint,
		"ResetColor": fmt.Sprint,
		"Color":      printColor,
		"Join": func(v string, args []string) string {
			return strings.Join(args, v)
		},
	}
	outputTemplate = template.Must(template.New("HTTPTrace").Funcs(funcs).Parse(outputTemplateText))
)

const (
	// Design: blue for host names, magenta for HTTP header idioms
	outputTemplateText = `
{{- define "TLSHandshakeStart" -}}
{{ Gray }}* Establishing TLS connection...{{ ResetColor }}
{{ end -}}

{{- define "Got1xxResponse" -}}
{{ Gray }}< Got {{ .Code | Magenta }} {{.Header}}{{ ResetColor }}
{{ end -}}

{{- define "GetConn" -}}
{{ Gray }}* Connecting to {{ .HostPort | Blue }}{{ ResetColor }}...
{{ end -}}

{{- define "DNSStart" -}}
{{ Gray }}* Resolving name {{ .Host | Blue }}{{ResetColor}}...
{{ end -}}

{{- define "WroteHeaderField" -}}
{{ Gray }}{{ if .Response }}< {{ else }}> {{ end -}}
{{ .Key | Magenta }}: {{ .Value | Join ", " | Gray }}{{ResetColor}}
{{ end -}}

{{- define "StartRequest" -}}
{{ Gray }}> {{ .Method | Magenta }} {{ .RequestURI }} {{ .Proto }}{{ ResetColor }}
{{ end -}}

{{- define "Redirected" -}}
{{ Gray }}* Redirecting to {{ .Location }}
{{- if gt .Times 1 }} (
    {{- .Ordinal }} redirect)
{{- end }} ...{{ ResetColor }}
{{ end -}}

{{- define "GotConn" -}}
{{ Gray }}* Connected to {{ .Remote }} ({{ .LocalAddr }}{{ if .Reused }}, reused{{ end }}){{ResetColor}}
{{ end -}}

{{- define "TLSHandshakeDone" -}}
{{ Gray -}}
* SSL connection using {{ .Proto }} / {{ .CipherSuite }}
* Server certificate:
{{ range .ServerCertificate -}}
*   {{ .Name }}: {{ .Value }}
{{ end -}}
{{- ResetColor -}}
{{ end -}}

{{- define "DNSDone" -}}
{{ Gray }}* Resolved to {{ .Addrs | Join ", " }}{{ResetColor}}
{{ end -}}

{{- define "StatusCode" -}}
{{ Color .Status.Color }}{{ .Status }}{{ ResetColor }}
{{ end -}}

{{- define "GenericError" -}}
{{ Red }}{{ .Error }}{{ ResetColor }}
{{ end -}}

`
)

func newClientTrace(logger TraceLogger) *httptrace.ClientTrace {
	// Filter out pseudo-headers
	wrote := func(key string, value []string) {
		if strings.HasPrefix(key, ":") {
			return
		}
		logger.WroteHeaderField(key, value)
	}

	return &httptrace.ClientTrace{
		ConnectDone:       logger.ConnectDone,
		ConnectStart:      logger.ConnectStart,
		DNSDone:           logger.DNSDone,
		DNSStart:          logger.DNSStart,
		GetConn:           logger.GetConn,
		Got1xxResponse:    logger.Got1xxResponse,
		GotConn:           logger.GotConn,
		TLSHandshakeDone:  logger.TLSHandshakeDone,
		TLSHandshakeStart: logger.TLSHandshakeStart,
		Wait100Continue:   logger.Wait100Continue,
		WroteHeaderField:  wrote,
		WroteRequest:      logger.WroteRequest,
	}
}

// SetTraceLevel provides the action for a flag which sets the trace level
// corresponding to the flag's value.
func SetTraceLevel(s ...*TraceLevel) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "trace",
			HelpText:  "Set which client operations are traced",
			UsageText: "LEVEL",
			EnvVars:   []string{"HTTP_CLIENT_TRACE_LEVEL"},
		},
		withBinding((*Client).setTraceLevelHelper, s),
		tagged,
	)
}

func (l *TraceLevel) Set(arg string) error {
	var res TraceLevel
	for _, j := range strings.Split(arg, ",") {
		j = strings.TrimSpace(j)
		in := indexTraceString(j)
		if in < 0 {
			return fmt.Errorf("unknown trace level %q", arg)
		}
		res |= traceEnum[in]
	}
	*l = res
	return nil
}

func (l TraceLevel) String() string {
	if l == TraceOff {
		return "off"
	}

	result := make([]string, 0)
	for i, e := range traceEnum {
		if e == 0 {
			continue
		}
		if l&e == e {
			l &^= e
			result = append(result, traceString[i])
		}
	}
	if l != 0 {
		result = append(result, strconv.Itoa(int(l)))
	}
	return strings.Join(result, ",")
}

func (l TraceLevel) connections() bool {
	return l&TraceConnections == TraceConnections
}

func (l TraceLevel) requestHeaders() bool {
	return l&TraceRequestHeaders == TraceRequestHeaders
}

func (l TraceLevel) responseHeaders() bool {
	return l&TraceResponseHeaders == TraceResponseHeaders
}

func (l TraceLevel) redirects() bool {
	return l&TraceRedirects == TraceRedirects
}

func (l TraceLevel) dns() bool {
	return l&TraceDNS == TraceDNS
}

func (l TraceLevel) tls() bool {
	return l&TraceTLS == TraceTLS
}

func (l TraceLevel) http1xx() bool {
	return l&TraceHTTP1XX == TraceHTTP1XX
}

func (l TraceLevel) requestBody() bool {
	return l&TraceRequestBody == TraceRequestBody
}

func (l *defaultTraceLogger) ConnectDone(network, addr string, err error) {
	if !l.flags.connections() {
		return
	}
}

func (l *defaultTraceLogger) ConnectStart(network, addr string) {
	if !l.flags.connections() {
		return
	}
}

func (l *defaultTraceLogger) DNSDone(info httptrace.DNSDoneInfo) {
	if !l.flags.dns() {
		return
	}
	if info.Err != nil {
		fmt.Println(info.Err)
		return
	}

	addrs := make([]string, 0, len(info.Addrs))
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.String())
	}
	l.render("DNSDone", struct {
		Addrs []string
	}{
		Addrs: addrs,
	})
}

func (l *defaultTraceLogger) DNSStart(info httptrace.DNSStartInfo) {
	if !l.flags.dns() {
		return
	}
	l.render("DNSStart", struct {
		Host string
	}{
		Host: info.Host,
	})
}

func (l *defaultTraceLogger) GetConn(hostPort string) {
	if !l.flags.connections() {
		return
	}

	l.render("GetConn", struct {
		HostPort string
	}{
		HostPort: hostPort,
	})
}

func (l *defaultTraceLogger) Got1xxResponse(code int, header textproto.MIMEHeader) (err error) {
	if !l.flags.http1xx() {
		return
	}

	l.render("Got1xxResponse", struct {
		Code   int
		Header textproto.MIMEHeader
	}{
		Code:   code,
		Header: header,
	})
	return nil
}

func (l *defaultTraceLogger) GotConn(info httptrace.GotConnInfo) {
	if !l.flags.connections() {
		return
	}

	l.render("GotConn", struct {
		Remote    string
		LocalAddr string
		Reused    bool
	}{
		Remote:    info.Conn.RemoteAddr().String(),
		LocalAddr: info.Conn.LocalAddr().String(),
		Reused:    info.Reused,
	})
}

func (l *defaultTraceLogger) TLSHandshakeDone(state tls.ConnectionState, err error) {
	if !l.flags.tls() {
		return
	}

	l.render("TLSHandshakeDone", struct {
		Proto             string
		CipherSuite       string
		ServerCertificate []NameValue
	}{
		Proto:             versionString(state.Version),
		CipherSuite:       tls.CipherSuiteName(state.CipherSuite),
		ServerCertificate: formatCert(state.PeerCertificates[0]),
	})
}

type NameValue struct {
	Name  string
	Value string
}

func formatCert(c *x509.Certificate) []NameValue {
	return []NameValue{
		{"Subject", fmt.Sprint(c.Subject)},
		{"Not Before", fmt.Sprint(c.NotBefore)},
		{"Not After", fmt.Sprint(c.NotAfter)},
		{"Issuer", fmt.Sprint(c.Issuer)},
	}
}

func (l *defaultTraceLogger) TLSHandshakeStart() {
	if !l.flags.tls() {
		return
	}

	l.render("TLSHandshakeStart", nil)
}

func (l *defaultTraceLogger) Wait100Continue() {
	if !l.flags.http1xx() {
		return
	}
}

func (l *defaultTraceLogger) WroteHeaderField(key string, value []string) {
	if !l.flags.requestHeaders() {
		return
	}

	l.render("WroteHeaderField", struct {
		Key      string
		Value    []string
		Response bool
	}{
		Key:      key,
		Value:    value,
		Response: false,
	})
}

func (l *defaultTraceLogger) WroteRequest(info httptrace.WroteRequestInfo) {
	if !l.flags.requestBody() {
		return
	}
}

func (l *defaultTraceLogger) renderError(err error) {
	l.render("GenericError", struct {
		Error error
	}{
		Error: err,
	})
}

func (l *defaultTraceLogger) render(fn string, data interface{}) {
	err := l.template.ExecuteTemplate(l.out, fn, data)
	if err != nil {
		panic(err)
	}
}

func (l *defaultTraceLogger) StartRequest(req *http.Request) {
	path := req.URL.RequestURI()
	proto := req.Proto
	if path == "" {
		path = "/"
	}
	if proto == "" {
		proto = "HTTP/1.1"
	}
	l.render("StartRequest", struct {
		Method     string
		RequestURI string
		Proto      string
	}{
		Method:     req.Method,
		RequestURI: path,
		Proto:      proto,
	})
}

func (l *defaultTraceLogger) Redirected(req *http.Request, via []*http.Request, err error) {
	if !l.flags.redirects() {
		return
	}
	ordSuffix := func(i int) string {
		if i%100 == 11 || i%100 == 12 || i%100 == 13 {
			return "th"
		}
		if i%10 < 4 {
			return [4]string{"th", "st", "nd", "rd"}[i%10]
		}
		return "th"
	}

	if err != nil {
		l.renderError(err)
	}

	times := len(via)
	l.render("Redirected", struct {
		Location string
		Times    int
		Ordinal  string
	}{
		Location: req.URL.String(),
		Times:    times,
		Ordinal:  fmt.Sprintf("%d%s", times, ordSuffix(times)),
	})
}

func (l *defaultTraceLogger) ResponseDone(resp *http.Response, err error) {
	if resp == nil || err != nil {
		l.renderError(err)
		return
	}

	l.render("StatusCode", struct {
		Status httpStatus
	}{
		Status: httpStatus(resp.StatusCode),
	})

	if !l.flags.responseHeaders() {
		return
	}
	for k, v := range resp.Header {
		l.render("WroteHeaderField", struct {
			Key      string
			Value    []string
			Response bool
		}{
			Key:      k,
			Value:    v,
			Response: true,
		})
	}
}

func (nopTraceLogger) ConnectDone(network, addr string, err error) {
}

func (nopTraceLogger) ConnectStart(network, addr string) {
}

func (nopTraceLogger) DNSDone(httptrace.DNSDoneInfo) {
}

func (nopTraceLogger) DNSStart(httptrace.DNSStartInfo) {
}

func (nopTraceLogger) GetConn(hostPort string) {
}

func (nopTraceLogger) Got1xxResponse(code int, header textproto.MIMEHeader) error {
	return nil
}

func (nopTraceLogger) GotConn(httptrace.GotConnInfo) {
}

func (nopTraceLogger) TLSHandshakeDone(tls.ConnectionState, error) {
}

func (nopTraceLogger) TLSHandshakeStart() {
}

func (nopTraceLogger) Wait100Continue() {
}

func (nopTraceLogger) WroteHeaderField(key string, value []string) {
}

func (nopTraceLogger) WroteRequest(httptrace.WroteRequestInfo) {
}

func (nopTraceLogger) StartRequest(*http.Request) {
}

func (nopTraceLogger) Redirected(*http.Request, []*http.Request, error) {
}

func (nopTraceLogger) ResponseDone(*http.Response, error) {
}

func (t *traceableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	ctx = httptrace.WithClientTrace(ctx, newClientTrace(t.logger))
	req = req.WithContext(ctx)

	t.logger.StartRequest(req)

	rsp, err := t.Transport.RoundTrip(req)
	t.logger.ResponseDone(rsp, err)
	return rsp, err
}

func indexTraceString(j string) int {
	for i, s := range traceString {
		if s == j {
			return i
		}
	}
	return -1
}

func traceTemplate(ctx context.Context) *template.Template {
	c := cli.FromContext(ctx)
	tpl := c.Template("HTTPTrace")
	if tpl != nil {
		return tpl.Template
	}

	return outputTemplate
}

var (
	_ flag.Value = (*TraceLevel)(nil)
)
