// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"maps"
)

// Vars provides template variables
type Vars map[string]any

func (t Vars) Add(v ...*Var) {
	for _, u := range v {
		switch val := u.Value.(type) {
		case map[string]any:
			t.setMapHelper(u.Name, val)
		case []any:
			t.setArrayHelper(u.Name, val)
		case string:
			t.setStringHelper(u.Name, val)
		default:
			panic("unreachable!")
		}
	}
}

func (t Vars) Update(u map[string]any) (err error) {
	for k, v := range u {
		switch val := v.(type) {
		case map[string]any:
			err = t.setMapHelper(k, val)
		case []any:
			err = t.setArrayHelper(k, val)
		case string:
			err = t.setStringHelper(k, val)
		default:
			panic("unreachable!")
		}
		if err != nil {
			return
		}
	}
	return
}

func (t Vars) Items() []*Var {
	res := make([]*Var, 0, len(t))
	for k, v := range t {
		var item *Var
		switch val := v.(type) {
		case map[string]any:
			item = MapVar(k, val)
		case []any:
			item = ArrayVar(k, val)
		case string:
			item = StringVar(k, val)
		default:
			item = StringVar(k, fmt.Sprint(val))
		}

		res = append(res, item)
	}
	return res
}

func (t Vars) setStringHelper(name, value string) error {
	t[name] = value
	return nil
}

func (t Vars) setArrayHelper(name string, values []any) error {
	if current, ok := t[name]; ok {
		switch c := current.(type) {
		case []any:
			t[name] = append(c, values...)
			return nil
		case string:
			t[name] = append([]any{c}, values...)
			return nil
		case map[string]any:
			for _, v := range values {
				c[fmt.Sprint(v)] = ""
			}
			t[name] = c
			return nil
		}
	}

	t[name] = values
	return nil
}

func (t Vars) setMapHelper(name string, values map[string]any) error {
	if current, ok := t[name]; ok {
		switch c := current.(type) {
		case []any:
			return fmt.Errorf("existing value is array, cannot apply map")
		case string:
			return fmt.Errorf("existing value is array, cannot apply string")
		case map[string]any:
			maps.Copy(c, values)
			t[name] = c
			return nil
		}
	}

	t[name] = values
	return nil
}

func (t *Vars) Set(arg string) error {
	if *t == nil {
		*t = Vars{}
	}

	p := *t
	k, value, ok := strings.Cut(arg, "=")
	if !ok {
		p[k] = k
		*t = p
		return nil
	}
	switch {
	case strings.HasPrefix(value, "["):
		if !strings.HasSuffix(value, "]") {
			return fmt.Errorf("expected `]' to end array")
		}
		tokens := strings.Split(trimAffix(value, "[", "]"), ",")
		items := make([]any, len(tokens))
		for i, t := range tokens {
			items[i] = strings.TrimSpace(t)
		}
		return t.andDeref(p, p.setArrayHelper(k, items))

	case strings.HasPrefix(value, "{"):
		if !strings.HasSuffix(value, "}") {
			return fmt.Errorf("expected `}' to end array")
		}
		tokens := strings.Split(trimAffix(value, "{", "}"), ",")
		items := map[string]any{}
		for _, t := range tokens {
			k, v, _ := strings.Cut(t, ":")
			items[k] = v
		}
		return t.andDeref(p, p.setMapHelper(k, items))

	default:
		return t.andDeref(p, p.setStringHelper(k, value))
	}
}

func (t *Vars) andDeref(v Vars, err error) error {
	if err != nil {
		return err
	}
	*t = v
	return nil
}

func (t Vars) String() string {
	var (
		buf   bytes.Buffer
		comma bool
	)
	for k, v := range t {
		if comma {
			buf.WriteString(",")
		}
		comma = true

		buf.WriteString(k)
		buf.WriteString("=")
		switch val := v.(type) {
		case map[string]any:
			buf.WriteString(printMap(val))
		case []any:
			buf.WriteString(printArray(val))
		case string:
			if val == k {
				buf.Truncate(buf.Len() - 1)
				continue
			}
			buf.WriteString(val)
		default:
			buf.WriteString(fmt.Sprint(val))
		}
	}
	return buf.String()
}

func (t *Vars) SetData(r io.Reader) error {
	return json.NewDecoder(r).Decode(t)
}

func (t *Vars) Reset() {
	*t = Vars{}
}

func (t *Vars) Copy() *Vars {
	res := *t
	return &res
}

func printMap(v map[string]any) string {
	items := make([]string, len(v))
	var i int
	for k, atom := range v {
		items[i] = k + ":" + cli.Quote(fmt.Sprint(atom))
		i++
	}
	sort.Strings(items)
	return "{" + strings.Join(items, ",") + "}"
}

func printArray(v []any) string {
	items := make([]string, len(v))
	for i, atom := range v {
		items[i] = cli.Quote(fmt.Sprint(atom))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func trimAffix(s, l, r string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, l), r)
}

var _ flag.Value = (*Vars)(nil)
