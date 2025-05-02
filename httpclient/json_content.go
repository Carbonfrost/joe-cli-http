// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

type JSONContent struct {
	data any
}

func (c *JSONContent) Query() (url.Values, error) {
	panic("not impl")
}

func (c *JSONContent) Set(name, value string) error {
	if strings.Contains(name, ".") {
		panic("not impl")
	}
	switch data := c.data.(type) {
	case nil:
		c.data = map[string]any{
			name: value,
		}
	case map[string]any:
		data[name] = value
	default:
		panic("not impl")
	}
	return nil
}

func (c *JSONContent) ContentType() string {
	return "application/json"
}

func (c *JSONContent) SetFile(name, file io.Reader) error {
	panic("not impl")
}

func (c *JSONContent) Read() io.Reader {
	buf, _ := json.MarshalIndent(c.data, "", "    ")
	return bytes.NewBuffer(buf)
}
