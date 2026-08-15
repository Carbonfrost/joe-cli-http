// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
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

func NewIntegrityDownloaderMiddleware(i Integrity) DownloaderMiddleware {
	return func(_ context.Context, d Downloader) Downloader {
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

var (
	integrityAlgorithms = map[string]crypto.Hash{
		"md5":        crypto.MD5,
		"ripemd160":  crypto.RIPEMD160,
		"sha1":       crypto.SHA1,
		"sha224":     crypto.SHA224,
		"sha256":     crypto.SHA256,
		"sha384":     crypto.SHA384,
		"sha512":     crypto.SHA512,
		"sha512-224": crypto.SHA512_224,
		"sha512-256": crypto.SHA512_256,
	}
)

func parseHash(name string) (crypto.Hash, error) {
	h, ok := integrityAlgorithms[name]
	if !ok {
		return 0, fmt.Errorf("unknown algorithm: %s", name)
	}
	return h, nil
}

// ListIntegrityAlgorithms provides an action which lists the integrity algorithms
func ListIntegrityAlgorithms() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-integrity-algorithms",
			HelpText: "List hash algorithms that can be used with integrity checker",
			Category: responseOptions,
			Options:  cli.Exits,
			Value:    new(bool),
		},
		bind.Call(doListIntegrityAlgorithms, bind.Stdout()),
		tagged,
	)
}

func doListIntegrityAlgorithms(w io.Writer) error {
	for name, alg := range integrityAlgorithms {
		available := "not available"
		if alg.Available() {
			available = "available"
		}
		fmt.Fprintf(w, "%s\t%s\n", name, available)
	}
	return nil
}

var _ encoding.TextUnmarshaler = (*Integrity)(nil)
