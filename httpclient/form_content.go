// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"io"
	"net/url"
)

type FormDataContent struct {
	bufferedContent
}

type MultipartFormDataContent struct {
	bufferedContent
}

type URLEncodedFormDataContent struct {
	bufferedContent
}

func (c *FormDataContent) Query() (url.Values, error) {
	panic("not impl")
}

func (c *FormDataContent) Set(name, value string) error {
	panic("not impl")
}

func (c *FormDataContent) SetFile(name, file io.Reader) error {
	panic("not impl")
}

func (c *FormDataContent) ContentType() string {
	return ""
}

func (c *MultipartFormDataContent) Query() (url.Values, error) {
	panic("not impl")
}

func (c *MultipartFormDataContent) Set(name, value string) error {
	panic("not impl")
}

func (c *MultipartFormDataContent) SetFile(name, file io.Reader) error {
	panic("not impl")
}

func (c *MultipartFormDataContent) ContentType() string {
	return ""
}

func (c *URLEncodedFormDataContent) Query() (url.Values, error) {
	panic("not impl")
}

func (c *URLEncodedFormDataContent) Set(name, value string) error {
	panic("not impl")
}

func (c *URLEncodedFormDataContent) SetFile(name, file io.Reader) error {
	panic("not impl")
}

func (c *URLEncodedFormDataContent) ContentType() string {
	return ""
}
