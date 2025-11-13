// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"math/rand"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli-http/internal/build"
)

const defaultAccessLog = `- - [%(start:02/Jan/2006 15:04:05)] "%(method:C) %(urlPath) %(protocol)" %(statusCode:C) -`

var (

	// vt100 ansi codes
	colors = map[string]int{
		"default":       39,
		"black":         30,
		"red":           31,
		"green":         32,
		"yellow":        33,
		"blue":          34,
		"magenta":       35,
		"cyan":          36,
		"gray":          37,
		"darkGray":      90,
		"brightRed":     91,
		"brightGreen":   92,
		"brightYellow":  93,
		"brightBlue":    94,
		"brightMagenta": 95,
		"brightCyan":    96,
		"white":         97,

		"default.bg": 49,
		"black.bg":   40,
		"red.bg":     41,
		"green.bg":   42,
		"yellow.bg":  43,
		"blue.bg":    44,
		"magenta.bg": 45,
		"cyan.bg":    46,
		"gray.bg":    47,

		"reset":           0,
		"bold":            1,
		"faint":           2,
		"italic":          3,
		"underline":       4,
		"slow":            5,
		"fast":            6,
		"reverse":         7,
		"erase":           8,
		"strikethrough":   9,
		"doubleUnderline": 21,

		"bold.off":            22,
		"italic.off":          23,
		"underline.off":       24,
		"doubleUnderline.off": 24, // same
		"slow.off":            25,
		"fast.off":            26,
		"reverse.off":         27,
		"erase.off":           28,
		"strikethrough.off":   29,
	}

	meta    = map[string]*Pattern{}
	space   = nopExpr{"space"}
	tab     = nopExpr{"tab"}
	newline = nopExpr{"newline"}
	empty   = nopExpr{"empty"}
)

func init() {
	meta["accessLog.default"] = Compile(defaultAccessLog)
}

type Pattern struct {
	exprs []expr

	// textual representation, which is the value which was compiled after meta expr have been expanded
	repr string
}

// Expander converts the given string key into its variable expansion
type Expander func(string) any

type expr interface {
	Format(expand Expander) any
}

type formatExpr struct {
	name        string
	format      string
	trailingOpt string // optional whitespace iff the expr evaluates non-empty
}

// nopExpr is reserved for whitespace expressions that are reserved names
type nopExpr struct {
	name string
}

type literal struct {
	text     string
	trailing string // whitespace after literal (produced by ws expressions)
}

// Renderer is a specialized writer that understands writing to multiple
// files and the corresponding variable support
type Renderer struct {
	io.Writer
	out, err io.Writer
}

func (r *Renderer) expandFiles(k string) any {
	switch k {
	case "stderr":
		r.Writer = r.err
		return ""

	case "stdout":
		r.Writer = r.out
		return ""
	}
	return nil
}

func NewRenderer(stdout, stderr io.Writer) *Renderer {
	return &Renderer{
		Writer: stdout,
		out:    stdout,
		err:    stderr,
	}
}

func Fprint(w io.Writer, pattern *Pattern, e Expander) (count int, err error) {
	// Implicitly upgrade w to *Renderer
	if r, ok := w.(*Renderer); ok {
		e = ComposeExpanders(r.expandFiles, e)
	}
	for _, item := range pattern.exprs {
		count, err = fmt.Fprint(w, item.Format(e))
		if err != nil {
			break
		}
	}
	return
}

func Compile(pattern string) *Pattern {
	return CompilePattern(pattern, "%(", ")")
}

func CompilePattern(pattern string, start string, end string) *Pattern {
	endBytes := []byte(end)
	if len(endBytes) > 1 {
		panic("end sequence must be one byte")
	}
	return compilePatternCore([]byte(pattern), []byte(start), endBytes[0])
}

// Prefix provides an expander which looks for and cuts a given prefix
// and delegates the result to the underlying expander
func Prefix(p string, e Expander) Expander {
	prefix := p + "."
	return func(k string) any {
		if name, ok := strings.CutPrefix(k, prefix); ok {
			return e(name)
		}
		return nil
	}
}

func ExpandEnv(s string) any {
	result, ok := os.LookupEnv(s)
	if ok {
		return result
	}
	return nil
}

func ExpandMap(m map[string]any) Expander {
	return func(k string) any {
		v, ok := m[k]
		if !ok {
			return nil
		}
		return v
	}
}

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

func ExpandColors(k string) any {
	if a, ok := colors[k]; ok {
		return fmt.Sprintf("\x1b[%dm", a)
	}
	return nil
}

func ExpandURL(u *url.URL) Expander {
	return func(k string) any {
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
	}
}

func ComposeExpanders(expanders ...Expander) Expander {
	return func(k string) any {
		for _, x := range expanders {
			v := x(k)
			if v != nil {
				return v
			}
		}
		return nil
	}
}

func Unknown(s string) any {
	return UnknownToken(s)
}

