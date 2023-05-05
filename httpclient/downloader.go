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

type downloaderWithFileName interface {
	Downloader
	FileName(*Response) string
}

type directAdapter struct {
	*cli.File
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

// Download modes
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

func (s stripComponents) OpenDownload(resp *Response) (io.WriteCloser, error) {
	return openFileName(s, resp)
}

func openFileName(d downloaderWithFileName, resp *Response) (io.WriteCloser, error) {
	fn := d.FileName(resp)
	if fn == "" {
		return nil, fmt.Errorf("cannot download file: the request path has no file name")
	}
	ensureDirectory(filepath.Dir(fn))
	return os.Create(fn)
}

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
	return openFileName(d, resp)
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
