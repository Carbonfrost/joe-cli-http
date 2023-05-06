package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

const defaultAccessLog = `- - [%(start:02/Jan/2006 15:04:05)] "%(method) %(urlPath) %(protocol)" %(status) -\n`

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
	},
}

// NewRequestLogger provides handler middleware to write to access log
func NewRequestLogger(format string, next http.Handler) http.Handler {
	if format == "" {
		format = defaultAccessLog
	}
	logFormat := expr.Compile(format)
	return requestLoggerHandler(next, logFormat)
}

// NewPingHandler provides a handler which simply replies with a message
func NewPingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ping\n"))
	})
}

// FileServerHandlerSpec creates a file server.  The physical path in the virtual path
// specifies the base directory for the file server.  An option named
// no_directory_listing controls whether the directory listing response is served.
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

func requestLoggerHandler(next http.Handler, format *expr.Pattern) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ww := newWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()

		defer func() {
			vars := map[string]interface{}{
				"bytesWritten": ww.BytesWritten(),
				"duration":     time.Since(t1),
				"end":          time.Now(),
				"headers":      ww.Header(),
				"method":       r.Method,
				"protocol":     r.Proto,
				"start":        t1,
				"status":       ww.Status(),
				"urlPath":      r.URL.Path,
			}
			expr.Fprint(os.Stderr, format, expr.ExpandMap(vars))
		}()

		next.ServeHTTP(ww, r)
	}
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
	for k, v := range src {
		dst[k] = v
	}
}
