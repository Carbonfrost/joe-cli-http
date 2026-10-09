// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialer_test

import (
	"context"
	"errors"
	"net"

	"github.com/Carbonfrost/joe-cli-http/net/dialer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dialer", func() {

	var listener net.Listener

	BeforeEach(func() {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		listener = l
		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				c.Close()
			}
		}()
		DeferCleanup(l.Close)
	})

	Describe("DialContext", func() {
		It("connects", func() {
			conn, err := dialer.New().DialContext(context.Background(), "tcp", listener.Addr().String())
			Expect(err).NotTo(HaveOccurred())
			conn.Close()
		})

		It("uses the resolver", func() {
			r := &net.Resolver{PreferGo: true}
			d := dialer.New(dialer.WithResolver(r))

			conn, err := d.Dial("tcp", listener.Addr().String())
			Expect(err).NotTo(HaveOccurred())
			conn.Close()
			Expect(d.Dialer.Resolver).To(BeIdenticalTo(r))
		})

		It("reports errors from the resolver factory", func() {
			expected := errors.New("factory failed")
			d := dialer.New(dialer.WithResolverFactory(func(context.Context) (*net.Resolver, error) {
				return nil, expected
			}))

			_, err := d.Dial("tcp", listener.Addr().String())
			Expect(err).To(MatchError(expected))
		})

		It("reports errors from options passed to New", func() {
			d := dialer.New(dialer.WithInterface("no-such-interface-xyz"))

			_, err := d.Dial("tcp", listener.Addr().String())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("NewResolver", func() {
		It("creates a default resolver", func() {
			r, err := dialer.New().NewResolver(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(r).NotTo(BeNil())
		})

		It("caches the resolver", func() {
			count := 0
			d := dialer.New(dialer.WithResolverFactory(func(context.Context) (*net.Resolver, error) {
				count++
				return new(net.Resolver), nil
			}))

			r1, _ := d.NewResolver(context.Background())
			r2, _ := d.NewResolver(context.Background())
			Expect(r1).To(BeIdenticalTo(r2))
			Expect(count).To(Equal(1))
		})

		It("prefers the discrete value over the factory", func() {
			r := new(net.Resolver)
			d := dialer.New(
				dialer.WithResolverFactory(func(context.Context) (*net.Resolver, error) {
					return nil, errors.New("should not be called")
				}),
				dialer.WithResolver(r),
			)

			actual, err := d.NewResolver(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(r))
		})

		It("applies middleware", func() {
			d := dialer.New(dialer.WithResolverMiddleware(
				func(_ context.Context, r *net.Resolver) *net.Resolver {
					r.PreferGo = true
					return r
				},
			))

			r, err := d.NewResolver(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(r.PreferGo).To(BeTrue())
		})
	})

	Describe("ID", func() {
		It("round-trips text", func() {
			var id dialer.ID
			Expect(id.UnmarshalText([]byte(dialer.IDInterface.String()))).To(Succeed())
			Expect(id).To(Equal(dialer.IDInterface))
			Expect(id.UnmarshalText([]byte("nope"))).NotTo(Succeed())
		})
	})
})
