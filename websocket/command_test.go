// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/websocket"
	ws "github.com/gorilla/websocket"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConnectAndPrint", func() {

	var (
		server *httptest.Server

		// mu guards received, which is appended to from the handler goroutine
		mu       sync.Mutex
		received []string
	)

	messages := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}

	BeforeEach(func() {
		received = nil
		upgrader := &ws.Upgrader{}
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				mu.Lock()
				received = append(received, string(msg))
				mu.Unlock()
				if err := conn.WriteMessage(ws.TextMessage, []byte("echo "+string(msg))); err != nil {
					return
				}
			}
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	It("sends each message from the input and prints the responses", func() {
		var out bytes.Buffer

		app := &cli.App{
			Uses:   websocket.New(),
			Action: websocket.ConnectAndPrint(),
			Stdin:  strings.NewReader("hello\nworld\n"),
			Stdout: &out,
			Stderr: io.Discard,
		}

		args, _ := cli.Split("_ " + server.URL + " --read-timeout 150ms")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(messages()).To(Equal([]string{"hello", "world"}))
		Expect(out.String()).To(Equal("echo hello\necho world\n"))
	})

	It("sends the messages specified by the message flag", func() {
		var out bytes.Buffer

		app := &cli.App{
			Uses:   websocket.New(),
			Action: websocket.ConnectAndPrint(),
			Stdin:  strings.NewReader(""),
			Stdout: &out,
			Stderr: io.Discard,
		}

		args, _ := cli.Split("_ " + server.URL + " -m ping -m pong --read-timeout 150ms")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(messages()).To(Equal([]string{"ping", "pong"}))
	})

	It("echoes the messages which are sent when verbose", func() {
		var errOut bytes.Buffer

		app := &cli.App{
			Uses:   websocket.New(),
			Action: websocket.ConnectAndPrint(),
			Stdin:  strings.NewReader(""),
			Stdout: io.Discard,
			Stderr: &errOut,
		}

		args, _ := cli.Split("_ " + server.URL + " -m ping --verbose --read-timeout 150ms")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(errOut.String()).To(ContainSubstring("→ ping"))
	})

	It("reports the error when the server can't be reached", func() {
		app := &cli.App{
			Uses:   websocket.New(),
			Action: websocket.ConnectAndPrint(),
			Stdin:  strings.NewReader(""),
			Stdout: io.Discard,
			Stderr: io.Discard,
		}

		args, _ := cli.Split("_ ws://localhost:1/nope --handshake-timeout 1s")
		err := app.RunContext(context.Background(), args)

		Expect(err).To(MatchError(ContainSubstring("connect ws://localhost:1/nope")))
	})
})

var _ = Describe("Set actions", func() {

	It("has Source annotation", func() {
		app := &cli.App{
			Uses: websocket.New(),
		}
		app.Initialize(context.Background())

		f, _ := app.Flag("protocol")
		Expect(f.Data).To(HaveKeyWithValue("Source", "github.com/Carbonfrost/joe-cli-http/websocket"))
	})
})
