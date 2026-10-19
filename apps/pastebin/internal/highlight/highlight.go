// Package highlight colours pastes with chroma (docs/specs/pastebin.md, F3, D4).
//
// It only splits the content into classed spans: the HTML is written by html/template,
// which escapes every span, and the colours live in a style sheet served as a file, so
// the strict CSP (NF1) needs neither inline styles nor unescaped markup.
package highlight

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// classPrefix keeps chroma's class names (".line", ".k"…) apart from the app's own.
const classPrefix = "hl-"

// PreClass must be set on the element wrapping the lines: the style sheet only colours
// spans inside it.
const PreClass = classPrefix + "chroma"

// Light and dark styles of the style sheet, following the browser's colour scheme.
const (
	lightStyle = "github"
	darkStyle  = "github-dark"
)

// Span is a piece of a line with the CSS class of its token type. Class is empty for
// plain text.
type Span struct {
	Class string
	Text  string
}

// Line is one line of a paste, without its line break.
type Line []Span

// Lines splits content into highlighted lines. language is a chroma lexer name; an
// unknown one, or a lexer failure, gives plain text. A final line break does not open
// an extra empty line, and Windows line endings lose their "\r".
func Lines(content, language string) []Line {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, content+"\n")
	if err != nil {
		return plainLines(content)
	}

	var out []Line
	for _, tokens := range chroma.SplitTokensIntoLines(it.Tokens()) {
		line := Line{}
		for _, t := range tokens {
			text := strings.TrimSuffix(t.Value, "\n")
			if text != "" {
				line = append(line, Span{Class: class(t.Type), Text: text})
			}
		}
		out = append(out, line)
	}
	return out
}

func plainLines(content string) []Line {
	var out []Line
	for _, l := range strings.Split(content, "\n") {
		out = append(out, Line{{Text: l}})
	}
	return out
}

// class mirrors chroma's HTML formatter: a token type without a class of its own takes
// the class of its parent type (a "KeywordReserved" is coloured as a "Keyword").
func class(t chroma.TokenType) string {
	for ; t != 0; t = t.Parent() {
		if cls, ok := chroma.StandardTypes[t]; ok {
			if cls == "" {
				return ""
			}
			return classPrefix + cls
		}
	}
	return ""
}

// CSS returns the style sheet of the token classes: the light style, then the dark one
// when the browser prefers a dark colour scheme.
func CSS() ([]byte, error) {
	f := chromahtml.New(chromahtml.WithClasses(true), chromahtml.ClassPrefix(classPrefix))
	var b bytes.Buffer
	if err := f.WriteCSS(&b, styles.Get(lightStyle)); err != nil {
		return nil, fmt.Errorf("write %s style: %w", lightStyle, err)
	}
	b.WriteString("@media (prefers-color-scheme: dark) {\n")
	if err := f.WriteCSS(&b, styles.Get(darkStyle)); err != nil {
		return nil, fmt.Errorf("write %s style: %w", darkStyle, err)
	}
	b.WriteString("}\n")
	return b.Bytes(), nil
}
