// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"reflect"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

const (
	handshakeOptions = "Handshake options"
	messageOptions   = "Message options"
	advancedOptions  = "Advanced options"
)

var (
	tagged  = cli.Data(SourceAnnotation())
	pkgPath = reflect.TypeFor[Client]().PkgPath()
)

// Action is an alias for the action within the Joe framework
type Action = cli.Action

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// ContextValue provides an action which stores the client in the context
func ContextValue(c *Client) Action {
	return cli.WithContextValue(servicesKey, c)
}

// ConnectAndPrint provides an action which connects to the WebSocket server,
// sends each of the messages which have been configured, and prints each
// message which is received.
func ConnectAndPrint() Action {
	return cli.ActionOf(Do)
}

// FlagsAndArgs adds the flags and args that can be used to configure the
// client in the context.
func FlagsAndArgs() Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: SetHeader()},
			{Uses: SetSubprotocol()},
			{Uses: SetOrigin()},
			{Uses: SetHandshakeTimeout()},

			// Message options
			{Uses: SetMessage()},
			{Uses: SetInput()},
			{Uses: SetMessageType()},
			{Uses: SetBinary()},
			{Uses: SetReadTimeout()},
			{Uses: SetWriteTimeout()},
			{Uses: SetCloseTimeout()},

			// Advanced options
			{Uses: SetReadBufferSize()},
			{Uses: SetWriteBufferSize()},
			{Uses: SetReadLimit()},
			{Uses: SetCompression()},

			{Uses: SetVerbose()},
		}...),

		cli.AddArg(&cli.Arg{
			Uses: SetURLValue(),
		}),
	)
}

// SetURLValue sets the location that the client connects to, which either uses
// the specified value or reads from the corresponding flag/arg to get the value
// to set.
func SetURLValue(u ...*URLValue) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "url",
			HelpText: "Set the {URL} of the WebSocket server",
			Value:    new(URLValue),
			Category: handshakeOptions,
		},
		bind.Action(withURLValue, bind.Exact(u...)),
		tagged,
	)
}

// SetHeader adds a header to the handshake request, which either uses the
// specified value or reads from the corresponding flag/arg to get the value
// to set.
func SetHeader(v ...*httpclient.HeaderValue) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "header",
			Uses:     cli.OptionalAlias("H"),
			HelpText: "Sets header to {NAME} and {VALUE}",
			Value:    new(httpclient.HeaderValue),
			Options:  cli.EachOccurrence,
			Category: handshakeOptions,
		},
		bind.Action(withHeaderValue, bind.Exact(v...)),
		tagged,
	)
}

// SetSubprotocol adds a subprotocol which is requested during the handshake,
// which either uses the specified value or reads from the corresponding
// flag/arg to get the value to set.
func SetSubprotocol(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "protocol",
			Uses:     cli.OptionalAlias("P"),
			HelpText: "Requests the subprotocol {NAME} in the handshake.  Can be used multiple times",
			Options:  cli.EachOccurrence,
			Category: handshakeOptions,
		},
		bind.Action(WithSubprotocol, bind.Exact(v...)),
		tagged,
	)
}

// SetOrigin sets the Origin header used in the handshake request, which either
// uses the specified value or reads from the corresponding flag/arg to get the
// value to set.
func SetOrigin(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "origin",
			HelpText: "Sets the Origin header to {URL}",
			Category: handshakeOptions,
		},
		bind.Action(WithOrigin, bind.Exact(v...)),
		tagged,
	)
}

// SetHandshakeTimeout sets the maximum duration to allow the handshake to
// complete, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetHandshakeTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "handshake-timeout",
			HelpText: "Sets the maximum {DURATION} to complete the handshake",
			Category: handshakeOptions,
		},
		bind.Action(WithHandshakeTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetMessage adds a message which is sent once the connection is established,
// which either uses the specified value or reads from the corresponding
// flag/arg to get the value to set.
func SetMessage(v ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "message",
			Uses:     cli.OptionalAlias("m"),
			HelpText: "Sends the {MESSAGE} once connected.  Can be used multiple times",
			Options:  cli.EachOccurrence | cli.AllowFileReference,
			Category: messageOptions,
		},
		bind.Action(WithMessage, bind.Exact(v...)),
		tagged,
	)
}

