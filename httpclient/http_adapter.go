package httpclient

import (
	"fmt"
	"io"
	"net/http"
)

type Response struct {
	*http.Response

	IncludeHeaders bool
}

func (r *Response) CopyTo(w io.Writer) error {
	body := r.Response.Body
	defer body.Close()

	if r.IncludeHeaders {
		r.printHeaders()
	}

	_, err := io.Copy(w, body)
	return err
}

func (r *Response) printHeaders() {
	for k, values := range r.Response.Header {
		fmt.Print(k, ": ")
		for i, val := range values {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Print(val)
		}
		fmt.Println()
	}
}
