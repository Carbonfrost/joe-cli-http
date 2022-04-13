package httpclient

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/Carbonfrost/joe-cli"
)

type contextKey string

const servicesKey contextKey = "httpclient_services"

type ContextServices struct {
	Client    *http.Client
	Dialer    *net.Dialer
	DNSDialer *net.Dialer
	Request   *http.Request

	IncludeHeaders bool
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
		Response:       resp,
		IncludeHeaders: h.IncludeHeaders,
	}, nil
}

func (h *ContextServices) tlsConfig() *tls.Config {
	return h.Client.Transport.(*http.Transport).TLSClientConfig
}
