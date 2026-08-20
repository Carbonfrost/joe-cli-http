// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

// Conn represents a WebSocket connection which has been established by the
// client.  It wraps the underlying websocket.Conn so that the timeouts and
// message type which were configured on the client are applied to each
// operation.  The embedded connection can be used directly when more control
// is required.
type Conn struct {
	*ws.Conn

	response     *http.Response
	messageType  MessageType
	readTimeout  time.Duration
	writeTimeout time.Duration
	closeTimeout time.Duration
}

// Response obtains the HTTP response from the handshake
func (c *Conn) Response() *http.Response {
	return c.response
}

// Send writes the message using the message type which was configured on the
// client
func (c *Conn) Send(data []byte) error {
	if c.writeTimeout > 0 {
		if err := c.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return err
		}
	}
	return c.WriteMessage(int(c.messageType), data)
}

// Receive reads the next message from the connection
func (c *Conn) Receive() (MessageType, []byte, error) {
	if c.readTimeout > 0 {
		if err := c.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return 0, nil, err
		}
	}
	messageType, data, err := c.ReadMessage()
	return MessageType(messageType), data, err
}

// CloseNormal performs the closing handshake, indicating normal closure
func (c *Conn) CloseNormal() error {
	deadline := time.Now().Add(c.closeTimeout)
	message := ws.FormatCloseMessage(ws.CloseNormalClosure, "")
	return c.WriteControl(ws.CloseMessage, message, deadline)
}
