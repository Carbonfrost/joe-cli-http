package httpclient // intentional

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/Carbonfrost/joe-cli"
)

type ClientAttributes struct {
	TLSConfig              *tls.Config
	Dialer                 *net.Dialer
	DNSDialer              *net.Dialer
	BaseURL                string
	BodyContent            Content
	BodyContentString      string
	BodyForm               url.Values
	IncludeResponseHeaders bool
	CheckRedirect          any
	Request                *RequestAttributes

	Downloader               Downloader
	DownloaderWithMiddleware Downloader
}

type RequestAttributes struct {
	Method     string
	URL        *url.URL
	Proto      string
	ProtoMajor int
	ProtoMinor int
	Header     http.Header
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
		IncludeResponseHeaders: c.IncludeResponseHeaders,
		CheckRedirect:          c.Client.CheckRedirect,
		Request:                newRequestAttributes(c.Request),
		Downloader:             c.downloader,
		DownloaderWithMiddleware: c.actualDownloader(&cli.Context{
			Stdout: cli.NewWriter(new(bytes.Buffer)),
		}),
	}
}

func newRequestAttributes(r *http.Request) *RequestAttributes {
	return &RequestAttributes{
		Method:     r.Method,
		URL:        r.URL,
		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,
		Header:     r.Header,
	}
}
