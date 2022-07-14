package httpclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

type JSONContent struct {
	data interface{}
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
		c.data = map[string]interface{}{
			name: value,
		}
	case map[string]interface{}:
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
