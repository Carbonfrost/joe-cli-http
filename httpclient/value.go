package httpclient

import (
	"flag"
	"net/url"
	"strings"
)

// URLValue provides ergonomics for entering URLs as values.  When the text looks like
// a port (e.g. :8080), the URL is interpeted as localhost.  When the text looks like a
// hostname, the prefix http:// is preprended.
type URLValue struct {
	url.URL
}

func (u *URLValue) Set(arg string) error {
	v, err := url.Parse(fixupAddress(arg))
	if err == nil {
		u.URL = *v
	}
	return err
}

func (u *URLValue) String() string {
	return u.URL.String()
}

func fixupAddress(addr string) string {
	if strings.HasPrefix(addr, ":") {
		addr = "http://localhost" + addr
	}

	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	return addr
}

var _ flag.Value = (*URLValue)(nil)
