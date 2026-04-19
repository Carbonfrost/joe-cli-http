// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"fmt"
	"net"
)

const (
	// DefaultInterfaceResolver is the default interface resolver
	DefaultInterfaceResolver = defaultResolver(0)
)

// InterfaceResolver resolves the network interface to use
// for connections
type InterfaceResolver interface {
	// Resolve converts a location, typically an IP address
	// or adapter name, into a TCP address
	Resolve(context.Context, string) (*net.TCPAddr, error)
}

type defaultResolver int

func (defaultResolver) Resolve(_ context.Context, v string) (*net.TCPAddr, error) {
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
	return nil, fmt.Errorf("failed to resolve %q", v)
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
