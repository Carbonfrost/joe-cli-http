// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	ws "github.com/gorilla/websocket"
)

// URLValue provides ergonomics for entering WebSocket URLs as values.  When the
// text looks like a port (e.g. :8080), the URL is interpreted as localhost.
// When the text looks like a hostname, the prefix ws:// is prepended.  The HTTP
// schemes are converted to their WebSocket counterparts, so that http becomes
// ws and https becomes wss.
type URLValue struct {
	loc string
}

// MessageType identifies the type of a WebSocket message
type MessageType int

const (
	// TextMessage denotes a text data message, whose payload is interpreted as
	// UTF-8 encoded text
	TextMessage = MessageType(ws.TextMessage)

	// BinaryMessage denotes a binary data message
	BinaryMessage = MessageType(ws.BinaryMessage)
)

var wsSchemes = []string{"ws:", "wss:"}

// NewURLValue creates a new URLValue from a string
func NewURLValue(loc string) *URLValue {
	return &URLValue{loc}
}

// URL interprets the value as a URL
func (u *URLValue) URL() (*url.URL, error) {
	return url.Parse(fixupAddress(u.loc))
}

// Set updates the value from the text of the flag or arg
func (u *URLValue) Set(arg string) error {
	u.loc = arg
	return nil
}

// String obtains the text of the value
func (u *URLValue) String() string {
	return u.loc
}

// Reset clears the value, which facilitates its re-use
func (u *URLValue) Reset() {
	// To facilitate re-use
	u.loc = ""
}

// Copy duplicates the value
func (u *URLValue) Copy() *URLValue {
	res := *u
	return &res
}

// Synopsis obtains the placeholder text used in help screens
func (MessageType) Synopsis() string {
	return "TYPE"
}

// String obtains the name of the message type
func (m MessageType) String() string {
	switch m {
	case TextMessage:
		return "text"
	case BinaryMessage:
		return "binary"
	}
	return ""
}

// MarshalText provides the textual representation of the message type
func (m MessageType) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText parses the textual representation of the message type
func (m *MessageType) UnmarshalText(b []byte) error {
	return m.Set(string(b))
}

// Set updates the value from the text of the flag or arg
func (m *MessageType) Set(arg string) error {
	switch strings.ToLower(arg) {
	case "":
		return nil
	case "text":
		*m = TextMessage
	case "binary":
		*m = BinaryMessage
	default:
		return fmt.Errorf("unknown message type %q", arg)
	}
	return nil
}

// fixupAddress applies the ergonomics described by URLValue
func fixupAddress(addr string) string {
	if addr == "" || strings.HasPrefix(addr, "/") {
		return addr
	}
	if strings.HasPrefix(addr, ":") {
		return "ws://localhost" + addr
	}
	if rest, ok := strings.CutPrefix(addr, "http:"); ok {
		return "ws:" + rest
	}
	if rest, ok := strings.CutPrefix(addr, "https:"); ok {
		return "wss:" + rest
	}
	for _, scheme := range wsSchemes {
		if strings.HasPrefix(addr, scheme) {
			return addr
		}
	}
	return "ws://" + addr
}

var (
	_ flag.Value = (*URLValue)(nil)
	_ flag.Value = (*MessageType)(nil)
)
