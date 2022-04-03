package httpclient

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
)

type CipherSuites []uint16

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

func doListCiphers() {
	listCiphers(tls.CipherSuites())
	listCiphers(tls.InsecureCipherSuites())
}

func listCiphers(items []*tls.CipherSuite) {
	for _, cs := range items {
		fmt.Printf("%s\t%s\n", cs.Name, sprintSupportedVersions(cs.SupportedVersions))
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
