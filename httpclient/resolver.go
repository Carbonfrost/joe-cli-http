// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"errors"
	"net"
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

// InterfaceResolver resolves the network interface to use
// for connections
type InterfaceResolver interface {
	// Resolve converts a location, typically an IP address
	// or adapter name, into a TCP address
	Resolve(string) (*net.TCPAddr, error)
}

// LocationResolver provides the logic of the how the URL to request
// is resolved.
type LocationResolver interface {
	// Add a location to be resolved.  This is typically in the syntax
	// of either a URL or a URI template.
	Add(location string) error
	// Add a variable that can be applied to templates.
	AddVar(v *uritemplates.Var) error
	// Set the base URL used for resolving relative URLs.
	SetBase(base *url.URL) error
	// Resolve the list of locations. Each location is represented
	// as a URL and the context the client should use to issue the
	// request.
	Resolve(context.Context) ([]Location, error)
}

// Location specifies the request location.  A location comprises a URL that
// will be requested by the HTTP client and a context function that helps initialize
// any context values that might be needed such as by middleware.  This is an
// indirection typically used to provide behavior dependent upon the request URL.
type Location interface {
	// URL derives a context that should be used for the client request for the
	// given URL, the URL itself to use.  An error can be returned, usually if the
	// input context ctx is lacking a required service.
	URL(ctx context.Context) (context.Context, *url.URL, error)
}

type urlLocation struct {
	u *url.URL
}

type defaultResolver struct{}

type defaultLocationResolver struct {
	urls []string
	vars uritemplates.Vars
	base *url.URL
}

// NewDefaultLocationResolver provides a location resolver that
// supports relative addressing and URI templates
func NewDefaultLocationResolver() LocationResolver {
	return &defaultLocationResolver{
		vars: uritemplates.Vars{},
	}
}

// URLLocation provides a basic implementation of Location for a URL.  It isn't
// dependent upon and makes no modifications to the context
func URLLocation(u *url.URL) Location {
	return urlLocation{u}
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

func (d *defaultLocationResolver) Resolve(context.Context) ([]Location, error) {
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

	ll := make([]Location, len(res))
	for i := range res {
		ll[i] = URLLocation(res[i])
	}
	return ll, nil
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

func (l urlLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	return ctx, l.u, nil
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
