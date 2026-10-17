// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"net/url"
	"path"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

// LocationResolver provides the logic of the how the URL to request
// is resolved.
type LocationResolver interface {
	// Add a location to be resolved.  This is typically in the syntax
	// of either a URL or a URI template.
	Add(location string) error
	// Add a variable that can be applied to templates.
	AddVar(name string, value any) error
	// Set the base URL used for resolving relative URLs.  The base is
	// not interpreted until it is needed, so it can be in the same simplified
	// syntax as locations that are added.  If a base has already been set, the
	// given base is resolved relative to it.
	SetBaseURL(base string) error
	// SetSecure sets whether locations which don't specify a scheme
	// are assumed to be https instead of http.
	SetSecure(secure bool) error
	// Resolve the list of locations. Each location is represented
	// as a URL and the context the client should use to issue the
	// request.
	Resolve(context.Context) ([]Location, error)
	// Vars gets the variables that were added.
	Vars() map[string]any
	// BaseURL retrieves the base URL.
	BaseURL() string
	// Secure retrieves whether locations which don't specify a scheme are
	// assumed to be https.
	Secure() bool
}

// Location specifies the request location.  A location comprises a URL that
// will be requested by the HTTP client and a context function that helps initialize
// any context values that might be needed such as by middleware.  This is an
// indirection typically used to provide behavior dependent upon the request URL.
// If Location also implements client Middleware, it will be the first middleware
// function used when the client fetches the request
type Location interface {
	// URL derives a context that should be used for the client request for the
	// given URL, the URL itself to use.  An error can be returned, usually if the
	// input context ctx is lacking a required service.
	URL(ctx context.Context) (context.Context, *url.URL, error)
}

type urlLocation struct {
	u *url.URL
}

type templateLocation struct {
	template string
	varfn    func(context.Context) any
}

type autoLocation struct {
	url    string
	varfn  func(context.Context) any
	secure bool
}

type defaultLocationResolver struct {
	urls   []string
	vars   uritemplates.Vars
	bases  []string
	secure bool
}

// NewDefaultLocationResolver provides a location resolver that
// supports relative addressing and URI templates
func NewDefaultLocationResolver() LocationResolver {
	return &defaultLocationResolver{
		vars: uritemplates.Vars{},
	}
}

// NewURLLocation provides a basic implementation of Location for a URL.  It isn't
// dependent upon and makes no modifications to the context
func NewURLLocation(u *url.URL) Location {
	return urlLocation{u}
}

// NewLocation provides the default implementation, which is a URL, URI template,
// or simplified human representation of a local URL automatically determined by the
// context. If varfn is specified,
// it is a function that provides URI template variables using the acceptable
// types supported by [URITemplate.Expand].  Only when varfn is present, the URL is
// interpreted as a URI template. Otherwise, it is interpreted as a typical
// URL except as a special case the syntax :port can be used to refer to a port
// listening to an address on localhost or if no scheme is present, http://
// is implicitly prepended.  This interpretation is deferred until the URL
// method of the location is called.
func NewLocation(url string, varfn func(context.Context) any) Location {
	return newLocation(url, varfn, false)
}

func newLocation(url string, varfn func(context.Context) any, secure bool) Location {
	return autoLocation{
		url:    url,
		varfn:  varfn,
		secure: secure,
	}
}

// NewURITemplateLocation provides a location based on a URI template. varfn
// is a function that provides URI template variables using the acceptable
// types supported by [URITemplate.Expand].
func NewURITemplateLocation(template string, varfn func(context.Context) any) Location {
	return templateLocation{
		template: template,
		varfn:    varfn,
	}
}

func (d *defaultLocationResolver) Add(location string) error {
	d.urls = append(d.urls, location)
	return nil
}

func (d *defaultLocationResolver) AddVar(name string, value any) error {
	d.vars.Add(uritemplates.NewVar(name, value))
	return nil
}

func (d *defaultLocationResolver) Vars() map[string]any {
	return d.vars
}

func (d *defaultLocationResolver) BaseURL() string {
	base, _ := d.resolveBase()
	return base
}

func (d *defaultLocationResolver) SetBaseURL(base string) error {
	d.bases = append(d.bases, base)
	return nil
}

func (d *defaultLocationResolver) Secure() bool {
	return d.secure
}

func (d *defaultLocationResolver) SetSecure(secure bool) error {
	d.secure = secure
	return nil
}

func (d *defaultLocationResolver) resolveBase() (string, error) {
	var base *url.URL
	for i, b := range d.bases {
		if i == 0 {
			var err error
			base, err = url.Parse(fixupAddress(b, d.secure))
			if err != nil {
				return "", err
			}
			continue
		}

		ref, err := url.Parse(b)
		if err != nil {
			return "", err
		}
		base = base.ResolveReference(ref)
	}
	if base == nil {
		return "", nil
	}
	return base.String(), nil
}

func (d *defaultLocationResolver) Resolve(context.Context) ([]Location, error) {
	locations := make([]Location, len(d.urls))
	varfn := func(context.Context) any {
		return d.Vars()
	}
	if !d.isURITemplates() {
		varfn = nil
	}

	var base string
	for i := range d.urls {
		u := resolveURL(base, d.urls[min(i, 1):i+1])
		locations[i] = newLocation(u, varfn, d.secure)

		if i == 0 {
			base = d.urls[0]
		}
	}

	return locations, nil
}

func (d *defaultLocationResolver) isURITemplates() bool {
	return len(d.vars) > 0
}

func (l urlLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	return ctx, l.u, nil
}

func (l templateLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	tt, err := uritemplates.Parse(l.template)
	if err != nil {
		return ctx, nil, err
	}

	vars := l.varfn(ctx)
	expanded, err := tt.Expand(vars)
	if err != nil {
		return ctx, nil, err
	}
	u, err := url.Parse(expanded)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, u.JoinPath(), nil
}

func (a autoLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	address := fixupAddress(a.url, a.secure)
	if a.varfn == nil {
		u, err := url.Parse(address)
		if err != nil {
			return nil, nil, err
		}

		return NewURLLocation(u).URL(ctx)
	}

	return NewURITemplateLocation(address, a.varfn).URL(ctx)
}

func resolveURL(base string, prefix []string) string {
	// Treat as absolute URI when it is qualified
	if len(prefix) > 0 && looksLikeURLPattern.MatchString(prefix[0]) {
		base = prefix[0]
		prefix = prefix[1:]
	}

	if base != "" && len(prefix) > 0 {
		base += "/"
	}

	return base + path.Join(prefix...)
}
