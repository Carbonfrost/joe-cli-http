// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket_test

import (
	"context"
	"time"

	"github.com/Carbonfrost/joe-cli-http/websocket"
	ws "github.com/gorilla/websocket"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Client", func() {

	Describe("NewDialer", func() {

		DescribeTable("examples", func(opts []websocket.Option, expected Fields) {
			client := websocket.New()
			client.Apply(opts...)

			dialer, err := client.NewDialer(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(dialer).To(PointTo(MatchFields(IgnoreExtras, expected)))
		},
			Entry("subprotocols", []websocket.Option{
				websocket.WithSubprotocol("graphql-transport-ws"),
				websocket.WithSubprotocol("wamp"),
			}, Fields{
				"Subprotocols": Equal([]string{"graphql-transport-ws", "wamp"}),
			}),
			Entry("handshake timeout", []websocket.Option{
				websocket.WithHandshakeTimeout(2 * time.Second),
			}, Fields{
				"HandshakeTimeout": Equal(2 * time.Second),
			}),
			Entry("buffer sizes", []websocket.Option{
				websocket.WithReadBufferSize(512),
				websocket.WithWriteBufferSize(1024),
			}, Fields{
				"ReadBufferSize":  Equal(512),
				"WriteBufferSize": Equal(1024),
			}),
			Entry("compression", []websocket.Option{
				websocket.WithCompression(true),
			}, Fields{
				"EnableCompression": BeTrue(),
			}),
		)

		It("caches the dialer", func() {
			client := websocket.New()

			first, err := client.NewDialer(context.Background())
			Expect(err).NotTo(HaveOccurred())

			second, err := client.NewDialer(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(second).To(BeIdenticalTo(first))
		})

		It("applies the connection settings to a discrete dialer", func() {
			expected := &ws.Dialer{ReadBufferSize: 4096}
			client := websocket.New(
				websocket.WithDialer(expected),
				websocket.WithSubprotocol("wamp"),
			)

			actual, err := client.NewDialer(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(expected))
			Expect(actual.Subprotocols).To(Equal([]string{"wamp"}))
			Expect(actual.ReadBufferSize).To(Equal(4096))
		})
	})

	Describe("Location", func() {

		It("reports when no URL was specified", func() {
			_, err := websocket.New().Location()
			Expect(err).To(MatchError(websocket.ErrNoLocation))
		})

		It("obtains the URL that was specified", func() {
			client := websocket.New(websocket.WithURL(":8080"))

			loc, err := client.Location()
			Expect(err).NotTo(HaveOccurred())

			u, err := loc.URL()
			Expect(err).NotTo(HaveOccurred())
			Expect(u.String()).To(Equal("ws://localhost:8080"))
		})
	})

	Describe("Dial", func() {

		It("reports when no URL was specified", func() {
			_, err := websocket.New().Dial(context.Background())
			Expect(err).To(MatchError(websocket.ErrNoLocation))
		})
	})
})

var _ = Describe("Options", func() {

	It("applies each of the parts", func() {
		client := websocket.New()
		options := &websocket.Options{
			URL:          new("wss://example.com/graphql"),
			Headers:      map[string]string{"X-Trace": "on"},
			Subprotocols: []string{"wamp"},
			ReadLimit:    new(int64(2048)),
		}
		client.Apply(options)

		loc, err := client.Location()
		Expect(err).NotTo(HaveOccurred())
		Expect(loc.String()).To(Equal("wss://example.com/graphql"))

		dialer, err := client.NewDialer(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(dialer.Subprotocols).To(Equal([]string{"wamp"}))
	})
})
