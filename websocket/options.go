// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"context"
	"maps"
	"slices"
	"time"
)

// Options contains settings for the client which have data representations.
// Each non-nil field is applied when Options is used as an Option.
type Options struct {
	URL              *string           `toml:"url"                json:"url,omitempty"`
	Headers          map[string]string `toml:"headers"            json:"headers,omitempty"`
	Origin           *string           `toml:"origin"             json:"origin,omitempty"`
	Subprotocols     []string          `toml:"subprotocols"       json:"subprotocols,omitempty"`
	Messages         []string          `toml:"messages"           json:"messages,omitempty"`
	MessageType      *MessageType      `toml:"message-type"       json:"messageType,omitempty"`
	HandshakeTimeout *time.Duration    `toml:"handshake-timeout"  json:"handshakeTimeout,omitempty"`
	ReadTimeout      *time.Duration    `toml:"read-timeout"       json:"readTimeout,omitempty"`
	WriteTimeout     *time.Duration    `toml:"write-timeout"      json:"writeTimeout,omitempty"`
	CloseTimeout     *time.Duration    `toml:"close-timeout"      json:"closeTimeout,omitempty"`
	ReadBufferSize   *int              `toml:"read-buffer-size"   json:"readBufferSize,omitempty"`
	WriteBufferSize  *int              `toml:"write-buffer-size"  json:"writeBufferSize,omitempty"`
	ReadLimit        *int64            `toml:"read-limit"         json:"readLimit,omitempty"`
	Compression      *bool             `toml:"compression"        json:"compression,omitempty"`
	Verbose          *bool             `toml:"verbose"            json:"verbose,omitempty"`
}

// Execute applies the options to the client in the context
func (o *Options) Execute(ctx context.Context) error {
	o.apply(FromContext(ctx))
	return nil
}

func (o *Options) apply(c *Client) {
	c.Apply(o.parts()...)
}

func (o *Options) parts() (results []Option) {
	if o.URL != nil {
		results = append(results, WithURL(*o.URL))
	}

	// Headers are applied in sorted order so that the result is deterministic
	for _, name := range slices.Sorted(maps.Keys(o.Headers)) {
		results = append(results, WithHeader(name, o.Headers[name]))
	}
	if o.Origin != nil {
		results = append(results, WithOrigin(*o.Origin))
	}
	for _, name := range o.Subprotocols {
		results = append(results, WithSubprotocol(name))
	}
	for _, message := range o.Messages {
		results = append(results, WithMessage(message))
	}
	if o.MessageType != nil {
		results = append(results, WithMessageType(*o.MessageType))
	}
	if o.HandshakeTimeout != nil {
		results = append(results, WithHandshakeTimeout(*o.HandshakeTimeout))
	}
	if o.ReadTimeout != nil {
		results = append(results, WithReadTimeout(*o.ReadTimeout))
	}
	if o.WriteTimeout != nil {
		results = append(results, WithWriteTimeout(*o.WriteTimeout))
	}
	if o.CloseTimeout != nil {
		results = append(results, WithCloseTimeout(*o.CloseTimeout))
	}
	if o.ReadBufferSize != nil {
		results = append(results, WithReadBufferSize(*o.ReadBufferSize))
	}
	if o.WriteBufferSize != nil {
		results = append(results, WithWriteBufferSize(*o.WriteBufferSize))
	}
	if o.ReadLimit != nil {
		results = append(results, WithReadLimit(*o.ReadLimit))
	}
	if o.Compression != nil {
		results = append(results, WithCompression(*o.Compression))
	}
	if o.Verbose != nil {
		results = append(results, WithVerbose(*o.Verbose))
	}
	return
}
