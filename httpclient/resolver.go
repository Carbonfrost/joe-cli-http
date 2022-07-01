package httpclient

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

type InterfaceResolver interface {
	Resolve(string) (*net.TCPAddr, error)
}

type LocationResolver interface {
	Add(location string) error
	AddVar(v *uritemplates.Var) error
	SetBase(base *url.URL) error
	Resolve(context.Context) ([]*url.URL, error)
}

type defaultResolver struct{}

type defaultLocationResolver struct {
	urls []string
	vars uritemplates.Vars
	base *url.URL
}

func NewDefaultLocationResolver() LocationResolver {
	return &defaultLocationResolver{
		vars: uritemplates.Vars{},
	}
}

func (d *defaultLocationResolver) Add(location string) error {
	d.urls = append(d.urls, location)
	return nil
}

func (d *defaultLocationResolver) AddVar(v *uritemplates.Var) error {
	d.vars.Add(v)
	return nil
}

func (d *defaultLocationResolver) SetBase(base *url.URL) error {
	if d.base == nil {
		d.base = base
		return nil
	}

	d.base = d.base.ResolveReference(base)
	return nil
}

func (d *defaultLocationResolver) Resolve(context.Context) ([]*url.URL, error) {
	var locations []string

	if d.isURITemplates() {
		locations = make([]string, len(d.urls))
		for i, u := range d.urls {
			tt, err := uritemplates.Parse(u)
			if err != nil {
				return nil, err
			}

			expanded, err := tt.Expand(d.vars)
			if err != nil {
				return nil, err
			}
			locations[i] = expanded
		}
	} else {
		locations = d.urls
	}

	var err error
	res := make([]*url.URL, len(locations))
	for i, loc := range locations {
		if i == 0 {
			loc = fixupAddress(loc)
		}
		res[i], err = url.Parse(loc)
		if err != nil {
			return nil, err
		}
	}

	// Resolve URLs relative to the previously specified one, and to
	// the base for the first instance
	for i := range res {
		if i > 0 {
			res[i] = res[i-1].ResolveReference(res[i])

		} else if d.base != nil {
			res[i] = d.base.ResolveReference(res[i])
		}
	}
	return res, nil
}

func (d *defaultLocationResolver) isURITemplates() bool {
	return len(d.vars) > 0
}

func (*defaultResolver) Resolve(v string) (*net.TCPAddr, error) {
	ip := net.ParseIP(v)
	if ip != nil {
		return resolveTCP(ip.String())
	}
	eth, err := net.InterfaceByName(v)
	if err != nil {
		return nil, err
	}
	addrs, err := eth.Addrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		if a.Network() == "ip+net" {
			return resolveIPNet(a.String())
		}
	}
	return nil, errors.New("failed to resolve " + v)
}

func resolveTCP(value string) (*net.TCPAddr, error) {
	return net.ResolveTCPAddr("tcp", net.JoinHostPort(value, "0"))
}

func resolveIPNet(value string) (*net.TCPAddr, error) {
	ip, _, err := net.ParseCIDR(value)
	if err != nil {
		return nil, err
	}
	return resolveTCP(ip.String())
}
