package httpclient_test

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Integrity", func() {

	Describe("ParseIntegrity", func() {

		var (
			bytes32 [32]byte
			hex32   = strings.Repeat("0", 32)
			hex40   = strings.Repeat("0", 40)
			hex56   = strings.Repeat("0", 56)
			hex64   = strings.Repeat("0", 64)
			hex96   = strings.Repeat("0", 96)
			hex128  = strings.Repeat("0", 128)
		)

		DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
			actual, err := httpclient.ParseIntegrity(text)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(expected)
		},
			Entry("nominal", "sha256:"+hex64, Equal(httpclient.Integrity{crypto.SHA256, bytes32[:]})),
			Entry("md5", "md5:"+hex32, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.MD5)})),
			Entry("ripemd160", "ripemd160:"+hex40, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.RIPEMD160)})),
			Entry("sha1", "sha1:"+hex40, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA1)})),
			Entry("sha224", "sha224:"+hex56, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA224)})),
			Entry("sha256", "sha256:"+hex64, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA256)})),
			Entry("sha384", "sha384:"+hex96, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA384)})),
			Entry("sha512", "sha512:"+hex128, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA512)})),
			Entry("sha512-224", "sha512-224:"+hex56, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA512_224)})),
			Entry("sha512-256", "sha512-256:"+hex64, MatchFields(IgnoreExtras, Fields{"Hash": Equal(crypto.SHA512_256)})),
		)

		DescribeTable("errors", func(text string, expectedError types.GomegaMatcher) {
			_, err := httpclient.ParseIntegrity(text)
			Expect(err).To(expectedError)
		},
			Entry("invalid", "shabazz1024:", MatchError("invalid subresource integrity string: unknown algorithm: shabazz1024")),
			Entry("invalid hex string", "sha256:lolo", MatchError("invalid subresource integrity string: encoding/hex: invalid byte: U+006C 'l'")),
			Entry("wrong length error", "sha256:234234", MatchError("invalid subresource integrity string: expected digest length 64")),
		)
	})

})

var _ = Describe("IntegrityDownloader", func() {

	It("computes and compares the hash of the response body", func() {
		testResponse := &httpclient.Response{
			&http.Response{
				Body: io.NopCloser(bytes.NewBufferString("this the response body")),
			},
		}

		expectedHash, _ := hex.DecodeString("e5f0df198cb4543513a9b1a99468828753e3b8fb")

		d := httpclient.NewIntegrityDownloader(httpclient.Integrity{
			Hash:   crypto.SHA1,
			Digest: expectedHash,
		}, httpclient.NewDownloaderTo(io.Discard))
		writer, _ := d.OpenDownload(testResponse)
		_ = testResponse.CopyTo(writer)

		err := writer.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error if hash mismatch", func() {
		testResponse := &httpclient.Response{
			&http.Response{
				Body: io.NopCloser(bytes.NewBufferString("this the response body")),
			},
		}

		d := httpclient.NewIntegrityDownloader(httpclient.Integrity{Hash: crypto.SHA1}, httpclient.NewDownloaderTo(io.Discard))
		writer, _ := d.OpenDownload(testResponse)
		_ = testResponse.CopyTo(writer)

		err := writer.Close()
		Expect(err).To(MatchError("response body does not match expected hash"))
	})

	It("writes to the output of inner downloader", func() {
		testResponse := &httpclient.Response{
			&http.Response{
				Body: io.NopCloser(bytes.NewBufferString("this the response body")),
			},
		}

		var buf bytes.Buffer
		d := httpclient.NewIntegrityDownloader(httpclient.Integrity{Hash: crypto.SHA1}, httpclient.NewDownloaderTo(&buf))
		writer, _ := d.OpenDownload(testResponse)
		_ = testResponse.CopyTo(writer)
		_ = writer.Close()

		Expect(string(buf.Bytes())).To(Equal("this the response body"))
	})
})
