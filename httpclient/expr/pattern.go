package expr

import (
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli-http/internal/build"
)

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
)

type Pattern struct {
	exprs []expr
}

type Expander func(string) any

type expr interface {
	Format(expand Expander) any
}

type formatExpr struct {
	name   string
	format string
}

type literal struct {
	text string
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

func UnknownToken(tok string) error {
	return fmt.Errorf("unknown: %s", tok)
}

func (l *literal) Format(expand Expander) any {
	return l.text
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

func (f *Pattern) Fprint(w io.Writer, expand Expander) {
	for _, item := range f.exprs {
		fmt.Fprint(w, item.Format(expand))
	}
}

func (f *Pattern) Expand(expand Expander) string {
	var b strings.Builder
	f.Fprint(&b, expand)
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
		result = append(result, newExpr(key))
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
)
