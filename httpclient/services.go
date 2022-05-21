package httpclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

type contextKey string

const servicesKey contextKey = "httpclient_services"

type ContextServices struct {
	Client         *http.Client
	Request        *http.Request
	IncludeHeaders bool

	dialer    *net.Dialer
	dnsDialer *net.Dialer
	tlsConfig *tls.Config
}

func Do(c *cli.Context) (*Response, error) {
	return Services(c).Do(c)
}

func Services(c context.Context) *ContextServices {
	return c.Value(servicesKey).(*ContextServices)
}

func (h *ContextServices) SetTraceLevel(v TraceLevel) {
	h.Client.Transport.(*traceableTransport).level = v
}

func (h *ContextServices) Do(ctx context.Context) (*Response, error) {
	h.Request = h.Request.WithContext(ctx)

	resp, err := h.Client.Do(h.Request)
	if err != nil {
		return nil, err
	}
	return &Response{
		Response: resp,
	}, nil
}

func (h *ContextServices) TLSConfig() *tls.Config {
	return h.tlsConfig
}

func (h *ContextServices) Dialer() *net.Dialer {
	return h.dialer
}

func (h *ContextServices) DNSDialer() *net.Dialer {
	return h.dnsDialer
}

func (h *ContextServices) SetMethod(s string) error {
	h.Request.Method = s
	return nil
}

func (h *ContextServices) SetFollowRedirects(value bool) error {
	if value {
		h.Client.CheckRedirect = nil // default policy to follow 10 times
		return nil
	}

	// Follow no redirects
	h.Client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return nil
}

func (h *ContextServices) SetUserAgent(value string) error {
	h.Request.Header.Set("User-Agent", value)
	return nil
}

func (h *ContextServices) SetURL(u *URLValue) error {
	h.Request.URL = &u.URL
	h.Request.Host = u.Host
	return nil
}

func (h *ContextServices) SetIncludeHeaders(v bool) error {
	h.IncludeHeaders = v
	return nil
}

func (h *ContextServices) SetInsecureSkipVerify(v bool) error {
	h.TLSConfig().InsecureSkipVerify = v
	return nil
}

func (h *ContextServices) SetCiphers(ids *CipherSuites) error {
	h.TLSConfig().CipherSuites = []uint16(*ids)
	return nil
}

func (h *ContextServices) SetPreferGoDialer(v bool) error {
	h.Dialer().Resolver.PreferGo = v
	return nil
}

func (h *ContextServices) SetStrictErrorsDNS(v bool) error {
	h.Dialer().Resolver.StrictErrors = v
	return nil
}

func (h *ContextServices) SetDisableDialKeepAlive(v bool) error {
	if v {
		h.Dialer().KeepAlive = time.Duration(-1)
	}
	return nil
}

func (h *ContextServices) SetHeader(n *cli.NameValue) error {
	name, value := n.Name, n.Value
	// If a colon was used, then assume the syntax Header:Value was used.
	if strings.Contains(name, ":") && value == "true" {
		args := strings.SplitN(name, ":", 2)
		name = args[0]
		value = args[1]
	}
	h.Request.Header.Set(name, value)
	return nil
}

func (h *ContextServices) SetInterface(value string) error {
	addr, err := resolveInterface(value)
	if err != nil {
		return err
	}
	h.Dialer().LocalAddr = addr
	return nil
}

func (h *ContextServices) SetDNSInterface(value string) error {
	if value == "" {
		return nil
	}
	addr, err := resolveInterface(value)
	if err != nil {
		return err
	}
	h.DNSDialer().LocalAddr = addr
	return nil
}

func (h *ContextServices) SetDialTimeout(v time.Duration) error {
	h.Dialer().Timeout = v
	return nil
}

func (h *ContextServices) SetDialKeepAlive(v time.Duration) error {
	h.Dialer().KeepAlive = v
	return nil
}
