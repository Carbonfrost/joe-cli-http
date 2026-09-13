// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"math/rand"
	"net"
	"net/url"
	"runtime"
	"strings"
	"time"
	"uuid"

	"github.com/Carbonfrost/joe-cli-http/internal/build"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

// ExpandGlobals provides an exander for global variables:
//
//   - go.version
//   - wig.version
//   - time, time.now, time.now.utc (and what expander.Time provides)
//   - random (or random.int)
//   - random.float
//   - random.uuid (or random.uuidv4)
//   - random.uuidv7
func ExpandGlobals() expander.Interface {
	return expander.Compose(
		globals,
		expander.Prefix("time.utc", expandTimeNowUTC),
		expander.Prefix("time", expandTimeNow),
		expander.Prefix("time.now", expandTimeNow),

		expander.Func(func(k string) any {
			switch k {
			case "time", "time.now":
				return time.Now()
			case "time.now.utc":
				return time.Now().UTC()
			case "random", "random.int":
				return rand.Int()
			case "random.float":
				return rand.Float64()
			case "random.uuid", "random.uuidv4":
				return uuid.NewV4()
			case "random.uuidv7":
				return uuid.NewV7()
			}
			return nil
		}))
}

var (
	globals = expander.Map{
		"go.version":  runtime.Version(),
		"wig.version": build.Version,
	}
	expandTimeNow expander.Func = func(k string) any {
		return expander.Time(time.Now()).Expand(k)
	}
	expandTimeNowUTC expander.Func = func(k string) any {
		return expander.Time(time.Now().UTC()).Expand(k)
	}
)

// ExpandURLValues provides an expander for [url.Values]
func ExpandURLValues(v url.Values) expander.Interface {
	return expander.Func(func(s string) any {
		value, ok := v[s]
		if !ok {
			return nil
		}
		return strings.Join(value, ",")
	})
}

// ExpandURL provides an expander for [url.URL]
func ExpandURL(u *url.URL) expander.Interface {
	return expander.Compose(expander.Func(func(k string) any {
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
	}), expander.Prefix("query", ExpandURLValues(u.Query())))
}
