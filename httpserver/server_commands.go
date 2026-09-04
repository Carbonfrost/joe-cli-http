// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver

import (
	"context"
	"fmt"
	"reflect"
	"syscall"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

const (
	listenerCategory = "Listener options"
	advancedCategory = "Advanced options"
	serverCategory   = "Server options"

	allowStartupTime = 1 * time.Second
)

var (
	tagged  = cli.Data(SourceAnnotation())
	pkgPath = reflect.TypeFor[Server]().PkgPath()
)

// Action provides a context action that affects the server
type Action = cli.Action

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// FlagsAndArgs adds numerous flags that can be used to configure the
// server in the context.
// The default flags list contains all of the flag actions
// in this package except for SetHandler and its variants.
func FlagsAndArgs() Action {
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
			{Uses: SetHideDirectoryListings()},
			{Uses: SetOpenInBrowser()},
			{Uses: SetAccessLog()},
			{Uses: SetServerHeader()},
			{Uses: SetTLSCertFile()},
			{Uses: SetTLSKeyFile()},
		}...),
	)
}

func ContextValue(s *Server) Action {
	return cli.WithContextValue(servicesKey, s)
}

// SetHostname sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetHostname(s ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "host",
			Uses:     cli.OptionalAlias("h"),
			HelpText: "Sets the server {HOST} name to use",
			Category: listenerCategory,
		},
		bind.Action(WithHostname, bind.Exact(s...)),
		tagged,
	)
}

// SetPort sets the server port, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetPort(s ...int) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "port",
			Uses:     cli.OptionalAlias("p"),
			HelpText: "Sets the server {PORT} that will be used",
			Category: listenerCategory,
		},
		bind.Action(WithPort, bind.Exact(s...)),
		tagged,
	)
}

// SetAddr sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetAddr(s ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "addr",
			HelpText: "Sets the server {ADDRESS} to use",
			Category: listenerCategory,
		},
		bind.Action(WithAddr, bind.Exact(s...)),
		tagged,
	)
}

// SetReadTimeout sets the maximum duration for reading the entire
// request, including the body, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-timeout",
			HelpText: "Sets the maximum {DURATION} for reading the entire request",
			Category: advancedCategory,
		},
		bind.Action(WithReadTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetShutdownTimeout sets the maximum duration to wait for shutting down
// the server.
func SetShutdownTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "shutdown-timeout",
			HelpText: "Sets the maximum {DURATION} for shutting down the server",
			Category: advancedCategory,
		},
		bind.Action(WithShutdownTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetReadHeaderTimeout sets the amount of time allowed to read
// request headers, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadHeaderTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-header-timeout",
			Value:    new(time.Duration),
			HelpText: "Sets the amount of {TIME} allowed to read request headers",
			Category: advancedCategory,
		},
		bind.Action(WithReadHeaderTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetWriteTimeout sets the maximum duration before timing out
// writes of the response, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetWriteTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "write-header-timeout",
			Value:    new(time.Duration),
			HelpText: "Sets the amount of {TIME} allowed to write response",
			Category: advancedCategory,
		},
		bind.Action(WithWriteTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetIdleTimeout sets the maximum amount of time to wait for the
// next request when keep-alives are enabled, which either uses the specified
// value or reads from the corresponding flag/arg to get the value to set.
// If zero is set, then the value of read time is used, unless both are zero
// in which case there is no timeout.
func SetIdleTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "idle-timeout",
			Value:    new(time.Duration),
			HelpText: "Sets the amount of {TIME} allowed to read request headers",
			Category: advancedCategory,
		},
		bind.Action(WithIdleTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetMaxHeaderBytes sets the maximum header size in bytes
func SetMaxHeaderBytes(v ...int) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "max-header-bytes",
			HelpText: "Specify the maximum header bytes allowed for headers",
			Value:    new(int),
			Category: advancedCategory,
		},
		bind.Action(WithMaxHeaderBytes, bind.Exact(v...)),
		tagged,
	)
}

// SetStaticDirectory sets the static directory to host
func SetStaticDirectory(f ...*cli.File) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "directory",
			Uses:     cli.OptionalAlias("d"),
			Value:    new(cli.File),
			Options:  cli.MustExist,
			HelpText: "Serve static files from the specified directory",
			Category: serverCategory,
		},
		bind.Action(WithStaticDirectory, bind.Exact(f...).(*bind.FileBinder).Name()),
		tagged,
	)
}

// SetHideDirectoryListings causes directories not to be listed
func SetHideDirectoryListings() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "hide-directory-listings",
			HelpText: "When set, don't display directory listings",
			Category: serverCategory,
		},
		cli.At(cli.ActionTiming, WithHideDirectoryListings(true)),
		tagged,
	)
}

