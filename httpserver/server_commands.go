package httpserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/internal/cliutil"
)

const (
	listenerCategory = "Listener options"
	advancedCategory = "Advanced options"

	allowStartupTime  = 1 * time.Second
	allowShutdownTime = 3 * time.Second
)

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: SetHostname()},
			{Uses: SetAddr()},
			{Uses: SetPort()},
			{Uses: SetReadTimeout()},
			{Uses: SetReadHeaderTimeout()},
			{Uses: SetWriteTimeout()},
			{Uses: SetStaticDirectory()},
			{Uses: SetNoDirectoryListings()},
		}...),
	)
}

// SetHostname sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetHostname(s ...string) cli.Action {
	return createFlag((*Server).SetHostname, s, &cli.Prototype{
		Name:     "host",
		Aliases:  []string{"h"},
		HelpText: "Sets the server {HOST} name to use",
		Category: listenerCategory,
	})
}

// SetPort sets the server port, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetPort(s ...int) cli.Action {
	return createFlag((*Server).SetPort, s, &cli.Prototype{
		Name:     "port",
		Aliases:  []string{"p"},
		HelpText: "Sets the server {PORT} that will be used",
		Category: listenerCategory,
	})
}

// SetAddr sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetAddr(s ...string) cli.Action {
	return createFlag((*Server).SetAddr, s, &cli.Prototype{
		Name:     "addr",
		HelpText: "Sets the server {ADDRESS} to use",
		Category: listenerCategory,
	})
}

// SetReadTimeout sets the maximum duration for reading the entire
// request, including the body, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadTimeout(d ...time.Duration) cli.Action {
	return createFlag((*Server).SetReadTimeout, d, &cli.Prototype{
		Name:     "read-timeout",
		HelpText: "Sets the maximum {DURATION} for reading the entire request",
		Category: advancedCategory,
	})
}

// SetReadHeaderTimeout sets the amount of time allowed to read
// request headers, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadHeaderTimeout(d ...time.Duration) cli.Action {
	return createFlag((*Server).SetReadHeaderTimeout, d, &cli.Prototype{
		Name:     "read-header-timeout",
		Value:    new(time.Duration),
		HelpText: "Sets the amount of {TIME} allowed to read request headers",
		Category: advancedCategory,
	})
}

// SetWriteTimeout sets the maximum duration before timing out
// writes of the response, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetWriteTimeout(d ...time.Duration) cli.Action {
	return createFlag((*Server).SetWriteTimeout, d, &cli.Prototype{
		Name:     "read-header-timeout",
		Value:    new(time.Duration),
		HelpText: "Sets the amount of {TIME} allowed to read request headers",
		Category: advancedCategory,
	})
}

// SetIdleTimeout sets the maximum amount of time to wait for the
// next request when keep-alives are enabled, which either uses the specified
// value or reads from the corresponding flag/arg to get the value to set.
// If zero is set, then the value of read time is used, unless both are zero
// in which case there is no timeout.
func SetIdleTimeout(d ...time.Duration) cli.Action {
	return createFlag((*Server).SetIdleTimeout, d, &cli.Prototype{
		Name:     "idle-timeout",
		Value:    new(time.Duration),
		HelpText: "Sets the amount of {TIME} allowed to read request headers",
		Category: advancedCategory,
	})
}

// SetStaticDirectory sets the static directory to host
func SetStaticDirectory(f ...*cli.File) cli.Action {
	return createFlag((*Server).setStaticDirectoryHelper, f, &cli.Prototype{
		Name:     "directory",
		Aliases:  []string{"d"},
		Value:    new(cli.File),
		Options:  cli.MustExist,
		HelpText: "Serve static files from the specified directory",
	})
}

// SetNoDirectoryListings causes directories not to be listed
func SetNoDirectoryListings() cli.Action {
	return &cli.Prototype{
		Name:     "no-directory-listings",
		HelpText: "When set, don't display directory listings",
		Setup:    dualSetup(cli.BindContext(FromContext, (*Server).SetNoDirectoryListings)),
	}
}

// RunServer locates the server in context and runs it until interrupt signal
// is detected
func RunServer() cli.Action {
	return cli.Setup{
		Uses: cli.HandleSignal(os.Interrupt),
		Action: func(c *cli.Context) error {
			srv := FromContext(c)
			c.After(func() {
				// Shutting down happens in After because the signal handler will be unregistered
				timeoutCtx, cancel := context.WithTimeout(context.Background(), allowShutdownTime)
				defer cancel()

				_ = srv.Shutdown(timeoutCtx)
			})
			return execContext(c, srv.ListenAndServe, srv.actualReady())
		},
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

func createFlag[V any](binder func(*Server, V) error, args []V, proto *cli.Prototype) cli.Action {
	return cliutil.FlagBinding(FromContext, binder, args, proto)
}

func dualSetup(a cli.Action) cli.Setup {
	return cliutil.DualSetup(a)
}
