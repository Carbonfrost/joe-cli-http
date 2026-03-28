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

// Config reepresents TLS configuration plus a default action.
type Config struct {
	*gotls.Config
	cli.Action
}

// Option provides an option to the TLS configuration
type Option func(*Config) error

const servicesKey contextKey = "httpserver_services"

// New creates a new TLS configuration. By default, it is also initialized
// with a default action that registers useful flags.
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

// WithInsecureSkipVerify sets up the option to skip verification in TLS
// handshake. This option is insecure.
func WithInsecureSkipVerify(v bool) Option {
	return func(c *Config) error {
		c.InsecureSkipVerify = v
		return nil
	}
}

// WithCiphers sets up cipher suites
func WithCiphers(ids *CipherSuites) Option {
	return func(c *Config) error {
		c.CipherSuites = []uint16(*ids)
		return nil
	}
}

// WithCiphers sets up curves
func WithCurves(ids *CurveIDs) Option {
	return func(c *Config) error {
		c.CurvePreferences = []gotls.CurveID(*ids)
		return nil
	}
}

// WithServerName sets the server name
func WithServerName(s string) Option {
	return func(c *Config) error {
		c.ServerName = s
		return nil
	}
}

// WithTimeHelper specifies the file whose modtime contains the time
// for the server
func WithTimeHelper(f *cli.File) Option {
	return func(c *Config) error {
		s, err := f.Stat()
		if err != nil {
			return err
		}
		c.Time = s.ModTime
		return nil
	}
}

// WithNextProtos sets next protos
func WithNextProtos(s []string) Option {
	return func(c *Config) error {
		c.NextProtos = s
		return nil
	}
}

// AddCertificate adds a certificate to the pool
func AddCertificate(cert gotls.Certificate) Option {
	return func(cfg *Config) error {
		cfg.Certificates = append(cfg.Certificates, cert)
		return nil
	}
}

// AddRootCACertPath adds a search path to the CA certificate pool
func AddRootCACertPath(path string) Option {
	return func(c *Config) error {
		paths, err := fs.Glob(os.DirFS("."), "*.pem")

		if err != nil {
			return err
		}
		for _, p := range paths {
			AddRootCACertFile(p)(c)
		}
		return nil
	}
}

// WithRootCACertFile adds a CA certificate to the pool
func AddRootCACertFile(filename string) Option {
	return func(cfg *Config) error {
		cert, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		caCertPool := cfg.ensureCACertPool()
		_ = caCertPool.AppendCertsFromPEM(cert)
		return nil
	}
}

// AddX509KeyPair adds a certificate to the pool
func AddX509KeyPair(certFile, keyFile string) Option {
	return func(cfg *Config) error {
		cert, err := gotls.LoadX509KeyPair(certFile, keyFile)

		if err != nil {
			return err
		}
		return AddCertificate(cert)(cfg)
	}
}

func (o Option) Execute(c context.Context) error {
	o(FromContext(c))
	return nil
}