// SetOpenInBrowser causes the default Web browser to open when the server
// is ready
func SetOpenInBrowser() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "open",
			HelpText: "When set, open the default Web browser when the server is ready",
			Category: serverCategory,
			Value:    new(bool),
		},
		cli.At(cli.ActionTiming, AddReadyFunc(OpenInBrowser())),
		tagged,
	)
}

// SetHandler adds the specified handler to the mux. This can be called multiple
// times. SetHandler only works if a Registry named "handlers" is present
// in the context to convert the handler spec to the correct implementation.
// Consider adding [HandlerRegistry] to the Uses pipeline..
// This handler is not included in [FlagsAndArgs]
func SetHandler(v ...httpclient.VirtualPath) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "handler",
			Uses:      cli.OptionalAlias("H"),
			UsageText: "route:handler[,options]",
			HelpText:  "Binds a handler to the given route",
			Value:     new(httpclient.VirtualPath),
			Options:   cli.EachOccurrence,
			Category:  serverCategory,
		},
		bind.Action2(HandleSpec, bind.Exact(v...), bind.Exact(RegistryHandlerSpec("handlers"))),
		tagged,
	)
}

// ListHandlers provides an action which lists the handlers for the
// handler flag. When used in the Uses pipeline, also sets reasonable defaults
// for a flag.
// This handler is not included in [FlagsAndArgs]
func ListHandlers() Action {
	return cli.Pipeline(
		provider.ListProviders("handlers"),
		cli.HelpText("List available providers for the handler option then exit"),
	)
}

// HandleSpec registers the given handler spec with the context server
func HandleSpec(vpath httpclient.VirtualPath, spec HandlerSpec) Action {
	return cli.ActionOf(func(c context.Context) error {
		handler, err := spec(c, vpath)
		if err != nil {
			return err
		}
		return FromContext(c).Handle(vpath.RequestPath, handler)
	})
}

// SetFileServerHandler adds the specified file server handler to the mux.
// This can be called multiple times.
// This handler is not included in [FlagsAndArgs]
func SetFileServerHandler(v ...httpclient.VirtualPath) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "files",
			UsageText: "[route:]directory",
			HelpText:  "Binds a handler to the given route",
			Value:     new(httpclient.VirtualPath),
			Options:   cli.EachOccurrence,
		},
		bind.Action2(HandleSpec, bind.Exact(v...), bind.Exact(FileServerHandlerSpec())),
		tagged,
	)
}

func SetAccessLog(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "access-log",
			Uses:     cli.OptionalAlias("a"),
			HelpText: "Set access log format",
			Category: advancedCategory,
			Options:  cli.No,
		},
		bind.Action(WithAccessLog, bind.Exact(v...)),
		tagged,
	)
}

func SetServerHeader(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "server",
			Uses:     cli.OptionalAlias("S"),
			HelpText: "Set value of the Server response header",
			Category: advancedCategory,
		},
		bind.Action(WithServerHeader, bind.Exact(v...)),
		tagged,
	)
}

func SetTLSKeyFile(v ...*cli.File) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "key",
			HelpText: "Specify the FILE that contains the TLS private key",
			Category: listenerCategory,
			Options:  cli.MustExist,
			Uses:     cli.Requires("cert"),
		},
		bind.Action(WithTLSKeyFile, bind.Exact(v...).(*bind.FileBinder).Name()),
		tagged,
	)
}

func SetTLSCertFile(v ...*cli.File) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "cert",
			HelpText: "Specify the FILE that contains the TLS certificate",
			Category: listenerCategory,
			Options:  cli.MustExist,
			Uses:     cli.Requires("key"),
		},
		bind.Action(WithTLSCertFile, bind.Exact(v...).(*bind.FileBinder).Name()),
		tagged,
	)
}

// RunServer locates the server in context and runs it until interrupt signal
// is detected. Optional actions run just before the server starts up, typically
// used to provide context-bound modifications to the server just in time.
func RunServer(actionopt ...cli.Action) Action {
	return cli.Setup{
		Uses: cli.HandleSignal(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT),
		Action: cli.Pipeline(cli.ActionOf(actionopt), cli.ActionFunc(func(c *cli.Context) error {
			srv := FromContext(c)
			c.After(cli.ActionOf(func() {
				// Shutting down happens in After because the signal handler will be unregistered
				timeoutCtx, cancel := context.WithTimeout(context.Background(), srv.ShutdownTimeout())
				defer cancel()

				_ = srv.Shutdown(timeoutCtx)

				srv.actualShutdown()(timeoutCtx)
			}))
			return execContext(c, func() error {
				return srv.ListenAndServe(c)
			}, srv.actualReady())
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
