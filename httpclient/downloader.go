package httpclient

import (
	"io"

	"github.com/Carbonfrost/joe-cli"
)

type Downloader interface {
	OpenDownload(*Response) (io.Writer, error)
}

type directAdapter struct {
	*cli.File
}

func (d *directAdapter) OpenDownload(_ *Response) (io.Writer, error) {
	return d.Create()
}
