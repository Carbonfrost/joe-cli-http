// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialer_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/net/dialer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

type otherKey struct{}

var _ = Describe("Set actions", func() {

	It("has Source annotation", func() {
		app := &cli.App{
			Uses: dialer.New(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("--dial-timeout")
		Expect(f.Data).To(HaveKeyWithValue("Source", "github.com/Carbonfrost/joe-cli-http/net/dialer"))
	})

	It("registers flags with Id annotation", func() {
		app := &cli.App{
			Uses: dialer.New(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("--bind-address")
		id, ok := dialer.LookupID(f)
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal(dialer.IDBindAddress))
	})

	DescribeTable("examples", func(act cli.Action, command string, expected Fields) {
		d := dialer.New()
		// Override default action so no flags are registered; only place in the context
		d.Action = dialer.ContextValue(d)
		app := &cli.App{
			Uses:   d,
			Action: func() {},
			Stdout: io.Discard,
			Flags: []*cli.Flag{
				{
					Name: "a",
					Uses: act,
				},
			},
		}
		args, _ := cli.Split(command)

		err := app.RunContext(context.Background(), args...)
		Expect(err).NotTo(HaveOccurred())
		Expect(d.Dialer).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry(
			"SetDialTimeout",
			dialer.SetDialTimeout(),
			"app -a 5s",
			Fields{"Timeout": Equal(5 * time.Second)}),

		Entry(
			"SetDialKeepAlive",
			dialer.SetDialKeepAlive(),
			"app -a 7s",
			Fields{"KeepAlive": Equal(7 * time.Second)}),

		Entry(
			"SetDisableDialKeepAlive",
			dialer.SetDisableDialKeepAlive(),
			"app -a",
			Fields{"KeepAlive": Equal(time.Duration(-1))}),

		Entry(
			"SetBindAddress",
			dialer.SetBindAddress(),
			"app -a 127.0.0.1:0",
			Fields{"LocalAddr": Equal(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})}),

		Entry(
			"SetInterface by address",
			dialer.SetInterface(),
			"app -a 127.0.0.1",
			Fields{"LocalAddr": PointTo(MatchFields(IgnoreExtras, Fields{
				"IP": Equal(net.ParseIP("127.0.0.1")),
			}))}),
	)

	It("reports errors from options", func() {
		d := dialer.New()
		d.Action = dialer.ContextValue(d)
		app := &cli.App{
			Uses:   d,
			Action: func() {},
			Flags:  []*cli.Flag{{Name: "a", Uses: dialer.SetInterface()}},
		}
		err := app.RunContext(context.Background(), "app", "-a", "no-such-interface-xyz")
		Expect(err).To(HaveOccurred())
	})

	It("lists interfaces", func() {
		var out bytes.Buffer
		d := dialer.New()
		app := &cli.App{
			Uses:   d,
			Stdout: &out,
			Action: func() {},
		}
		// The flag exits, which is reported as an error
		_ = app.RunContext(context.Background(), "app", "--list-interfaces")
		Expect(out.String()).NotTo(BeEmpty())
	})
})

var _ = Describe("Context key", func() {

	It("defaults when not specified", func() {
		d := dialer.New()
		var found *dialer.Dialer
		app := &cli.App{
			Uses:   d,
			Action: func(c *cli.Context) { found = dialer.FromContext(c) },
		}
		Expect(app.RunContext(context.Background(), "app")).To(Succeed())
		Expect(found).To(BeIdenticalTo(d))
	})

	It("stores the dialer under a custom key", func() {
		d := dialer.New(dialer.WithContextKey(otherKey{}))
		var byKey, byDefault *dialer.Dialer
		app := &cli.App{
			Uses: d,
			Action: func(c *cli.Context) {
				byKey = dialer.FromContextKey(c, otherKey{})
				byDefault = dialer.FromContext(c)
			},
		}
		Expect(app.RunContext(context.Background(), "app")).To(Succeed())
		Expect(byKey).To(BeIdenticalTo(d))
		Expect(byDefault).To(BeNil())
		Expect(d.ContextKey()).To(Equal(otherKey{}))
	})

	It("annotates the default flags with the custom key", func() {
		// The key is applied even though it comes after the default action
		d := dialer.New(dialer.WithContextKey(otherKey{}))
		app := &cli.App{Uses: d}
		app.Initialize(context.Background())

		f, _ := app.Flag("--dial-timeout")
		key, ok := dialer.LookupContextKey(f)
		Expect(ok).To(BeTrue())
		Expect(key).To(Equal(otherKey{}))
	})

	It("configures the dialer matching the flag annotation", func() {
		origin := dialer.New(dialer.WithAction(nil))
		proxy := dialer.New(dialer.WithAction(nil), dialer.WithContextKey(otherKey{}))
		app := &cli.App{
			Uses: cli.Pipeline(
				dialer.ContextValue(origin),
				dialer.ContextValue(proxy),
			),
			Action: func() {},
			Flags: []*cli.Flag{
				{
					Name: "origin-timeout",
					Uses: dialer.SetDialTimeout(),
				},
				{
					Name: "proxy-timeout",
					Uses: cli.Pipeline(
						dialer.SetDialTimeout(),
						dialer.ContextKeyAnnotation(otherKey{}),
					),
				},
			},
		}

		err := app.RunContext(context.Background(), "app", "--origin-timeout", "1s", "--proxy-timeout", "2s")
		Expect(err).NotTo(HaveOccurred())
		Expect(origin.Timeout).To(Equal(1 * time.Second))
		Expect(proxy.Timeout).To(Equal(2 * time.Second))
	})

	It("can be annotated on an enclosing command", func() {
		proxy := dialer.New(dialer.WithAction(nil), dialer.WithContextKey(otherKey{}))
		app := &cli.App{
			Uses: cli.Pipeline(
				dialer.ContextValue(proxy),
				dialer.ContextKeyAnnotation(otherKey{}),
			),
			Action: func() {},
			Flags:  []*cli.Flag{{Name: "a", Uses: dialer.SetDialTimeout()}},
		}

		err := app.RunContext(context.Background(), "app", "-a", "3s")
		Expect(err).NotTo(HaveOccurred())
		Expect(proxy.Timeout).To(Equal(3 * time.Second))
	})

	It("annotates flags from FlagsAndArgs with the key", func() {
		app := &cli.App{
			Uses: dialer.FlagsAndArgs(otherKey{}),
		}
		app.Initialize(context.Background())

		for _, name := range []string{"dial-timeout", "interface", "list-interfaces"} {
			f, _ := app.Flag("--" + name)
			key, ok := dialer.LookupContextKey(f)
			Expect(ok).To(BeTrue(), name)
			Expect(key).To(Equal(otherKey{}))
		}
	})

	It("has no annotation without a key", func() {
		app := &cli.App{
			Uses: dialer.FlagsAndArgs(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("--dial-timeout")
		_, ok := dialer.LookupContextKey(f)
		Expect(ok).To(BeFalse())
	})

	It("fails when the dialer is missing from the context", func() {
		app := &cli.App{
			Action: func() {},
			Flags:  []*cli.Flag{{Name: "a", Uses: dialer.SetDialTimeout()}},
		}
		err := app.RunContext(context.Background(), "app", "-a", "1s")
		Expect(err).To(MatchError(ContainSubstring("no dialer")))
	})
})
