package httpclient

import (
	"flag"
	"fmt"
)

type ContentType int

const (
	// ContentTypeFormData is the content type associated with either URL-encoded or multipart form data
	// depending upon whether files are set
	ContentTypeFormData ContentType = iota

	// ContentTypeRaw is the content type associated with raw content
	ContentTypeRaw ContentType = iota

	// ContentTypeURLEncodedFormData is URL encoding form data body content
	ContentTypeURLEncodedFormData

	// ContentTypeMultipartFormData is multi-part form data body content
	ContentTypeMultipartFormData

	// ContentTypeJSON is JSON data body content
	ContentTypeJSON

	maxContentType
)

var (
	contentTypeStrings = [maxContentType]string{
		"FORM_DATA",
		"RAW",
		"URL_ENCODED_FORM_DATA",
		"MULTIPART_FORM_DATA",
		"JSON",
	}
)

func (ContentType) Synopsis() string {
	return "TYPE"
}

func (c ContentType) String() string {
	return contentTypeStrings[c]
}

func (c ContentType) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *ContentType) UnmarshalText(b []byte) error {
	str := string(b)
	for i, s := range contentTypeStrings {
		if s == str {
			*c = ContentType(i)
			return nil
		}
	}
	return nil
}

func (c *ContentType) Set(arg string) error {
	switch arg {
	case "form":
		*c = ContentTypeFormData
		return nil
	case "raw":
		*c = ContentTypeRaw
		return nil
	case "urlencoded":
		*c = ContentTypeURLEncodedFormData
		return nil
	case "multipart":
		*c = ContentTypeMultipartFormData
		return nil
	case "json":
		*c = ContentTypeJSON
		return nil
	}
	return fmt.Errorf("unknown content type %q", arg)
}

var _ flag.Value = (*ContentType)(nil)