// SetInput sets the files which provide the messages to send, one message per
// line, which either uses the specified value or reads from the corresponding
// flag/arg to get the value to set.
func SetInput(v ...*cli.FileSet) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "file",
			Uses:     cli.OptionalAlias("f"),
			HelpText: "Sends a message for each line read from {FILE}",
			Value:    new(cli.FileSet),
			Options:  cli.MustExist,
			Category: messageOptions,
		},
		bind.Action(WithInput, bind.Exact(v...)),
		tagged,
	)
}

// SetMessageType sets the type of the messages which are sent, which either
// uses the specified value or reads from the corresponding flag/arg to get the
// value to set.
func SetMessageType(v ...MessageType) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:       "message-type",
			HelpText:   "Sets the type of messages that are sent: text, binary",
			Value:      new(MessageType),
			Category:   messageOptions,
			Completion: cli.ValueCompletion("text", "binary"),
		},
		bind.Action(WithMessageType, bind.Exact(v...)),
		tagged,
	)
}

// SetBinary causes messages to be sent as binary messages
func SetBinary() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "binary",
			HelpText: "Sends messages as binary rather than text",
			Value:    new(bool),
			Category: messageOptions,
		},
		cli.At(cli.ActionTiming, WithMessageType(BinaryMessage)),
		tagged,
	)
}

// SetReadTimeout sets the maximum duration to wait for each message which is
// received, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetReadTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-timeout",
			HelpText: "Sets the maximum {DURATION} to wait for a message to be received",
			Category: messageOptions,
		},
		bind.Action(WithReadTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetWriteTimeout sets the maximum duration to allow for writing each message,
// which either uses the specified value or reads from the corresponding
// flag/arg to get the value to set.
func SetWriteTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "write-timeout",
			HelpText: "Sets the maximum {DURATION} to write a message",
			Category: messageOptions,
		},
		bind.Action(WithWriteTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetCloseTimeout sets the maximum duration to allow for the closing
// handshake, which either uses the specified value or reads from the
// corresponding flag/arg to get the value to set.
func SetCloseTimeout(d ...time.Duration) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "close-timeout",
			HelpText: "Sets the maximum {DURATION} to complete the closing handshake",
			Category: advancedOptions,
		},
		bind.Action(WithCloseTimeout, bind.Exact(d...)),
		tagged,
	)
}

// SetReadBufferSize sets the size of the read buffer in bytes
func SetReadBufferSize(v ...int) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-buffer-size",
			HelpText: "Specify the size in bytes of the read buffer",
			Value:    new(int),
			Category: advancedOptions,
		},
		bind.Action(WithReadBufferSize, bind.Exact(v...)),
		tagged,
	)
}

// SetWriteBufferSize sets the size of the write buffer in bytes
func SetWriteBufferSize(v ...int) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "write-buffer-size",
			HelpText: "Specify the size in bytes of the write buffer",
			Value:    new(int),
			Category: advancedOptions,
		},
		bind.Action(WithWriteBufferSize, bind.Exact(v...)),
		tagged,
	)
}

// SetReadLimit sets the maximum size in bytes of the messages which can be
// received
func SetReadLimit(v ...int64) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "read-limit",
			HelpText: "Specify the maximum size in bytes of a message that can be received",
			Value:    new(int64),
			Category: advancedOptions,
		},
		bind.Action(WithReadLimit, bind.Exact(v...)),
		tagged,
	)
}

// SetCompression enables negotiation of per-message compression
func SetCompression(v ...bool) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "compression",
			HelpText: "Negotiate per-message compression",
			Options:  cli.No,
			Category: advancedOptions,
		},
		bind.Action(WithCompression, bind.Exact(v...)),
		tagged,
	)
}

// SetVerbose causes the messages which are sent to be echoed to stderr
func SetVerbose() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "verbose",
			Uses:     cli.OptionalAlias("v"),
			Value:    new(bool),
			HelpText: "Display the messages which are sent",
		},
		bind.Action(WithVerbose, bind.Seen()),
		tagged,
	)
}
