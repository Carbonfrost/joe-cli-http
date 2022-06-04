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

func (c *MultipartFormDataContent) Query() (url.Values, error) {
	panic("not impl")
}

func (c *MultipartFormDataContent) Set(name, value string) error {
	panic("not impl")
}

func (c *MultipartFormDataContent) SetFile(name, file io.Reader) error {
	panic("not impl")
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
