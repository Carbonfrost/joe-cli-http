package httpclient_test

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/spf13/afero"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DownloadMode", func() {

	Describe("FileName", func() {
		DescribeTable("examples",
			func(mode httpclient.DownloadMode, u string, expected string) {
				request, _ := http.NewRequest("GET", u, nil)
				cs := mode.FileName(&httpclient.Response{
					Response: &http.Response{
						Request: request,
					},
				})
				Expect(cs).To(Equal(expected))
			},
			Entry("empty", httpclient.PreserveRequestFile, "https://example.com/", ""),
			Entry("simple", httpclient.PreserveRequestFile, "https://example.com/hello", "hello"),
			Entry("query string", httpclient.PreserveRequestFile, "https://example.com/hello?a=b", "hello?a=b"),

			Entry("empty", httpclient.PreserveRequestPath, "https://example.com/", ""),
			Entry("simple", httpclient.PreserveRequestPath, "https://example.com/hello/world", "hello/world"),
			Entry("query string", httpclient.PreserveRequestPath, "https://example.com/hello/world?a=b", "hello/world?a=b"),
		)

		Context("when stripping components", func() {
			DescribeTable("examples", func(count int, expected string) {
				mode := httpclient.PreserveRequestPath
				request, _ := http.NewRequest("GET", "https://example.com/a/b/c/d/e/f.txt", nil)

				downloader := mode.WithStripComponents(count)
				downloader.(fileNameDownloader).FileName(&httpclient.Response{
					Response: &http.Response{
						Request: request,
					},
				})
			},
				Entry("zero", 0, "a/b/c/d/e/f.txt"),
				Entry("nominal", 1, "b/c/d/e/f.txt"),
				Entry("negative", -1, "e/f.txt"),
				Entry("exceeds limit", 99, "f.txt"),
				Entry("negative exceeds limit", -99, "a/b/c/d/e/f.txt"),
				Entry("negative exceeds limit (boundary)", -6, "a/b/c/d/e/f.txt"),
				Entry("negative (boundary)", -5, "b/c/d/e/f.txt"),
			)
		})
	})

	Describe("OpenDownload", func() {
		DescribeTable("examples",
			func(mode httpclient.DownloadMode, u string, expected string) {
				request, _ := http.NewRequest("GET", u, nil)
				_, err := mode.OpenDownload(context.Background(), &httpclient.Response{
					Response: &http.Response{
						Request: request,
					},
				})
				Expect(err).To(MatchError(expected))
			},
			Entry("empty request file", httpclient.PreserveRequestFile, "https://example.com/", "cannot download file: the request path has no file name"),
		)
	})
})

var _ = Describe("NewFileDownloader", func() {

	Describe("OpenDownload", func() {
		Context("when evaluating expr", func() {
			DescribeTable("examples", func(pat string, expected []string) {
				var testFileSystem = func() *wrapperFS {
					appFS := afero.NewMemMapFs()
					return &wrapperFS{Fs: appFS}
				}()

				request, _ := http.NewRequest("GET", "https://example.com/wherever", nil)
				e := httpclient.NewFileDownloader(pat, testFileSystem).(fileNameDownloader)

				for i := 0; i < 3; i++ {
					_, err := e.OpenDownload(context.Background(), &httpclient.Response{
						Response: &http.Response{
							Request:       request,
							ContentLength: int64((1 + i) * 10),
						},
					})
					Expect(err).NotTo(HaveOccurred())
				}

				actual := testFileSystem.ActualFiles
				Expect(actual).To(Equal(expected))
			},
				Entry("nominal", "file.txt", []string{"file.txt", "file.txt.1", "file.txt.2"}),
				Entry("expression", "file%(contentLength).txt", []string{"file10.txt", "file20.txt", "file30.txt"}),
				Entry("index", "file#%(index).txt", []string{"file#0.txt", "file#1.txt", "file#2.txt"}),
				Entry("index suffix", "file%(index.suffix).txt", []string{"file.txt", "file.1.txt", "file.2.txt"}),
			)
		})
	})
})

type fileNameDownloader interface {
	httpclient.Downloader
	FileName(*httpclient.Response) string
}

type wrapperFS struct {
	afero.Fs
	ActualFiles []string
}

func (w *wrapperFS) Create(name string) (fs.File, error) {
	w.ActualFiles = append(w.ActualFiles, name)
	return w.Fs.Create(name)
}

func (w wrapperFS) Open(name string) (fs.File, error) {
	return w.Fs.Open(name)
}
