package httpclient

import (
	"errors"
	"flag"
	"net/url"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

// URLValue provides ergonomics for entering URLs as values.  When the text looks like
// a port (e.g. :8080), the URL is interpeted as localhost.  When the text looks like a
// hostname, the prefix http:// is preprended.
type URLValue struct {
	loc string
}

// HeaderValue provides an instance of a value in a header
type HeaderValue struct {
	// Name in the header
	Name string
	// Value in the header
	Value string
}

// UserInfo provides the username and password
type UserInfo struct {
	User        string
	Password    string
	HasPassword bool
}

// VirtualPath identifies the mapping between a request path and a real file system path
type VirtualPath struct {
	// RequestPath identifies the request path prefix to match
	RequestPath string

	// PhysicalPath identifies the real path that contains the resource to be served
	PhysicalPath string
}

type headerValueCounter struct {
	count int
}

// NewURLValue creates a new URLValue from a string
func NewURLValue(loc string) *URLValue {
	return &URLValue{loc}
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

// URL interprets the value as a URL
func (u *URLValue) URL() (*url.URL, error) {
	return url.Parse(fixupAddress(u.loc))
}

// URITemplate interprets the value as a URI Template
func (u *URLValue) URITemplate() (*uritemplates.URITemplate, error) {
	return uritemplates.Parse(fixupAddress(u.loc))
}

func (u *URLValue) Set(arg string) error {
	u.loc = arg
	return nil
}

func (u *URLValue) String() string {
	return u.loc
}

func (u *URLValue) Reset() {
	// To facilitate re-use
	u.loc = ""
}

func (u *URLValue) Copy() *URLValue {
	res := *u
	return &res
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

func (v *VirtualPath) Set(arg string) error {
	a, err := ParseVirtualPath(arg)
	if err != nil {
		return err
	}
	*v = a
	return nil
}

func (v VirtualPath) String() string {
	return v.RequestPath + ":" + v.PhysicalPath
}

func (v *VirtualPath) NewCounter() cli.ArgCounter {
	return cli.ArgCount(1)
}

func (v *HeaderValue) Reset() {
	// Reset is required to facilitate use of EachOccurrence
	v.Name = ""
	v.Value = ""
}

func (v *HeaderValue) Copy() *HeaderValue {
	res := *v
	return &res
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
	if v.count == 0 {
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
			v.count++
		}
		return nil
	case 1:
		v.count++
		return nil
	case 2:
		v.count++
		return cli.EndOfArguments
	}

	return errors.New("too many arguments to header")
}

func ParseVirtualPath(v string) (VirtualPath, error) {
	r, s, ok := strings.Cut(string(v), ":")
	if ok {
		if s == "" {
			s = "."
		}
	} else {
		s = r
	}
	return VirtualPath{
		RequestPath:  r,
		PhysicalPath: s,
	}, nil
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
