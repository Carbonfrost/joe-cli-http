package httpclient

import (
	"bytes"
	"context"
	"crypto"
	"encoding"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"
)

type Integrity struct {
	Hash   crypto.Hash
	Digest []byte
}

type integrityDownloader struct {
	Downloader
	integrity Integrity
}

type integrityChecker struct {
	io.Writer
	output       io.Closer
	hash         hash.Hash
	expectedHash []byte
}

func NewIntegrityDownloadMiddleware(i Integrity) func(Downloader) Downloader {
	return func(d Downloader) Downloader {
		return NewIntegrityDownloader(i, d)
	}
}

func NewIntegrityDownloader(i Integrity, d Downloader) Downloader {
	return &integrityDownloader{
		Downloader: d,
		integrity:  i,
	}
}

func newIntegrityChecker(output io.Writer, hasher hash.Hash, expectedHash []byte) *integrityChecker {
	c, ok := output.(io.Closer)
	if !ok {
		c = io.NopCloser(nil)
	}
	return &integrityChecker{
		Writer:       io.MultiWriter(output, hasher),
		output:       c,
		hash:         hasher,
		expectedHash: expectedHash,
	}
}

func (c *integrityChecker) Close() error {
	actual := c.hash.Sum(nil)
	if !bytes.Equal(c.expectedHash, actual) {
		return fmt.Errorf("response body does not match expected hash")
	}
	return c.output.Close()
}

func (i *integrityDownloader) OpenDownload(c context.Context, r *Response) (io.WriteCloser, error) {
	output, err := i.Downloader.OpenDownload(c, r)
	if err != nil {
		return nil, err
	}

	if !i.integrity.Hash.Available() {
		return nil, fmt.Errorf("specified hash %v is not linked in the binary", i.integrity.Hash)
	}

	hash := i.integrity.Hash.New()
	return newIntegrityChecker(output, hash, i.integrity.Digest), nil
}

func ParseIntegrity(s string) (Integrity, error) {
	var (
		i   Integrity
		err error
	)

	hash, digest, ok := strings.Cut(s, ":")
	if !ok {
		return i, fmt.Errorf("invalid subresource integrity string")
	}

	i.Hash, err = parseHash(hash)
	if err != nil {
		return i, fmt.Errorf("invalid subresource integrity string: %w", err)
	}

	i.Digest, err = hex.DecodeString(digest)
	if err != nil {
		return i, fmt.Errorf("invalid subresource integrity string: %w", err)
	}
	if len(i.Digest) != i.Hash.Size() {
		return i, fmt.Errorf("invalid subresource integrity string: expected digest length %d", hex.EncodedLen(i.Hash.Size()))
	}
	return i, nil
}

func (i *Integrity) UnmarshalText(b []byte) error {
	n, err := ParseIntegrity(string(b))
	if err != nil {
		return err
	}
	*i = n
	return nil
}

func parseHash(name string) (crypto.Hash, error) {
	switch name {
	case "md5":
		return crypto.MD5, nil
	case "ripemd160":
		return crypto.RIPEMD160, nil
	case "sha1":
		return crypto.SHA1, nil
	case "sha224":
		return crypto.SHA224, nil
	case "sha256":
		return crypto.SHA256, nil
	case "sha384":
		return crypto.SHA384, nil
	case "sha512":
		return crypto.SHA512, nil
	case "sha512-224":
		return crypto.SHA512_224, nil
	case "sha512-256":
		return crypto.SHA512_256, nil
	default:
		return 0, fmt.Errorf("unknown algorithm: %s", name)
	}
}

var _ encoding.TextUnmarshaler = (*Integrity)(nil)
