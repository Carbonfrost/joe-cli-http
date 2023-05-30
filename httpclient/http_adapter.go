package httpclient

import (
	"io"
	"net/http"
)

type Response struct {
	*http.Response
}

func (r *Response) Success() bool {
	return r.Response.StatusCode < 400
}

func (r *Response) CopyTo(w io.Writer) error {
	body := r.Response.Body
	defer body.Close()

	_, err := io.Copy(w, body)
	return err
}

func (r *Response) CopyHeadersTo(w io.Writer) error {
	return r.Response.Header.Write(w)
}
