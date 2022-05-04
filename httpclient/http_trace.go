package httpclient

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"os"
	"strconv"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

// TraceLevel indicates the amount of client tracing to generate
type TraceLevel int

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
}

type nopTraceLogger struct{}

type filteredTraceLogger struct {
	TraceLogger
	flags TraceLevel
}

type defaultTraceLogger struct {
	out io.Writer
}

type traceableTransport struct {
	level     TraceLevel
	Transport *http.Transport
}

const (
	TraceConnections = TraceLevel(1 << iota)
	TraceRequestHeaders
	TraceDNS
	TraceTLS
	TraceHTTP1XX
	TraceRequestBody

	// TraceOff causes all tracing to be switched off
	TraceOff TraceLevel = 0

	// TraceOn enables tracing of when connections are made and header
	TraceOn = TraceConnections | TraceRequestHeaders

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
		TraceOff,
	}
)

// WithTraceLevel provides filtering at the specified level
func WithTraceLevel(l TraceLogger, v TraceLevel) TraceLogger {
	if v == TraceOff {
		return nopTraceLogger{}
	}
	return &filteredTraceLogger{l, v}
}

func newClientTrace(logger TraceLogger) *httptrace.ClientTrace {
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
		WroteHeaderField:  logger.WroteHeaderField,
		WroteRequest:      logger.WroteRequest,
	}
}

// SetTraceLevel provides the action for a flag which sets the trace level
// corresponding to the flag's value.
func SetTraceLevel() cli.Action {
	return cli.Setup{
		Uses: cli.ActionFunc(func(c *cli.Context) error {
			f := c.Flag()
			if f.Name == "" {
				f.Name = "trace"
			}
			f.EnvVars = []string{"HTTP_CLIENT_TRACE_LEVEL"}
			f.Value = new(TraceLevel)
			return nil
		}),
		Action: func(c *cli.Context) {
			level := *c.Value("").(*TraceLevel)
			Services(c).SetTraceLevel(level)
		},
	}
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

	l.log("* Resolved to ")
	for i, addr := range info.Addrs {
		if i > 0 {
			l.log(", ")
		}
		l.logf("%s", addr.String())
	}
	l.logln()
}

func (l *defaultTraceLogger) DNSStart(info httptrace.DNSStartInfo) {
	l.logf("* Resolving name %s...\n", info.Host)
}

func (l *defaultTraceLogger) GetConn(hostPort string) {
	l.logf("* Connecting to %s...\n", hostPort)
}

func (l *defaultTraceLogger) Got1xxResponse(code int, header textproto.MIMEHeader) (err error) {
	l.logf("< Got %d %s", code, header)
	return nil
}

func (l *defaultTraceLogger) GotConn(info httptrace.GotConnInfo) {
	remote := info.Conn.RemoteAddr()
	res := make([]string, 0)

	res = append(res, "on "+info.Conn.LocalAddr().String())
	if info.Reused {
		res = append(res, "reused")
	}

	l.logf("* Connected to %s (%s)\n", remote.String(), strings.Join(res, ", "))
}

func (l *defaultTraceLogger) TLSHandshakeDone(state tls.ConnectionState, err error) {
}

func (l *defaultTraceLogger) TLSHandshakeStart() {
	l.logln("* Establishing TLS connection...")
}

func (l *defaultTraceLogger) Wait100Continue() {
}

func (l *defaultTraceLogger) WroteHeaderField(key string, value []string) {
	l.logf("> %s: %s\n", key, strings.Join(value, " "))
}

func (l *defaultTraceLogger) WroteRequest(info httptrace.WroteRequestInfo) {
}

func (l *defaultTraceLogger) logf(format string, a ...interface{}) {
	fmt.Fprintf(l.out, format, a...)
}

func (l *defaultTraceLogger) logln(s ...interface{}) {
	fmt.Fprintln(l.out, s...)
}

func (l *defaultTraceLogger) log(s string) {
	fmt.Fprint(l.out, s)
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

func (t *traceableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	logger := WithTraceLevel(
		&defaultTraceLogger{out: os.Stderr}, t.level,
	)
	ctx := req.Context()
	ctx = httptrace.WithClientTrace(ctx, newClientTrace(logger))
	req = req.WithContext(ctx)

	return http.DefaultTransport.RoundTrip(req)
}

func indexTraceString(j string) int {
	for i, s := range traceString {
		if s == j {
			return i
		}
	}
	return -1
}

var (
	_ flag.Value = (*TraceLevel)(nil)
)
