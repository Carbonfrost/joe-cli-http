// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"math/rand"
	"net"
	"net/url"
	"runtime"
	"time"

	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

func ExpandGlobals(k string) any {
	switch k {
	case "go.version":
		return runtime.Version()
	case "wig.version":
		return build.Version
	case "time", "time.now":
		return time.Now()
	case "time.now.utc":
		return time.Now().UTC()
	case "random":
		return rand.Int()
	case "random.float":
		return rand.Float64()
	}
	return nil
}

func ExpandURL(u *url.URL) expander.Interface {
	return expander.Func(func(k string) any {
		switch k {
		case "scheme":
			return u.Scheme
		case "user":
			return u.User.Username()
		case "userInfo":
			return u.User.String()
		case "host":
			return u.Host
		case "path":
			return u.Path
		case "query":
			return u.Query().Encode()
		case "fragment":
			return u.Fragment
		case "requestURI":
			return u.RequestURI()
		case "authority":
			var res string
			if u.User != nil {
				res = u.User.String() + "@"
			}
			if u.Port() == "" {
				res += u.Host
			} else {
				res += net.JoinHostPort(u.Host, u.Port())
			}
			return res
		}
		return nil
	})
}
