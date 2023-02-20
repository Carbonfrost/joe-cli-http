package httpclient

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
)

// Content implements the logic to build the body content of a request
// or to produce the query string if else.
type Content interface {
	Read() io.Reader
	Query() (url.Values, error)
	ContentType() string
	Set(name, value string) error
	SetFile(name, file io.Reader) error
}

type bufferedContent struct {
	buf bytes.Buffer
}

type RawContent struct {
	bufferedContent
}

var (
	rawContentSetError = errors.New("structured form data is not supported for raw content")
)

func NewContent(ct ContentType) Content {
	switch ct {
	case ContentTypeRaw:
		return &RawContent{}
	case ContentTypeFormData:
		return &FormDataContent{}
	case ContentTypeJSON:
		return &JSONContent{}
	case ContentTypeMultipartFormData:
		return &MultipartFormDataContent{}
	case ContentTypeURLEncodedFormData:
		return &URLEncodedFormDataContent{}
	default:
		panic(fmt.Errorf("unknown content type: %v", ct))
	}
}

func NewRawContent(data []byte) *RawContent {
	res := new(RawContent)
	res.Write(data)
	return res
}

func convertContent(from Content, to ContentType) (Content, error) {
	if to == ContentTypeRaw {
		if raw, ok := from.(*RawContent); ok {
			return raw, nil
		}
		body, err := io.ReadAll(from.Read())
		return NewRawContent(body), err
	}
	return nil, fmt.Errorf("conversion not supported %T -> %v", from, to)
}

func (c *bufferedContent) Read() io.Reader {
	return bytes.NewReader(c.buf.Bytes())
}

func (c *RawContent) Write(d []byte) (int, error) {
	return c.bufferedContent.buf.Write(d)
}

func (c *RawContent) Query() (url.Values, error) {
	return nil, rawContentSetError
}

func (c *RawContent) Set(name, value string) error {
	return rawContentSetError
}

func (c *RawContent) SetFile(name, file io.Reader) error {
	return rawContentSetError
}

func (c *RawContent) ContentType() string {
	return ""
}
