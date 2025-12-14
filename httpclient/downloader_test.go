// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient_test

import (
	"context"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"slices"
	"testing/fstest"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"

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
				var testFileSystem = newMemoryWrapperFS()

				request, _ := http.NewRequest("GET", "https://example.com/wherever", nil)
				e := httpclient.NewFileDownloader(pat, testFileSystem).(fileNameDownloader)

				for i := range 3 {
					_, err := e.OpenDownload(context.Background(), &httpclient.Response{
						Response: &http.Response{
							Request:       request,
							ContentLength: int64((1 + i) * 10),
						},
					})
					Expect(err).NotTo(HaveOccurred())
				}

				actual := testFileSystem.Files()
				Expect(actual).To(ConsistOf(expected))
			},
				Entry("nominal", "file.txt", []string{"file.txt", "file.txt.1", "file.txt.2"}),
				Entry("expression", "file%(contentLength).txt", []string{"file10.txt", "file20.txt", "file30.txt"}),
				Entry("index", "file#%(index).txt", []string{"file#0.txt", "file#1.txt", "file#2.txt"}),
				Entry("index suffix", "file%(index.suffix).txt", []string{"file.txt", "file.1.txt", "file.2.txt"}),
			)
		})
	})
})

func newMemoryWrapperFS() *wrapperFS {
	memory := fstest.MapFS{}
	return &wrapperFS{
		FS:     cli.NewFS(memory),
		memory: memory,
	}
}

type fileNameDownloader interface {
	httpclient.Downloader
	FileName(*httpclient.Response) string
}

type wrapperFS struct {
	cli.FS
	memory fstest.MapFS
}

type wrapperFile struct {
	file *fstest.MapFile
}

func (w *wrapperFS) Create(name string) (fs.File, error) {
	file := new(fstest.MapFile)
	w.memory[name] = file
	return &wrapperFile{file}, nil
}

func (w wrapperFS) Open(name string) (fs.File, error) {
	return w.FS.Open(name)
}

func (w *wrapperFS) Files() []string {
	return slices.Collect(maps.Keys(w.memory))
}

func (w *wrapperFile) Write(p []byte) (n int, err error) {
	w.file.Data = append(w.file.Data, p...)
	return len(p), nil
}

func (w *wrapperFile) Close() error {
	return nil
}

func (w *wrapperFile) Read([]byte) (int, error) {
	panic("unimplemented")
}

func (w *wrapperFile) Stat() (fs.FileInfo, error) {
	panic("unimplemented")
}

var _ io.WriteCloser = (*wrapperFile)(nil)
