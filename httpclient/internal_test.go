package httpclient // intentional

import (
	"crypto/tls"
	"io"
	"net"
	"net/url"
)

type ClientAttributes struct {
	TLSConfig         *tls.Config
	Dialer            *net.Dialer
	DNSDialer         *net.Dialer
	BaseURL           string
	BodyContent       Content
	BodyContentString string
	BodyForm          url.Values
	IncludeHeaders    bool
}

func Attributes(c *Client) *ClientAttributes {
	return &ClientAttributes{
		TLSConfig:   c.TLSConfig(),
		Dialer:      c.Dialer(),
		DNSDialer:   c.DNSDialer(),
		BodyContent: c.BodyContent,
		BodyContentString: func() string {
			if c.BodyContent == nil {
				return "<nil>"
			}
			body, _ := io.ReadAll(c.BodyContent.Read())
			return string(body)
		}(),

		BaseURL: func() string {
			if c.LocationResolver == nil {
				return ""
			}
			return c.LocationResolver.(*defaultLocationResolver).base.String()
		}(),
		BodyForm: func() url.Values {
			m := url.Values{}
			for _, k := range c.bodyForm {
				m.Set(k.Name, k.Value)
			}
			return m
		}(),
		IncludeHeaders: c.IncludeHeaders,
	}
}
