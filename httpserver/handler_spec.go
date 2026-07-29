package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

// HandlerSpec creates a handler from a virtual path.  The virtual path
// defines how the handler works.  Typically, the physical path
// identifies a useful feature or the location of a file,
// and the options may be used for any purpose of customization.
type HandlerSpec func(context.Context, httpclient.VirtualPath) (http.Handler, error)

// FileServerHandlerSpec creates a file server.  The physical path in the virtual path
// specifies the base directory for the file server.  An option named
// hide_directory_listing controls whether the directory listing response is served.
// The handler also consults the server for whether directory listings can be served.
func FileServerHandlerSpec() HandlerSpec {
	return func(_ context.Context, vp httpclient.VirtualPath) (http.Handler, error) {
		dict := map[string]any{
			"directory": vp.PhysicalPath,
		}
		for k, v := range vp.Options {
			dict[k] = v
		}
		h, err := provider.FactoryOf(newFileServerHandlerWithOpts).New(dict)
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
		reg, ok := provider.Services(ctx).LookupRegistry(name)
		if !ok {
			return nil, fmt.Errorf("no handler for %q", name)
		}
		h, err := reg.New(vp.PhysicalPath, vp.Options)
		if err != nil {
			return nil, err
		}
		if h == nil {
			return nil, fmt.Errorf("no handler for %q", name)
		}
		if spec, ok := h.(HandlerSpec); ok {
			return spec(ctx, vp)
		}
		return http.StripPrefix(vp.RequestPath, h.(http.Handler)), err
	}
}
