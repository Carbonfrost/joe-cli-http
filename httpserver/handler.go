// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

const defaultAccessLog = `%(accessLog.default)\n`

var (
	metaDefaultAccessLog = expander.Compile(
		`- - [%(start:02/Jan/2006 15:04:05)] "%(method:C) %(urlPath) %(protocol)" %(statusCode:C) -`,
	)
)

// HandlerRegistry provides the default handler registry.
var HandlerRegistry = &provider.Registry{
	Name: "handlers",
	Providers: provider.Details{
		"ping": {
			Factory:  provider.FactoryOf(newPingHandlerWithOpts),
			HelpText: "Responds simply Generates a ping",
		},
		"file": {
			Factory:  provider.FactoryOf(newFileServerHandlerWithOpts),
			HelpText: "Serve a particular directory as static files",
			Defaults: map[string]string{
				"directory":              ".",
				"hide_directory_listing": "false",
			},
		},
		"reload": {
			Value: HandlerSpec(func(ctx context.Context, _ httpclient.VirtualPath) (http.Handler, error) {
				return NewReloadHandler(FromContext(ctx)), nil
			}),
			HelpText: "Reloads the server",
		},
		"redirect": {
			Factory:  provider.FactoryOf(newRedirectServerHandlerWithOpts),
			HelpText: "Redirect to the given path and provide a status code",
			Defaults: map[string]string{
				"to":   "/",
				"code": strconv.Itoa(http.StatusTemporaryRedirect),
			},
		},
		"echo": {
			Factory:  provider.FactoryOf(newEchoHandlerWithOpts),
			HelpText: "Reflects out the request and connection information",
			Defaults: map[string]string{
				"failsafe": "false",
			},
			Aliases: []string{"reflect"},
		},
	},
}

type requestLoggerHandler struct {
	mu     *sync.Mutex
	out    io.Writer
	format *expander.Pattern
	next   http.Handler
}

// NewReloadHandler provides a handler which triggers the server to reload all handlers
func NewReloadHandler(s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.ReloadAll()
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// NewPingHandler provides a handler which simply replies with a message
func NewPingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ping\n"))
	})
}

func newFileServerHandlerWithOpts(opts struct {
	Directory            string `mapstructure:"directory"`
	HideDirectoryListing bool   `mapstructure:"hide_directory_listing"`
}) (http.Handler, error) {
	return newFileServerHandler(opts.Directory, opts.HideDirectoryListing), nil
}

func newFileServerHandler(staticDir string, hideDirListing bool) http.Handler {
	result := http.FileServer(http.Dir(staticDir))
	if hideDirListing {
		result = hideListing(result)
	}
	return result
}

func newPingHandlerWithOpts(_ any) (http.Handler, error) {
	return NewPingHandler(), nil
}

func newEchoHandlerWithOpts(opts struct {
	Failsafe bool `mapstructure:"failsafe"`
}) (http.Handler, error) {
	return NewEchoHandler(opts.Failsafe), nil
}

func newRedirectServerHandlerWithOpts(opts struct {
	To   string `mapstructure:"to"`
	Code int    `mapstructure:"code"`
}) (http.Handler, error) {
	code := opts.Code
	if code == 0 {
		code = http.StatusTemporaryRedirect
	}
	return http.RedirectHandler(opts.To, code), nil
}

func newRequestLoggerHandler(out io.Writer, next http.Handler, format *expander.Pattern) http.Handler {
	return &requestLoggerHandler{
		mu:     new(sync.Mutex),
		out:    out,
		next:   next,
		format: format,
	}
}

func (h *requestLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ww := newWrapResponseWriter(w, r.ProtoMajor)
	t1 := time.Now()

	h.next.ServeHTTP(ww, r)

	exp := expander.Compose(
		expr.ExpandGlobals(),
		expander.Colors(),
		ExpandRequest(r, ww),
		expandTiming(t1, time.Now()),
	)
	h.mu.Lock()
	defer h.mu.Unlock()

	h.format.Fprint(h.out, exp)
}

func hideListing(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, req)
	}
}
