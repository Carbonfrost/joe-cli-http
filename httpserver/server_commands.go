package httpserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Carbonfrost/joe-cli"
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
		}...),
	)
}

// SetHostname sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetHostname(s ...string) cli.Action {
	switch len(s) {
	case 0:
		return cli.Prototype{
			Name:     "host",
			Aliases:  []string{"h"},
			HelpText: "Sets the server {HOST} name to use",
			Category: listenerCategory,
			Setup: cli.Setup{
				Uses: cli.BindContext(FromContext, (*Server).SetHostname),
			},
		}
	case 1:
		return cli.BindContext(FromContext, (*Server).SetHostname, s[0])
	default:
		panic(expectedOneArg)
	}
}

// SetPort sets the server port, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetPort(s ...int) cli.Action {
	switch len(s) {
	case 0:
		return cli.Prototype{
			Name:     "port",
			Aliases:  []string{"p"},
			HelpText: "Sets the server {PORT} that will be used",
			Category: listenerCategory,

			Setup: cli.Setup{
				Uses: cli.BindContext(FromContext, (*Server).SetPort),
			},
		}
	case 1:
		return cli.BindContext(FromContext, (*Server).SetPort, s[0])
	default:
		panic(expectedOneArg)
	}
}

// SetAddr sets the server address, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetAddr(s ...string) cli.Action {
	switch len(s) {
	case 0:
		return cli.Prototype{
			Name:     "addr",
			HelpText: "Sets the server {ADDRESS} to use",
			Category: listenerCategory,
			Setup: cli.Setup{
				Uses: cli.BindContext(FromContext, (*Server).SetAddr),
			},
		}
	case 1:
		return cli.BindContext(FromContext, (*Server).SetAddr, s[0])
	default:
		panic(expectedOneArg)
	}
}

// SetReadTimeout sets the maximum duration for reading the entire
// request, including the body, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadTimeout(d ...time.Duration) cli.Action {
	switch len(d) {
	case 0:
		return cli.Prototype{
			Name:     "read-timeout",
			HelpText: "Sets the maximum {DURATION} for reading the entire request",
			Category: advancedCategory,
			Setup: cli.Setup{
				Uses: cli.BindContext(FromContext, (*Server).SetReadTimeout),
			},
		}
	case 1:
		return cli.BindContext(FromContext, (*Server).SetReadTimeout, d[0])
	default:
		panic(expectedOneArg)
	}
}

// SetReadHeaderTimeout sets the amount of time allowed to read
// request headers, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadHeaderTimeout(d ...time.Duration) cli.Action {
	return createFlag((*Server).SetReadHeaderTimeout, d, &cli.Prototype{
		Name:     "read-header-timeout",
		Value:    new(time.Duration),
		HelpText: "Sets the amount of {TIME} allowed to read request headers",
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
	})
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

func createFlag[T any](binder func(*Server, T) error, args []T, proto *cli.Prototype) cli.Action {
	switch len(args) {
	case 0:
		proto.Setup = cli.Setup{
			Uses: cli.BindContext(FromContext, binder),
		}
		return proto
	case 1:
		return cli.BindContext(FromContext, binder, args[0])
	default:
		panic(expectedOneArg)
	}
}
