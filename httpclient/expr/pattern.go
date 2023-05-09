package expr

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli-http/internal/build"
)

const defaultAccessLog = `- - [%(start:02/Jan/2006 15:04:05)] "%(method) %(urlPath) %(protocol)" %(status) -`

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

	meta = map[string]*Pattern{}
)

func init() {
	meta["accessLog.default"] = Compile(defaultAccessLog)
}

type Pattern struct {
	exprs []expr
}

type Expander func(string) any

type expr interface {
	Format(expand Expander) any
	WriteTo(io.StringWriter)
}

type formatExpr struct {
	name   string
	format string
}

type literal struct {
	text string
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
	if name, ok := strings.CutPrefix(k, "color."); ok {
		if a, ok := colors[name]; ok {
			return fmt.Sprintf("\x1b[%dm", a)
		}
	}
	return nil
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
	return l.text
}

func (l *literal) WriteTo(w io.StringWriter) {
	w.WriteString(l.text)
}

func (f *formatExpr) Format(expand Expander) any {
	value := expand(f.name)
	switch t := value.(type) {
	case time.Time:
		return t.Format(f.format)
	case error:
		return fmt.Sprintf("%%!(%s)", t)
	}

	return fmt.Sprintf("%"+f.format, value)
}

func (f *formatExpr) WriteTo(w io.StringWriter) {
	w.WriteString("%(")
	w.WriteString(f.name)
	if f.format != "" && f.format != "v" {
		w.WriteString(":")
		w.WriteString(f.format)
	}
	w.WriteString(")")
}

func (p *Pattern) Expand(expand Expander) string {
	var b strings.Builder
	Fprint(&b, p, expand)
	return b.String()
}

func (p *Pattern) String() string {
	var b bytes.Buffer
	for _, e := range p.exprs {
		e.WriteTo(&b)
	}
	return b.String()
}

func compilePatternCore(content []byte, pat *regexp.Regexp) *Pattern {
	allIndexes := pat.FindAllSubmatchIndex(content, -1)
	result := []expr{}

	var index int
	for _, loc := range allIndexes {
		if index < loc[0] {
			result = append(result, newLiteral(content[index:loc[0]]))
		}
		key := content[loc[2]:loc[3]]

		if m, ok := meta[string(key)]; ok {
			result = append(result, m.exprs...)
		} else {
			result = append(result, newExpr(key))
		}
		index = loc[1]
	}
	if index < len(content) {
		result = append(result, newLiteral(content[index:]))
	}

	return &Pattern{
		result,
	}
}

func newLiteral(token []byte) expr {
	t := string(token)
	// Handle escape sequences
	if s, err := strconv.Unquote(`"` + t + `"`); err == nil {
		t = s
	}
	return &literal{t}
}

func newExpr(token []byte) expr {
	nameAndFormat := strings.SplitN(string(token), ":", 2)
	name := nameAndFormat[0]
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
