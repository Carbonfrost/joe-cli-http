package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"maps"
)

const defaultAccessLog = `%(accessLog.default)\n`

// HandlerSpec creates a handler from a virtual path.  The virtual path
// defines how the handler works.  Typically, the physical path
// identifies a useful feature or the location of a file,
// and the options may be used for any purpose of customization.
type HandlerSpec func(context.Context, httpclient.VirtualPath) (http.Handler, error)

// HandlerRegistry provides the default handler registry.
var HandlerRegistry = &provider.Registry{
	Name: "handlers",
	Providers: provider.Details{
		"ping": {
			Factory: provider.Factory(newPingHandlerWithOpts),
		},
		"file": {
			Factory: provider.Factory(newFileServerHandlerWithOpts),
		},
		"redirect": {
			Factory: provider.Factory(newRedirectServerHandlerWithOpts),
		},
	},
}

// NewRequestLogger provides handler middleware to write to access log
func NewRequestLogger(format string, out io.Writer, next http.Handler) http.Handler {
	if format == "" {
		format = defaultAccessLog
	}
	logFormat := expr.Compile(format)
	return requestLoggerHandler(out, next, logFormat)
}

// NewPingHandler provides a handler which simply replies with a message
func NewPingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ping\n"))
	})
}

// NewHeaderMiddleware provides handler middleware which simply adds the given
// header
func NewHeaderMiddleware(name, value string) func(http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(name, value)
			inner.ServeHTTP(w, r)
		})
	}
}

// FileServerHandlerSpec creates a file server.  The physical path in the virtual path
// specifies the base directory for the file server.  An option named
// hide_directory_listing controls whether the directory listing response is served.
// The handler also consults the server for whether directory listings can be served.
func FileServerHandlerSpec() HandlerSpec {
	return func(_ context.Context, vp httpclient.VirtualPath) (http.Handler, error) {
		dict := map[string]string{
			"directory": vp.PhysicalPath,
		}
		update(dict, vp.Options)
		h, err := provider.Factory(newFileServerHandlerWithOpts)(dict)
		if err != nil {
			return nil, err
		}
		return http.StripPrefix(vp.RequestPath, h.(http.Handler)), err
	}
}

// RegistryHandlerSpec creates a handler by looking it up as a provider in the
// registry that is named.  The physical path in the virtual path specifies the
// name of the provider which is used.  The virtual path's options are propagated
// to the registry factory function.
func RegistryHandlerSpec(name string) HandlerSpec {
	return func(ctx context.Context, vp httpclient.VirtualPath) (http.Handler, error) {
		reg := provider.Services(cli.FromContext(ctx)).Registry(name)
		h, err := reg.New(vp.PhysicalPath, vp.Options)
		if err != nil {
			return nil, err
		}
		if h == nil {
			return nil, fmt.Errorf("no handler for %q", name)
		}
		return http.StripPrefix(vp.RequestPath, h.(http.Handler)), err
	}
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

func requestLoggerHandler(out io.Writer, next http.Handler, format *expr.Pattern) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ww := newWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()

		defer func() {
			expander := expr.ComposeExpanders(
				expr.ExpandGlobals,
				expr.ExpandColors,
				ExpandRequest(r, ww),
				expandTiming(t1, time.Now()),
			)
			expr.Fprint(out, format, expander)
		}()

		next.ServeHTTP(ww, r)
	}
}

func ExpandRequest(r *http.Request, ww wrapResponseWriter) expr.Expander {
	return expr.ComposeExpanders(func(s string) any {
		switch s {
		case "bytesWritten":
			return ww.BytesWritten()
		case "method":
			return r.Method
		case "protocol":
			return r.Proto
		case "statusCode":
			return ww.Status()
		case "status":
			return fmt.Sprint(ww.Status(), " ", http.StatusText(ww.Status()))
		case "urlPath":
			return r.URL.Path
		case "header":
			var buf bytes.Buffer
			ww.Header().Write(&buf)
			return buf.String()
		}
		return nil
	}, expr.Prefix("header", httpclient.ExpandHeader(ww.Header())))
}

func expandTiming(start, end time.Time) expr.Expander {
	return expr.ExpandMap(map[string]any{
		"duration": end.Sub(start),
		"end":      end,
		"start":    start,
	})
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

func update(dst, src map[string]string) {
	maps.Copy(dst, src)
}
