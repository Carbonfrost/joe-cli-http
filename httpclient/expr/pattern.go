package expr

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli-http/internal/build"
)

const defaultAccessLog = `- - [%(start:02/Jan/2006 15:04:05)] "%(method) %(urlPath) %(protocol)" %(statusCode) -`

var (
	patternRegexp = regexp.MustCompile(`%\((.+?)\)`)

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
	return compilePatternCore([]byte(pattern), patternRegexp)
}

func CompilePattern(pattern string, start string, end string) *Pattern {
	pp := regexp.MustCompile(regexp.QuoteMeta(start) + `(.+?)` + regexp.QuoteMeta(end))
	return compilePatternCore([]byte(pattern), pp)
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

func ExpandMap(m map[string]any) Expander {
	return func(k string) any {
		v, ok := m[k]
		if !ok {
			return UnknownToken(k)
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

func (l *literal) Format(expand Expander) any {
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
	switch t := value.(type) {
	case time.Time:
		res = t.Format(f.format)
	case error:
		res = fmt.Sprintf("%%!(%s)", t)
	default:
		res = fmt.Sprintf("%"+f.format, value)
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

func compilePatternCore(content []byte, pat *regexp.Regexp) *Pattern {
	allIndexes := pat.FindAllSubmatchIndex(content, -1)
	result := []expr{}
	var repr bytes.Buffer

	var index int
	for _, loc := range allIndexes {
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

// convertWSExprs sets up trailing whitespace in format expressions
// by looking for successive whitespace expansions:
//
//	%(space)
//	%(tab)
//	%(newline)
func convertWSExprs(exprs []expr) []expr {
	var res []expr
	for i := 0; i < len(exprs); i++ {
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
		return &formatExpr{name: name, format: "v"}
	}

	return &formatExpr{name: name, format: nameAndFormat[1]}
}

var (
	_ Expander = ExpandGlobals
	_ Expander = ExpandColors
	_ Expander = Unknown
)
