// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpserver

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type fileInfo struct {
	Filename    string `json:"filename,omitempty"`
	Sniff       string `json:"sniff"`        // Detected MIME type
	SHA1        string `json:"sha1"`         // SHA-1 hex digest
	First32Base string `json:"first_32_b64"` // First 32 bytes, base64
}

type reflectedRequest struct {
	Method  string                `json:"method"`
	URL     string                `json:"url"`
	Remote  string                `json:"remote_ip"`
	Headers map[string][]string   `json:"headers"`
	Query   map[string][]string   `json:"query"`
	Cookies map[string]string     `json:"cookies,omitempty"`
	JSON    any                   `json:"json,omitempty"`
	Form    map[string][]string   `json:"form,omitempty"`
	Files   map[string][]fileInfo `json:"files,omitempty"`
	Errors  []string              `json:"errors,omitempty"`
	Body    fileInfo              `json:"body,omitzero"`
}

type echoHandler struct {
	failsafe bool
}

// NewEchoHandler provides a handler which echoes the request
// back in the response. An option failsafe indicates whether the response
// will always be 200 OK even if the request is malformed.
func NewEchoHandler(failsafe bool) http.Handler {
	return &echoHandler{
		failsafe: failsafe,
	}
}

func getRemoteIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (e *echoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := reflectedRequest{
		Method:  r.Method,
		URL:     r.URL.String(),
		Remote:  getRemoteIP(r),
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	handleError := func(e string) {
		w.WriteHeader(http.StatusBadRequest)
		resp.Errors = append(resp.Errors, e)
	}

	if e.failsafe {
		handleError = func(e string) {
			resp.Errors = append(resp.Errors, e)
		}
	}

	// Parse cookies
	if len(r.Cookies()) > 0 {
		result := make(map[string]string)
		for _, c := range r.Cookies() {
			result[c.Name] = c.Raw
		}
		resp.Cookies = result
	}

	ct := r.Header.Get("Content-Type")

	// Parse form data if applicable
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil && err != http.ErrNotMultipart {
			handleError(fmt.Sprintf("form parse error: %v", err))
		} else {
			resp.Form = r.Form
		}
	} else {
		// Interpret as JSON if the header is set or otherwise implicitly
		body, err := io.ReadAll(r.Body)
		if err == nil {
			defer r.Body.Close()
			if strings.HasPrefix(ct, "application/json") || json.Valid(body) {
				var decoded any
				if err := json.Unmarshal(body, &decoded); err != nil {
					handleError(fmt.Sprintf("invalid JSON in body: %v", err))
				} else {
					resp.JSON = decoded
				}

			}
			if len(body) > 0 {
				resp.Body, _ = sniffAndSum("", bytes.NewReader(body))
			}
		} else {
			handleError(fmt.Sprintf("read body error: %v", err))
		}

	}

	// Handle file uploads
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		resp.Files = make(map[string][]fileInfo)
		for field, fhs := range r.MultipartForm.File {
			for _, fh := range fhs {
				f, err := fh.Open()
				if err != nil {
					handleError(fmt.Sprintf("file open error (%s): %v", fh.Filename, err))
					continue
				}
				defer f.Close()

				fi, err := sniffAndSum(fh.Filename, f)
				if err != nil {
					handleError(fmt.Sprintf("file read error (%s): %v", fh.Filename, err))
				}

				resp.Files[field] = append(resp.Files[field], fi)
			}
		}
	}

	// If errors have occurred, only write out the errors unless we are in failsafe mode.
	var output any = resp

	if len(resp.Errors) > 0 && !e.failsafe {
		output = resp.Errors
	}
	if err := json.NewEncoder(w).Encode(output); err != nil {
		log.Printf("echo handler: error writing response: %v", err)
	}
}

func sniffAndSum(filename string, f io.Reader) (fi fileInfo, err error) {
	buf := make([]byte, 32)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return
	}

	sniff := http.DetectContentType(buf[:n])

	sha := sha1.New()
	sha.Write(buf[:n])
	_, err = io.Copy(sha, f)
	if err != nil {
		return
	}

	sum := sha.Sum(nil)
	return fileInfo{
		Filename:    filename,
		Sniff:       sniff,
		SHA1:        fmt.Sprintf("%x", sum),
		First32Base: base64.StdEncoding.EncodeToString(buf[:n]),
	}, nil
}
