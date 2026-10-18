// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type Response struct {
	*http.Response

	FailFast       bool
	IncludeHeaders bool
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

func (r *Response) Download(ctx context.Context, downloader Downloader) error {
	if r.FailFast && !r.Success() {
		return fmt.Errorf("request failed (%s): %s %s", r.Status, r.Request.Method, r.Request.URL)
	}

	output, err := downloader.OpenDownload(ctx, r)
	if err != nil {
		return err
	}

	if r.IncludeHeaders {
		err = r.CopyHeadersTo(output)
		fmt.Fprintln(output)
	}
	if err != nil {
		return err
	}

	err = r.CopyTo(output)
	if err != nil {
		return err
	}

	err = output.Close()
	if err != nil {
		return err
	}

	return nil
}
