// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver

import (
	"context"
	"time"
)

// Options contains settings for the server which have data representations.
// Each non-nil field is applied when Options is used as an Option.
type Options struct {
	Addr                  *string        `toml:"addr"                    json:"addr,omitempty"`
	Hostname              *string        `toml:"hostname"                json:"hostname,omitempty"`
	Port                  *int           `toml:"port"                    json:"port,omitempty"`
	TLSCertFile           *string        `toml:"tls-cert-file"           json:"tlsCertFile,omitempty"`
	TLSKeyFile            *string        `toml:"tls-key-file"            json:"tlsKeyFile,omitempty"`
	ServerHeader          *string        `toml:"server-header"           json:"serverHeader,omitempty"`
	AccessLog             *string        `toml:"access-log"              json:"accessLog,omitempty"`
	StaticDirectory       *string        `toml:"static-directory"        json:"staticDirectory,omitempty"`
	HideDirectoryListings *bool          `toml:"hide-directory-listings" json:"hideDirectoryListings,omitempty"`
	ShutdownTimeout       *time.Duration `toml:"shutdown-timeout"        json:"shutdownTimeout,omitempty"`
	ReadTimeout           *time.Duration `toml:"read-timeout"            json:"readTimeout,omitempty"`
	ReadHeaderTimeout     *time.Duration `toml:"read-header-timeout"     json:"readHeaderTimeout,omitempty"`
	WriteTimeout          *time.Duration `toml:"write-timeout"           json:"writeTimeout,omitempty"`
	IdleTimeout           *time.Duration `toml:"idle-timeout"            json:"idleTimeout,omitempty"`
	MaxHeaderBytes        *int           `toml:"max-header-bytes"        json:"maxHeaderBytes,omitempty"`
}

func (o *Options) Execute(ctx context.Context) error {
	o.apply(FromContext(ctx))
	return nil
}

func (o *Options) apply(s *Server) {
	s.Apply(o.parts()...)
}

func (o *Options) parts() (results []Option) {
	if o.Hostname != nil {
		results = append(results, WithHostname(*o.Hostname))
	}
	if o.Port != nil {
		results = append(results, WithPort(*o.Port))
	}
	if o.Addr != nil {
		// Addr must come after port and hostname
		results = append(results, WithAddr(*o.Addr))
	}
	if o.TLSCertFile != nil {
		results = append(results, WithTLSCertFile(*o.TLSCertFile))
	}
	if o.TLSKeyFile != nil {
		results = append(results, WithTLSKeyFile(*o.TLSKeyFile))
	}
	if o.ServerHeader != nil {
		results = append(results, WithServerHeader(*o.ServerHeader))
	}
	if o.AccessLog != nil {
		results = append(results, WithAccessLog(*o.AccessLog))
	}
	if o.StaticDirectory != nil {
		results = append(results, WithStaticDirectory(*o.StaticDirectory))
	}
	if o.HideDirectoryListings != nil {
		results = append(results, WithHideDirectoryListings(*o.HideDirectoryListings))
	}
	if o.ShutdownTimeout != nil {
		results = append(results, WithShutdownTimeout(*o.ShutdownTimeout))
	}
	if o.ReadTimeout != nil {
		results = append(results, WithReadTimeout(*o.ReadTimeout))
	}
	if o.ReadHeaderTimeout != nil {
		results = append(results, WithReadHeaderTimeout(*o.ReadHeaderTimeout))
	}
	if o.WriteTimeout != nil {
		results = append(results, WithWriteTimeout(*o.WriteTimeout))
	}
	if o.IdleTimeout != nil {
		results = append(results, WithIdleTimeout(*o.IdleTimeout))
	}
	if o.MaxHeaderBytes != nil {
		results = append(results, WithMaxHeaderBytes(*o.MaxHeaderBytes))
	}
	return
}
