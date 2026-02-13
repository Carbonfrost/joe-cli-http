// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tls

import (
	"context"
	gotls "crypto/tls"
	"crypto/x509"
	"io/fs"
	"os"

	cli "github.com/Carbonfrost/joe-cli"
)

type contextKey string

type Config struct {
	*gotls.Config
	cli.Action
}

type Option func(*Config)

const servicesKey contextKey = "httpserver_services"

func New(opts ...Option) *Config {
	c := &Config{
		Config: new(gotls.Config),
	}
	c.Apply(opts...)
	c.Action = defaultAction(c)
	return c
}

func ContextValue(c *Config) cli.Action {
	return cli.ContextValue(servicesKey, c)
}

// FromContext obtains the server from the context.
func FromContext(ctx context.Context) *Config {
	return ctx.Value(servicesKey).(*Config)
}

func defaultAction(c *Config) cli.Action {
	return cli.Pipeline(
		ContextValue(c),
		FlagsAndArgs(),
	)
}

func (c *Config) Apply(opts ...Option) {
	for _, o := range opts {
		o(c)
	}
}

func (c *Config) ensureCACertPool() *x509.CertPool {
	if c.RootCAs == nil {
		c.RootCAs = x509.NewCertPool()
	}
	return c.RootCAs
}

func WithInsecureSkipVerify(v bool) Option {
	return func(c *Config) {
		c.InsecureSkipVerify = v
	}
}

func WithCiphers(ids *CipherSuites) Option {
	return func(c *Config) {
		c.CipherSuites = []uint16(*ids)
	}
}

func WithCurves(ids *CurveIDs) Option {
	return func(c *Config) {
		c.CurvePreferences = []gotls.CurveID(*ids)
	}
}

func WithServerName(s string) Option {
	return func(c *Config) {
		c.ServerName = s
	}
}

func WithTimeHelper(f *cli.File) Option {
	return func(c *Config) {
		s, err := f.Stat()
		// TODO Improve error handling
		if err != nil {
			panic(err)
		}
		c.Time = s.ModTime
	}
}

func WithNextProtos(s []string) Option {
	return func(c *Config) {
		c.NextProtos = s
	}
}

func WithCertificate(cert gotls.Certificate) Option {
	return func(cfg *Config) {
		cfg.Certificates = append(cfg.Certificates, cert)
	}
}

func WithRootCACertPath(path string) Option {
	return func(c *Config) {
		paths, err := fs.Glob(os.DirFS("."), "*.pem")

		// TODO Improve error handling
		if err != nil {
			panic(err)
		}
		for _, p := range paths {
			WithRootCACertFile(p)(c)
		}
	}
}

func WithRootCACertFile(filename string) Option {
	return func(cfg *Config) {
		// TODO Should read from FS fs.ReadFile
		cert, err := os.ReadFile(filename)
		// TODO Improve error handling
		if err != nil {
			panic(err)
		}

		caCertPool := cfg.ensureCACertPool()
		_ = caCertPool.AppendCertsFromPEM(cert)
	}
}

func WithX509KeyPair(certFile, keyFile string) Option {
	cert, err := gotls.LoadX509KeyPair(certFile, keyFile)

	// TODO Improve error handling
	if err != nil {
		panic(err)
	}
	return WithCertificate(cert)
}

func (o Option) Execute(c context.Context) error {
	o(FromContext(c))
	return nil
}
