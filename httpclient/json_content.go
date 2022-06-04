package httpclient

import (
	"io"
	"net/url"
)

type JSONContent struct {
	bufferedContent
}

func (c *JSONContent) Query() (url.Values, error) {
	panic("not impl")
}

func (c *JSONContent) Set(name, value string) error {
	panic("not impl")
}

func (c *JSONContent) SetFile(name, file io.Reader) error {
	panic("not impl")
}
