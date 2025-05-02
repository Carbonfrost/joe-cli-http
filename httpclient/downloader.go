// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
)

// Downloader provides the behavior for downloading a response
type Downloader interface {
	// OpenDownload saves the download from the response.  This method can be
	// called multiple times if multiple URLs were requested.
	OpenDownload(context.Context, *Response) (io.WriteCloser, error)
}

// DownloadMode enumerates common download methods. It implements
// [Downloader]
type DownloadMode int

type downloaderWithFileName interface {
	Downloader
	FileName(*Response) string
}

type exprAdapter struct {
	FS fs.FS

	index int
	expr  Expr
}

type basicDownloader struct {
	w io.Writer
}

type nopWriteCloser struct {
	io.Writer
}

type stripComponents struct {
	count int
	mode  DownloadMode
}

// Download modes.  PreserveRequestFile uses the remote file name.
// PreserveRequestPath uses the remote file path.
const (
	PreserveRequestFile DownloadMode = iota
	PreserveRequestPath
)

// WithStripComponents returns a Downloader which strips the specified
// number of leading path elements from the resulting file name.  This
// only pertains to PreserveRequestPath.
func (d DownloadMode) WithStripComponents(count int) Downloader {
	if d == PreserveRequestPath {
		return stripComponents{count: count, mode: d}
	}
	return d
}

func (s stripComponents) FileName(r *Response) string {
	count := s.count
	res := s.mode.FileName(r)
	if count == 0 {
		return res
	}

	dir, base := filepath.Split(res)
	dirs := strings.Split(dir, string(filepath.Separator))
	switch {
	case count > len(dirs):
		dirs = nil
	case count < -len(dirs):
		// No change to dirs
	case count < 0:
		dirs = dirs[(len(dirs)+count)%len(dirs):]
	default:
		dirs = dirs[count:]
	}
	dirs = append(dirs, base)

	return filepath.Join(dirs...)
}

func (s stripComponents) OpenDownload(ctx context.Context, resp *Response) (io.WriteCloser, error) {
	return openFileName(s, fileSystemFrom(ctx, nil), resp)
}

func openFileName(d downloaderWithFileName, f cli.FS, resp *Response) (io.WriteCloser, error) {
	fn := d.FileName(resp)
	if fn == "" {
		return nil, fmt.Errorf("cannot download file: the request path has no file name")
	}

	dir := filepath.Dir(fn)
	if _, err := f.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		f.MkdirAll(dir, 0755)
	}

	c, err := f.Create(fn)
	if err != nil {
		return nil, err
	}
	return c.(io.WriteCloser), err
}

// NewDownloaderTo implements a basic downloader that copies to
// a writer
func NewDownloaderTo(w io.Writer) Downloader {
	return basicDownloader{w}
}

// NewFileDownloader implements a downloader that copies to
// a file system.  The f argument specifies the name of the file
// to copy to.  It may contain "write out" variables that are
// expanded.  By default, a suffix is added for successive downloads
// of the same file name.
func NewFileDownloader(f string, fileSystem fs.FS) Downloader {
	if !strings.Contains(f, "%(") {
		f += "%(index.suffix)"
	}
	return &exprAdapter{
		index: -1,
		expr:  Expr(f),
		FS:    fileSystem,
	}
}

func (e *exprAdapter) OpenDownload(ctx context.Context, resp *Response) (io.WriteCloser, error) {
	e.index++
	return openFileName(e, fileSystemFrom(ctx, e.FS), resp)
}

func (e *exprAdapter) FileName(r *Response) string {
	return e.expr.Compile().Expand(expr.ComposeExpanders(e.expandIndex, ExpandResponse(r)))
}

func (e *exprAdapter) expandIndex(k string) any {
	if k == "index" {
		return e.index
	}
	if k == "index.suffix" {
		if e.index == 0 {
			return ""
		}
		return fmt.Sprintf(".%d", e.index)
	}
	return nil
}

func (d DownloadMode) OpenDownload(ctx context.Context, resp *Response) (io.WriteCloser, error) {
	return openFileName(d, fileSystemFrom(ctx, nil), resp)
}

func (d DownloadMode) FileName(r *Response) string {
	uri := r.Request.URL.RequestURI()
	switch d {
	case PreserveRequestFile:
		return fileName(uri)

	case PreserveRequestPath:
		return strings.TrimLeft(uri, "/")

	default:
		panic("unreachable!")
	}
}

func (b basicDownloader) OpenDownload(_ context.Context, _ *Response) (io.WriteCloser, error) {
	return nopWriteCloser{b.w}, nil
}

func (nopWriteCloser) Close() error {
	return nil
}

func fileName(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+len("/"):]
	}
	return s
}

func fileSystemFrom(ctx context.Context, preferred fs.FS) (res cli.FS) {
	if preferred != nil {
		res = cli.NewFS(preferred)
		return
	}

	defer func() {
		// TODO This recovery is necessary until joe-cli provides a version that
		// doesn't panic on FromContext
		if rvr := recover(); rvr != nil {
			res = cli.NewSysFS(cli.DirFS("."), os.Stdin, os.Stdout)
		}
	}()

	res = cli.NewFS(cli.FromContext(ctx).FS)
	return
}
