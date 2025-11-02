// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

const (
	listenerCategory = "Listener options"
	advancedCategory = "Advanced options"
	serverCategory   = "Server options"

	allowStartupTime = 1 * time.Second
)

var (
	tagged = cli.Data(SourceAnnotation())
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", "joe-cli-http/httpclient"
}

// FlagsAndArgs adds numerous flags that can be used to configure the
// server in the context.
// The default flags list contains all of the flag actions
// in this package except for SetHandler and its variants.
func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: SetHostname()},
			{Uses: SetAddr()},
			{Uses: SetPort()},
			{Uses: SetReadTimeout()},
			{Uses: SetReadHeaderTimeout()},
			{Uses: SetWriteTimeout()},
			{Uses: SetIdleTimeout()},
			{Uses: SetMaxHeaderBytes()},
			{Uses: SetStaticDirectory()},
			{Uses: SetNoDirectoryListings()},
			{Uses: SetOpenInBrowser()},
			{Uses: SetAccessLog()},
			{Uses: SetNoAccessLog()},
			{Uses: SetServer()},
			{Uses: SetTLSCertFile()},
			{Uses: SetTLSKeyFile()},
		}...),
	)
}

func ContextValue(s *Server) cli.Action {
	return cli.ContextValue(servicesKey, s)
}

// SetHostname sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetHostname(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "host",
			Aliases:  []string{"h"},
			HelpText: "Sets the server {HOST} name to use",
			Category: listenerCategory,
		},
		withBinding((*Server).SetHostname, s),
		tagged,
	)
}

// SetPort sets the server port, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetPort(s ...int) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "port",
			Aliases:  []string{"p"},
			HelpText: "Sets the server {PORT} that will be used",
			Category: listenerCategory,
		},
		withBinding((*Server).SetPort, s),
		tagged,
	)
}

// SetAddr sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetAddr(s ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "addr",
			HelpText: "Sets the server {ADDRESS} to use",
			Category: listenerCategory,
		},
		withBinding((*Server).SetAddr, s),
		tagged,
	)
}

// SetReadTimeout sets the maximum duration for reading the entire
// request, including the body, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadTimeout(d ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-timeout",
			HelpText: "Sets the maximum {DURATION} for reading the entire request",
			Category: advancedCategory,
		},
		withBinding((*Server).SetReadTimeout, d),
		tagged,
	)
}

// SetShutdownTimeout sets the maximum duration to wait for shutting down
// the server.
func SetShutdownTimeout(d ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "shutdown-timeout",
			HelpText: "Sets the maximum {DURATION} for shutting down the server",
			Category: advancedCategory,
		},
		withBinding((*Server).SetShutdownTimeout, d),
		tagged,
	)
}

// SetReadHeaderTimeout sets the amount of time allowed to read
// request headers, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadHeaderTimeout(d ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-header-timeout",
			Value:    new(time.Duration),
			HelpText: "Sets the amount of {TIME} allowed to read request headers",
			Category: advancedCategory,
		},
		withBinding((*Server).SetReadHeaderTimeout, d),
		tagged,
	)
}

// SetWriteTimeout sets the maximum duration before timing out
// writes of the response, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetWriteTimeout(d ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "write-header-timeout",
			Value:    new(time.Duration),
			HelpText: "Sets the amount of {TIME} allowed to write response",
			Category: advancedCategory,
		},
		withBinding((*Server).SetWriteTimeout, d),
		tagged,
	)
}

// SetIdleTimeout sets the maximum amount of time to wait for the
// next request when keep-alives are enabled, which either uses the specified
// value or reads from the corresponding flag/arg to get the value to set.
// If zero is set, then the value of read time is used, unless both are zero
// in which case there is no timeout.
func SetIdleTimeout(d ...time.Duration) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "idle-timeout",
			Value:    new(time.Duration),
			HelpText: "Sets the amount of {TIME} allowed to read request headers",
			Category: advancedCategory,
		},
		withBinding((*Server).SetIdleTimeout, d),
		tagged,
	)
}

// SetMaxHeaderBytes sets the maximum header size in bytes
func SetMaxHeaderBytes(v ...int) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "max-header-bytes",
			HelpText: "Specify the maximum header bytes allowed for headers",
			Value:    new(int),
			Category: advancedCategory,
		},
		withBinding((*Server).SetMaxHeaderBytes, v),
		tagged,
	)
}

// SetStaticDirectory sets the static directory to host
func SetStaticDirectory(f ...*cli.File) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "directory",
			Aliases:  []string{"d"},
			Value:    new(cli.File),
			Options:  cli.MustExist,
			HelpText: "Serve static files from the specified directory",
			Category: serverCategory,
		},
		withBinding(bindFileAsString((*Server).SetStaticDirectory), f),
		tagged,
	)
}

// SetNoDirectoryListings causes directories not to be listed
func SetNoDirectoryListings() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "no-directory-listings",
			HelpText: "When set, don't display directory listings",
			Category: serverCategory,
		},
		withBinding((*Server).SetNoDirectoryListings, []bool{true}),
		tagged,
	)
}

// SetOpenInBrowser causes the default Web browser to open when the server
// is ready
func SetOpenInBrowser() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "open",
			HelpText: "When set, open the default Web browser when the server is ready",
			Category: serverCategory,
		},
		withBinding((*Server).setOpenInBrowserHelper, []bool{true}),
		tagged,
	)
}

