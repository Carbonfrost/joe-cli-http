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
	"os"
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
}

type nopTraceLogger struct{}

type filteredTraceLogger struct {
	TraceLogger
	flags TraceLevel
}

type defaultTraceLogger struct {
	out      io.Writer
	template *template.Template
}

type traceableTransport struct {
	level     TraceLevel
	Transport http.RoundTripper
}

const (
	TraceConnections = TraceLevel(1 << iota)
	TraceRequestHeaders
	TraceDNS
	TraceTLS
	TraceHTTP1XX
	TraceRequestBody
	TraceResponseStatus

	// TraceOff causes all tracing to be switched off
	TraceOff TraceLevel = 0

	// TraceOn enables tracing of when connections are made and header
	TraceOn = TraceConnections | TraceRequestHeaders | TraceResponseStatus

	// TraceVerbose enables tracing of DNS, TLS, HTTP 1xx responses
	TraceVerbose = TraceOn | TraceDNS | TraceTLS | TraceHTTP1XX
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
		TraceOff,
	}
)

var (
	funcs = template.FuncMap{

		// Stub color functions when used outside of Joe-cli
		"Gray":       func(s ...interface{}) string { return fmt.Sprint(s...) },
		"Magenta":    func(s ...interface{}) string { return fmt.Sprint(s...) },
		"Blue":       func(s ...interface{}) string { return fmt.Sprint(s...) },
		"ResetColor": func(s ...interface{}) string { return fmt.Sprint(s...) },
		"Color":      func(s ...interface{}) string { return fmt.Sprint(s...) },
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
{{ Gray }}> {{ .Key | Magenta }}: {{ .Value | Join ", " | Gray }}{{ResetColor}}
{{ end -}}

{{- define "StartRequest" -}}
{{ Gray }}> {{ .Method | Magenta }} {{ .Path }} {{ .Proto }}{{ ResetColor }}
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
{{ .Red }}{{ .Error }}{{ ResetColor }}
{{ end -}}

`
)

// WithTraceLevel provides filtering at the specified level
func WithTraceLevel(l TraceLogger, v TraceLevel) TraceLogger {
	if v == TraceOff {
		return nopTraceLogger{}
	}
	return &filteredTraceLogger{l, v}
}

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
		j = strings.ToLower(strings.TrimSpace(j))
		in := indexTraceString(j)
		if in < 0 {
			return fmt.Errorf("unknown trace level %q", arg)
		}
		res |= traceEnum[res]
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
			l = l & ^e
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

func (l *filteredTraceLogger) ConnectDone(network, addr string, err error) {
	if !l.flags.connections() {
		return
	}
	l.TraceLogger.ConnectDone(network, addr, err)
}

func (l *filteredTraceLogger) ConnectStart(network, addr string) {
	if !l.flags.connections() {
		return
	}
	l.TraceLogger.ConnectStart(network, addr)
}

func (l *filteredTraceLogger) DNSDone(info httptrace.DNSDoneInfo) {
	if !l.flags.dns() {
		return
	}
	l.TraceLogger.DNSDone(info)
}

func (l *filteredTraceLogger) DNSStart(info httptrace.DNSStartInfo) {
	if !l.flags.dns() {
		return
	}
	l.TraceLogger.DNSStart(info)
}

func (l *filteredTraceLogger) GetConn(hostPort string) {
	if !l.flags.connections() {
		return
	}
	l.TraceLogger.GetConn(hostPort)
}

func (l *filteredTraceLogger) Got1xxResponse(code int, header textproto.MIMEHeader) (err error) {
	if !l.flags.http1xx() {
		return
	}
	return l.TraceLogger.Got1xxResponse(code, header)
}

func (l *filteredTraceLogger) GotConn(info httptrace.GotConnInfo) {
	if !l.flags.connections() {
		return
	}
	l.TraceLogger.GotConn(info)
}

func (l *filteredTraceLogger) TLSHandshakeDone(state tls.ConnectionState, err error) {
	if !l.flags.tls() {
		return
	}
	l.TraceLogger.TLSHandshakeDone(state, err)
}

func (l *filteredTraceLogger) TLSHandshakeStart() {
	if !l.flags.tls() {
		return
	}
	l.TraceLogger.TLSHandshakeStart()
}

func (l *filteredTraceLogger) Wait100Continue() {
	if !l.flags.http1xx() {
		return
	}
	l.TraceLogger.Wait100Continue()
}

func (l *filteredTraceLogger) WroteHeaderField(key string, value []string) {
	if !l.flags.requestHeaders() {
		return
	}
	l.TraceLogger.WroteHeaderField(key, value)
}

func (l *filteredTraceLogger) WroteRequest(info httptrace.WroteRequestInfo) {
	if !l.flags.requestBody() {
		return
	}
	l.TraceLogger.WroteRequest(info)
}

func (l *defaultTraceLogger) ConnectDone(network, addr string, err error) {
}

func (l *defaultTraceLogger) ConnectStart(network, addr string) {
}

func (l *defaultTraceLogger) DNSDone(info httptrace.DNSDoneInfo) {
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
	l.render("DNSStart", struct {
		Host string
	}{
		Host: info.Host,
	})
}

func (l *defaultTraceLogger) GetConn(hostPort string) {
	l.render("GetConn", struct {
		HostPort string
	}{
		HostPort: hostPort,
	})
}

func (l *defaultTraceLogger) Got1xxResponse(code int, header textproto.MIMEHeader) (err error) {
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
	l.render("TLSHandshakeStart", nil)
}

func (l *defaultTraceLogger) Wait100Continue() {
}

func (l *defaultTraceLogger) WroteHeaderField(key string, value []string) {
	l.render("WroteHeaderField", struct {
		Key   string
		Value []string
	}{
		Key:   key,
		Value: value,
	})
}

func (l *defaultTraceLogger) WroteRequest(info httptrace.WroteRequestInfo) {
}

func (l *defaultTraceLogger) render(fn string, data interface{}) {
	l.template.ExecuteTemplate(l.out, fn, data)
}

func (l *defaultTraceLogger) StartRequest(req *http.Request) {
	path := req.URL.Path
	proto := req.Proto
	if path == "" {
		path = "/"
	}
	if proto == "" {
		proto = "HTTP/1.1"
	}
	l.render("StartRequest", struct {
		Method string
		Path   string
		Proto  string
	}{
		Method: req.Method,
		Path:   path,
		Proto:  proto,
	})
}

func (l *defaultTraceLogger) ResponseDone(resp *http.Response, err error) {
	if resp == nil || err != nil {
		l.render("GenericError", struct {
			Error error
		}{
			Error: err,
		})
		return
	}

	l.render("StatusCode", struct {
		Status httpStatus
	}{
		Status: httpStatus(resp.StatusCode),
	})
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

func (nopTraceLogger) ResponseDone(*http.Response, error) {
}

func (t *traceableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	tt := traceTemplate(ctx)

	logger := WithTraceLevel(
		&defaultTraceLogger{template: tt, out: os.Stderr}, t.level,
	)

	ctx = httptrace.WithClientTrace(ctx, newClientTrace(logger))
	req = req.WithContext(ctx)

	logger.StartRequest(req)

	rsp, err := t.Transport.RoundTrip(req)
	logger.ResponseDone(rsp, err)
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
	if c, ok := ctx.(*cli.Context); ok {
		tpl := c.Template("HTTPTrace")
		if tpl != nil {
			return tpl.Template
		}
	}

	return outputTemplate
}

var (
	_ flag.Value = (*TraceLevel)(nil)
)
