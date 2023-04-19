package expr

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	patternRegexp = regexp.MustCompile(`%\((.+?)\)`)
)

type Pattern struct {
	exprs []expr
}

type expr interface {
	Format(expand func(string) any) any
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

func ExpandMap(m map[string]any) func(string) any {
	return func(k string) any {
		v, ok := m[k]
		if !ok {
			return UnknownToken(k)
		}
		return v
	}
}

func UnknownToken(tok string) error {
	return fmt.Errorf("unknown: %s", tok)
}

func (l *literal) Format(expand func(string) any) any {
	return l.text
}

func (f *formatExpr) Format(expand func(string) any) any {
	value := expand(f.name)
	switch t := value.(type) {
	case time.Time:
		return t.Format(f.format)
	case error:
		return fmt.Sprintf("%%!(%s)", t)
	}

	return fmt.Sprintf("%"+f.format, value)
}

func (f *Pattern) Fprint(w io.Writer, expand func(string) any) {
	for _, item := range f.exprs {
		fmt.Fprint(w, item.Format(expand))
	}
}

func (f *Pattern) Expand(expand func(string) any) string {
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