func setHandlerSpec(spec HandlerSpec, v []httpclient.VirtualPath) cli.ActionFunc {
	switch len(v) {
	case 0:
		return func(c *cli.Context) error {
			vp := c.Value("").(*httpclient.VirtualPath)
			handler, err := spec(c, *vp)
			if err != nil {
				return err
			}
			return FromContext(c).Handle(vp.RequestPath, handler)
		}
	case 1:
		return func(c *cli.Context) error {
			vp := v[0]
			handler, err := spec(c, vp)
			if err != nil {
				return err
			}
			return FromContext(c).Handle(vp.RequestPath, handler)
		}
	default:
		panic(expectedOneArg)
	}
}

// SetHandler adds the specified handler to the mux. This can be called multiple
// times. SetHandler only works if a Registry named "handlers" is present
// in the context to convert the handler spec to the correct implementation.
// Consider adding [HandlerRegistry] to the Uses pipeline..
// This handler is not included in [FlagsAndArgs]
func SetHandler(v ...httpclient.VirtualPath) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "handler",
			Aliases:   []string{"H"},
			UsageText: "route:handler[,options]",
			HelpText:  "Binds a handler to the given route",
			Value:     new(httpclient.VirtualPath),
			Options:   cli.EachOccurrence,
			Category:  serverCategory,
		},
		cli.At(cli.ActionTiming, setHandlerSpec(RegistryHandlerSpec("handlers"), v)),
		tagged,
	)
}

// ListHandlers provides an action which lists the handlers for the
// handler flag. When used in the Uses pipeline, also sets reasonable defaults
// for a flag.
// This handler is not included in [FlagsAndArgs]
func ListHandlers() cli.Action {
	return cli.Pipeline(
		provider.ListProviders("handlers"),
		cli.HelpText("List available providers for the handler option then exit"),
	)
}

// Handle registers the given handler with the context server
func Handle(path string, h http.Handler) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		return FromContext(c).Handle(path, h)
	})
}

// SetFileServerHandler adds the specified file server handler to the mux.
// This can be called multiple times.
// This handler is not included in [FlagsAndArgs]
func SetFileServerHandler(v ...httpclient.VirtualPath) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "files",
			UsageText: "[route:]directory",
			HelpText:  "Binds a handler to the given route",
			Value:     new(httpclient.VirtualPath),
			Options:   cli.EachOccurrence,
		},
		cli.At(cli.ActionTiming, setHandlerSpec(FileServerHandlerSpec(), v)),
		tagged,
	)
}

func SetAccessLog(v ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "access-log",
			Aliases:  []string{"a"},
			HelpText: "Set access log format",
			Category: advancedCategory,
		},
		withBinding((*Server).SetAccessLog, v),
		tagged,
	)
}

func SetNoAccessLog(v ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "no-access-log",
			HelpText: "Disable the access log",
			Category: advancedCategory,
		},
		withBinding((*Server).SetNoAccessLog, v),
		tagged,
	)
}

func SetServer(v ...string) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "server",
			Aliases:  []string{"S"},
			HelpText: "Set value of the Server response header",
			Category: advancedCategory,
		},
		withBinding((*Server).SetServer, v),
		tagged,
	)
}

func SetTLSKeyFile(v ...*cli.File) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "key",
			HelpText: "Specify the FILE that contains the TLS private key",
			Category: listenerCategory,
			Options:  cli.MustExist,
		},
		withBinding(bindFileAsString((*Server).SetTLSKeyFile), v),
		tagged,
	)
}

func SetTLSCertFile(v ...*cli.File) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "cert",
			HelpText: "Specify the FILE that contains the TLS certificate",
			Category: listenerCategory,
			Options:  cli.MustExist,
		},
		withBinding(bindFileAsString((*Server).SetTLSCertFile), v),
		tagged,
	)
}

// RunServer locates the server in context and runs it until interrupt signal
// is detected. Optional actions run just before the server starts up, typically
// used to provide context-bound modifications to the server just in time.
func RunServer(actionopt ...cli.Action) cli.Action {
	return cli.Setup{
		Uses: cli.HandleSignal(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT),
		Action: cli.Pipeline().Append(actionopt...).Append(cli.ActionFunc(func(c *cli.Context) error {
			srv := FromContext(c)
			c.After(cli.ActionOf(func() {
				// Shutting down happens in After because the signal handler will be unregistered
				timeoutCtx, cancel := context.WithTimeout(context.Background(), srv.ShutdownTimeout)
				defer cancel()

				_ = srv.Shutdown(timeoutCtx)

				srv.actualShutdown()(timeoutCtx)
			}))
			return execContext(c, srv.ListenAndServe, srv.actualReady())
		})),
	}
}

func execContext(c context.Context, fn func() error, ready func(context.Context)) error {
	var (
		errors = make(chan error, 1)
		thunk  = func() {
			err := fn()
			if err != nil {
				errors <- err
			}
		}
	)

	go thunk()

	select {
	case <-time.After(allowStartupTime):
		go ready(c)
	case err := <-errors:
		return serverFailed(err)
	}

	<-c.Done()
	return nil
}

func serverFailed(err error) error {
	return cli.Exit(fmt.Sprintf("fatal: unable to start server: %s", err), 1)
}

func withBinding[V any](binder func(*Server, V) error, args []V) cli.Action {
	switch len(args) {
	case 0:
		return cli.BindContext(FromContext, binder)
	case 1:
		return cli.BindContext(FromContext, binder, args[0])
	default:
		panic(expectedOneArg)
	}
}

func bindFileAsString(fn func(*Server, string) error) func(*Server, *cli.File) error {
	return func(s *Server, f *cli.File) error {
		return fn(s, f.Name)
	}
}
