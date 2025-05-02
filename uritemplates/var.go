// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates

import (
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

// Var is a variable in a URI template
type Var struct {
	Name  string
	Value any

	state *varState
}

// VarType enumerates the types of variables in the expression of a URI template
type VarType int

type varState struct {
	Name  string
	Value any

	myType VarType
	state  varStateEnum
}

type varStateEnum int

const (
	varStateInitial varStateEnum = iota
	varStateSawType
	varStateSawName
	varStateEndState
)

// Types of variables
const (
	String VarType = iota
	Array
	Map
	maxVarType
)

var (
	inlineFormat = regexp.MustCompile(`^(array|string|map|a|s|m),(.+)=(.+)$`)

	varTypeStrings = [maxVarType]string{
		"string",
		"array",
		"map",
	}
)

func ArrayVar(name string, values ...any) *Var {
	return &Var{
		Name:  name,
		Value: values,
	}
}

func MapVar(name string, values map[string]any) *Var {
	return &Var{
		Name:  name,
		Value: values,
	}
}
func StringVar(name string, value string) *Var {
	return &Var{
		Name:  name,
		Value: value,
	}
}

func (v *Var) NewCounter() cli.ArgCounter {
	return new(varState)
}

func (v *Var) Reset() {
	v.Name = ""
	v.Value = nil
	v.state = nil
}

func (v *Var) Copy() *Var {
	res := *v
	return &res
}

func (v *Var) Set(arg string) error {
	if v.state == nil {
		v.state = new(varState)
	}
	err := v.state.next(arg)
	v.Name = v.state.Name
	v.Value = v.state.Value
	return err
}

func (*Var) Synopsis() string {
	return "[type,]name=value"
}

func (v *Var) String() string {
	value := func() string {
		switch o := v.Value.(type) {
		case map[string]any:
			panic("not impl")

		case []any:
			items := make([]string, len(o))
			for i := range o {
				items[i] = fmt.Sprint(o)
			}
			return cli.Join(items)

		default:
			return fmt.Sprint(o)
		}
	}()
	return fmt.Sprintf("%v,%s=%s", v.Type(), cli.Quote(v.Name), cli.Quote(value))
}

func (v *Var) Type() VarType {
	switch v.Value.(type) {
	case map[string]any:
		return Map
	case []any:
		return Array
	default:
		return String
	}
}

func (c *varState) Take(arg string, _ bool) error {
	return c.next(arg)
}

func (c *varState) next(arg string) error {
	switch c.state {
	case varStateInitial:
		if c.parseInlineFormat(arg) {
			return nil
		}
		if strings.Contains(arg, "=") {
			return c.parseNVP(arg)
		}
		if tt, ok := varTypes(arg); ok {
			return c.parseType(tt)
		}

		return fmt.Errorf("invalid template var %q", arg)

	case varStateEndState:
		return cli.EndOfArguments

	case varStateSawName:
		return c.parseValue(arg)

	case varStateSawType:
		if strings.Contains(arg, "=") {
			return c.parseNVP(arg)
		}
		return c.parseName(arg)
	}

	panic("unreachable!")
}

func (c *varState) parseInlineFormat(arg string) bool {
	if inlineFormat.MatchString(arg) {
		t, rest, _ := strings.Cut(arg, ",")
		name, value, _ := strings.Cut(rest, "=")
		tt, _ := varTypes(t)

		c.myType = tt
		c.Name = name
		c.setValue(value)
		c.state = varStateEndState
		return true
	}
	return false
}

func (c *varState) parseNVP(arg string) error {
	name, value, _ := strings.Cut(arg, "=")

	c.Name = name
	c.setValue(value)
	c.state = varStateEndState
	return nil
}

func (c *varState) parseType(tt VarType) error {
	c.myType = tt
	c.state = varStateSawType
	return nil
}

func (c *varState) parseName(arg string) error {
	c.Name = arg
	c.state = varStateSawName
	return nil
}

func (c *varState) parseValue(arg string) error {
	c.setValue(arg)
	c.state = varStateEndState
	return nil
}

func (c *varState) setValue(value string) {
	switch c.myType {
	case Map:
		k, v, _ := strings.Cut(value, "=")
		c.Value = map[string]any{
			k: v,
		}
	case String:
		c.Value = value
	case Array:
		c.Value = []any{value}
	}
}

func (c *varState) Done() error {
	if c.state == varStateEndState {
		return nil
	}
	return nil
}

func (t VarType) String() string {
	return varTypeStrings[int(t)]
}

func varTypes(m string) (VarType, bool) {
	switch m {
	case "a", "array":
		return Array, true
	case "s", "string":
		return String, true
	case "m", "map":
		return Map, true
	}
	return 0, false
}

var (
	_ flag.Value     = (*Var)(nil)
	_ cli.ArgCounter = (*varState)(nil)
)
