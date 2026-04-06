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
	// Set the base URL used for resolving relative URLs.
	SetBaseURL(base *url.URL) error
	// Resolve the list of locations. Each location is represented
	// as a URL and the context the client should use to issue the
	// request.
	Resolve(context.Context) ([]Location, error)
	// Vars gets the variables that were added.
	Vars() map[string]any
	// BaseURL retrievess the base URL.
	BaseURL() *url.URL
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
	url   string
	varfn func(context.Context) any
}

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
// is implicitly prepended.
func NewLocation(url string, varfn func(context.Context) any) Location {
	return autoLocation{
		url:   fixupAddress(url),
		varfn: varfn,
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

func (d *defaultLocationResolver) BaseURL() *url.URL {
	return d.base
}

func (d *defaultLocationResolver) SetBaseURL(base *url.URL) error {
	if d.base == nil {
		d.base = base
		return nil
	}

	d.base = d.base.ResolveReference(base)
	return nil
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
		locations[i] = NewLocation(u, varfn)

		if i == 0 {
			base = fixupAddress(d.urls[0])
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
	if a.varfn == nil {
		u, err := url.Parse(a.url)
		if err != nil {
			return nil, nil, err
		}

		return NewURLLocation(u).URL(ctx)
	}

	return NewURITemplateLocation(a.url, a.varfn).URL(ctx)
}

func resolveURL(base string, prefix []string) string {
	// Treat as absolute URI when it is qualified
	if len(prefix) > 0 && looksLikeURLPattern.MatchString(prefix[0]) {
		base = prefix[0]
		prefix = prefix[1:]
	}

	if base != "" && len(prefix) > 0 {
		base = base + "/"
	}

	return base + path.Join(prefix...)
}
