package httpclient

import (
	"errors"
	"flag"
	"net/url"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

// URLValue provides ergonomics for entering URLs as values.  When the text looks like
// a port (e.g. :8080), the URL is interpeted as localhost.  When the text looks like a
// hostname, the prefix http:// is preprended.
type URLValue struct {
	url.URL
}

// HeaderValue provides an instance of a value in a header
type HeaderValue struct {
	// Name in the header
	Name string
	// Value in the header
	Value string
}

type UserInfo struct {
	User        string
	Password    string
	HasPassword bool
}

func NewURLValue(u *url.URL) *URLValue {
	return &URLValue{*u}
}

func (*UserInfo) Synopsis() string {
	return "<user:password>"
}

func (u *UserInfo) Set(arg string) error {
	u.User, u.Password, u.HasPassword = strings.Cut(arg, ":")
	return nil
}

func (u *UserInfo) String() string {
	if u.HasPassword {
		return u.User + ":" + u.Password
	}
	return u.User
}

type headerValueCounter struct {
	count int
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

	if addr == "" || strings.HasPrefix(addr, "/") {
		return addr
	}

	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	return addr
}

func (v *HeaderValue) Reset() {
	// Reset is required to faciliate use of EachOccurrence
	v.Name = ""
	v.Value = ""
}

func (v *HeaderValue) Set(arg string) error {
	if v.Name == "" {
		v.Name, v.Value, _ = splitValuePair(arg)
	} else {
		v.Value += arg
	}
	return nil
}

// String obtains the string representation of the name-value pair
func (v *HeaderValue) String() string {
	return cli.Quote(v.Name + ":" + v.Value)
}

func (v *HeaderValue) NewCounter() cli.ArgCounter {
	return &headerValueCounter{}
}

func (v *headerValueCounter) Done() error {
	switch v.count {
	case 0:
		return errors.New("missing name and value")
	}
	return nil
}

func (v *headerValueCounter) Take(arg string, possibleFlag bool) error {
	switch v.count {
	case 0:
		if _, _, hasValue := splitValuePair(arg); hasValue {
			v.count += 2
		} else {
			v.count += 1
		}
		return nil
	case 1:
		v.count += 1
		return nil
	case 2:
		v.count += 1
		return cli.EndOfArguments
	}

	return errors.New("too many arguments to header")
}

func splitValuePair(arg string) (k, v string, hasValue bool) {
	a := cli.SplitList(arg, "=", 2)
	if len(a) == 2 {
		return a[0], strings.TrimSpace(a[1]), true
	}
	a = cli.SplitList(arg, ":", 2)
	if len(a) == 2 {
		return a[0], strings.TrimSpace(a[1]), true
	}

	return a[0], "", false
}

var (
	_ flag.Value = (*URLValue)(nil)
	_ flag.Value = (*HeaderValue)(nil)
	_ flag.Value = (*UserInfo)(nil)
)
