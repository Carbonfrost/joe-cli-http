package httpclient

import (
	"errors"
	"net"
)

type InterfaceResolver interface {
	Resolve(string) (*net.TCPAddr, error)
}

type defaultResolver struct{}

func (*defaultResolver) Resolve(v string) (*net.TCPAddr, error) {
	ip := net.ParseIP(v)
	if ip != nil {
		return resolveTCP(ip.String())
	}
	eth, err := net.InterfaceByName(v)
	if err != nil {
		return nil, err
	}
	addrs, err := eth.Addrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		if a.Network() == "ip+net" {
			return resolveIPNet(a.String())
		}
	}
	return nil, errors.New("failed to resolve " + v)
}

func resolveTCP(value string) (*net.TCPAddr, error) {
	return net.ResolveTCPAddr("tcp", net.JoinHostPort(value, "0"))
}

func resolveIPNet(value string) (*net.TCPAddr, error) {
	ip, _, err := net.ParseCIDR(value)
	if err != nil {
		return nil, err
	}
	return resolveTCP(ip.String())
}
