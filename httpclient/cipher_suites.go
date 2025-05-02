// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

type CipherSuites []uint16
type CurveIDs []tls.CurveID

var (
	curves = []tls.CurveID{
		tls.CurveP256,
		tls.CurveP384,
		tls.CurveP521,
		tls.X25519,
	}

	curveNames = []string{
		"P256",
		"P384",
		"P521",
		"X25519",
	}
)

func (c *CipherSuites) Set(arg string) error {
	items := *c
	for _, name := range strings.Split(arg, ",") {
		id, err := cipherSuite(name)
		if err != nil {
			return err
		}
		items = append(items, id)
	}
	*c = items
	return nil
}

func (c *CipherSuites) String() string {
	s := make([]string, len(*c))
	for i, id := range *c {
		s[i] = tls.CipherSuiteName(id)
	}
	return strings.Join(s, ",")
}

func (c *CipherSuites) Synopsis() string {
	return "SUITES"
}

func (c *CurveIDs) Set(arg string) error {
	items := *c
	for _, name := range strings.Split(arg, ",") {
		id, err := curve(name)
		if err != nil {
			return err
		}
		items = append(items, id)
	}
	*c = items
	return nil
}

func (c *CurveIDs) String() string {
	s := make([]string, len(*c))
	for i, id := range *c {
		s[i] = curveName(id)
	}
	return strings.Join(s, ",")
}

func (c *CurveIDs) Synopsis() string {
	return "CURVES"
}

func cipherSuite(name string) (uint16, error) {
	for _, c := range tls.CipherSuites() {
		if c.Name == name {
			return c.ID, nil
		}
	}
	for _, c := range tls.InsecureCipherSuites() {
		if c.Name == name {
			return c.ID, nil
		}
	}

	i, err := strconv.ParseUint(name, 16, 16)
	return uint16(i), err
}

func doListCiphers(c *cli.Context) error {
	listCiphers(c.Stdout, tls.CipherSuites())
	listCiphers(c.Stdout, tls.InsecureCipherSuites())
	return nil
}

func listCiphers(w io.Writer, items []*tls.CipherSuite) {
	for _, cs := range items {
		fmt.Fprintf(w, "%s\t%s\n", cs.Name, sprintSupportedVersions(cs.SupportedVersions))
	}
}

func sprintSupportedVersions(v []uint16) string {
	res := make([]string, len(v))
	for i, e := range v {
		res[i] = versionString(e)
	}
	return strings.Join(res, ", ")
}

func versionString(e uint16) string {
	switch e {
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return fmt.Sprintf("0x%04X", e)
	}
}

func doListCurves(c *cli.Context) error {
	for _, name := range curveNames {
		fmt.Fprintf(c.Stdout, "%s\n", name)
	}
	return nil
}

func curveName(c tls.CurveID) string {
	for i := range curves {
		if c == curves[i] {
			return curveNames[i]
		}
	}
	return ""
}

func curve(c string) (tls.CurveID, error) {
	for i := range curves {
		if c == curveNames[i] {
			return curves[i], nil
		}
	}
	return 0, fmt.Errorf("unknown curve or named group %q", c)
}