func UnknownToken(tok string) error {
	return fmt.Errorf("unknown: %s", tok)
}

func (l *literal) Format(_ Expander) any {
	return l.text + l.trailing
}

func (nopExpr) Format(Expander) any {
	return ""
}

func (n nopExpr) Space() string {
	switch n.name {
	case "space":
		return " "
	case "newline":
		return "\n"
	case "tab":
		return "\t"
	case "empty":
	}
	return ""
}

func (f *formatExpr) Format(expand Expander) any {
	var res string
	value := expand(f.name)
	format := f.format
	switch t := value.(type) {
	case time.Time:
		if format == "" {
			format = time.RFC3339
		}
		res = t.Format(format)
	case error:
		res = fmt.Sprintf("%%!(%s)", t)
	default:
		if format == "" {
			format = "v"
		}
		res = fmt.Sprintf("%"+format, value)
	}

	if res == "" {
		return res
	}
	return res + f.trailingOpt
}

func (p *Pattern) Expand(expand Expander) string {
	var b strings.Builder
	Fprint(&b, p, expand)
	return b.String()
}

func (p *Pattern) String() string {
	return p.repr
}

func compilePatternCore(content []byte, start []byte, end byte) *Pattern {
	allIndexes := findAllSubmatchIndex(content, start, end)
	result := []expr{}
	var repr bytes.Buffer

	var index int
	for loc := range allIndexes {
		if index < loc[0] {
			result = append(result, newLiteral(content[index:loc[0]]))
			repr.Write(content[index:loc[0]])
		}
		key := content[loc[2]:loc[3]]

		if m, ok := meta[string(key)]; ok {
			result = append(result, m.exprs...)
			repr.WriteString(m.String())
		} else {
			result = append(result, newExpr(key))
			repr.Write(content[loc[0]:loc[1]])
		}
		index = loc[1]
	}
	if index < len(content) {
		result = append(result, newLiteral(content[index:]))
		repr.Write(content[index:])
	}

	return &Pattern{
		exprs: convertWSExprs(result),
		repr:  repr.String(),
	}
}

// findAllSubmatchIndex provides the behavior of regexp FindAllSubmatchIndex
// except with simplifying assumptions but also detecting nested patterns. Only
// considers ASCII sequences, only allows a single byte end character.
func findAllSubmatchIndex(content, start []byte, end byte) iter.Seq[[4]int] {
	return func(yield func([4]int) bool) {
		var (
			nested     int
			lenStart   = len(start)
			lenContent = len(content)
			submatch   = func(i, j int) [4]int {
				return [4]int{i, j + 1, i + lenStart, j}
			}
		)

	OUTER:
		for i := 0; i < lenContent; i++ {
			c := content[i]
			// Submatch indexes - same as what regexp.Regexp.FindAllSubmatchIndex returns
			// 0 2    1,3
			// %(hello)
			if c != start[0] {
				continue
			}

			if bytes.Equal(content[i:i+lenStart], start) {
				for j := i + lenStart; j < lenContent; j++ {
					if content[j] == end {
						if nested == 0 {
							sub := submatch(i, j)
							if sub[2] != sub[3] {
								if !yield(sub) {
									return
								}
							}
							i = j
							continue OUTER

						} else {
							nested--
						}
					}

					// Detect nested occurrences
					if j < lenContent-lenStart && bytes.Equal(content[j:j+lenStart], start) {
						nested++
					}
				}
			}
		}
	}
}

// convertWSExprs sets up trailing whitespace in format expressions
// by looking for successive whitespace expansions:
//
//	%(space)
//	%(tab)
//	%(newline)
func convertWSExprs(exprs []expr) []expr {
	var res []expr
	for i := range exprs {
		if wse, ok := exprs[i].(nopExpr); ok {
			if len(res) == 0 {
				res = append(res, newLiteral([]byte(wse.Space())))
				continue
			}
			last := res[len(res)-1]

			switch prev := last.(type) {
			case *formatExpr:
				prev.trailingOpt += wse.Space()
			case *literal:
				prev.trailing += wse.Space()
			}
		} else {
			res = append(res, exprs[i])
		}
	}
	return res
}

func newLiteral(token []byte) expr {
	t := string(token)
	// Handle escape sequences
	if s, err := strconv.Unquote(`"` + t + `"`); err == nil {
		t = s
	}
	return &literal{t, ""}
}

func newExpr(token []byte) expr {
	nameAndFormat := strings.SplitN(string(token), ":", 2)
	name := nameAndFormat[0]
	switch name {
	case "space":
		return space
	case "newline":
		return newline
	case "tab":
		return tab
	case "empty":
		return empty
	}

	if len(nameAndFormat) == 1 {
		return &formatExpr{name: name, format: ""}
	}

	return &formatExpr{name: name, format: nameAndFormat[1]}
}

var (
	_ Expander = ExpandGlobals
	_ Expander = ExpandColors
	_ Expander = Unknown
)
