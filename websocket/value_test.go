// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket_test

import (
	"github.com/Carbonfrost/joe-cli-http/websocket"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("URLValue", func() {

	DescribeTable("URL examples", func(text string, expected string) {
		actual, err := websocket.NewURLValue(text).URL()

		Expect(err).NotTo(HaveOccurred())
		Expect(actual.String()).To(Equal(expected))
	},
		Entry("ws scheme", "ws://localhost:8080/graphql", "ws://localhost:8080/graphql"),
		Entry("wss scheme", "wss://example.com/graphql", "wss://example.com/graphql"),
		Entry("http scheme is converted", "http://example.com/graphql", "ws://example.com/graphql"),
		Entry("https scheme is converted", "https://example.com/graphql", "wss://example.com/graphql"),
		Entry("port implies localhost", ":8080", "ws://localhost:8080"),
		Entry("hostname implies ws", "example.com/graphql", "ws://example.com/graphql"),
	)

	It("can be reset for re-use", func() {
		value := websocket.NewURLValue("ws://localhost:8080")
		value.Reset()

		Expect(value.String()).To(BeEmpty())
	})
})

var _ = Describe("MessageType", func() {

	DescribeTable("Set examples", func(text string, expected websocket.MessageType) {
		var actual websocket.MessageType

		Expect(actual.Set(text)).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	},
		Entry("text", "text", websocket.TextMessage),
		Entry("binary", "binary", websocket.BinaryMessage),
		Entry("case insensitive", "BINARY", websocket.BinaryMessage),
	)

	It("reports unknown message types", func() {
		var actual websocket.MessageType

		Expect(actual.Set("unknown")).To(MatchError(`unknown message type "unknown"`))
	})

	DescribeTable("String examples", func(m websocket.MessageType, expected string) {
		Expect(m.String()).To(Equal(expected))
	},
		Entry("text", websocket.TextMessage, "text"),
		Entry("binary", websocket.BinaryMessage, "binary"),
	)
})
