package httpclient

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

type Downloader interface {
	OpenDownload(*Response) (io.WriteCloser, error)
}

type DownloadMode int

type directAdapter struct {
	*cli.File
}

type basicDownloader struct {
	w io.Writer
}

type nopWriteCloser struct {
	io.Writer
}

// Download modes
const (
	PreserveRequestFile DownloadMode = iota
	PreserveRequestPath
)

func NewDownloaderTo(w io.Writer) Downloader {
	return basicDownloader{w}
}

func NewFileDownloader(f *cli.File) Downloader {
	return &directAdapter{f}
}

func (d *directAdapter) OpenDownload(_ *Response) (io.WriteCloser, error) {
	ensureDirectory(d.Dir())
	w, err := d.Create()
	if err != nil {
		return nil, err
	}
	return w.(io.WriteCloser), err
}

func (d DownloadMode) OpenDownload(resp *Response) (io.WriteCloser, error) {
	fn := d.FileName(resp)
	if fn == "" {
		return nil, fmt.Errorf("cannot download file: the request path has no file name")
	}
	ensureDirectory(filepath.Dir(fn))
	return os.Create(fn)
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

func (b basicDownloader) OpenDownload(*Response) (io.WriteCloser, error) {
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

func ensureDirectory(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
}
