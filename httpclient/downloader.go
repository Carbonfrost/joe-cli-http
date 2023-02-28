package httpclient

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

type Downloader interface {
	OpenDownload(*Response) (io.Writer, error)
}

type DownloadMode int

type directAdapter struct {
	*cli.File
}

// Download modes
const (
	PreserveRequestFile DownloadMode = iota
	PreserveRequestPath
)

func (d *directAdapter) OpenDownload(_ *Response) (io.Writer, error) {
	w, err := d.Create()
	return w.(io.Writer), err
}

func (d DownloadMode) OpenDownload(resp *Response) (io.Writer, error) {
	fn := d.FileName(resp)
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
