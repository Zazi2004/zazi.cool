package highlight

import (
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"zazi.cool/pastebin/internal/paste"
)

// text joins the spans of each line, dropping the classes.
func text(lines []Line) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		var b strings.Builder
		for _, s := range l {
			b.WriteString(s.Text)
		}
		out = append(out, b.String())
	}
	return out
}

func TestLinesKeepText(t *testing.T) {
	tests := map[string][]string{
		"a":          {"a"},
		"a\n":        {"a"},
		"a\r\nb\r\n": {"a", "b"},
		"a\n\nb":     {"a", "", "b"},
		"\n\n":       {"", ""},
	}
	for _, lang := range []string{"plaintext", "go", "unknown"} {
		for in, want := range tests {
			got := text(Lines(in, lang))
			if strings.Join(got, "|") != strings.Join(want, "|") || len(got) != len(want) {
				t.Errorf("Lines(%q, %q) = %q, want %q", in, lang, got, want)
			}
		}
	}
}

func TestLinesHighlight(t *testing.T) {
	lines := Lines("package main\n// note\n", "go")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if first := lines[0][0]; first != (Span{Class: "hl-kn", Text: "package"}) {
		t.Errorf("first span = %+v, want the package keyword", first)
	}
	if got := lines[1][0].Class; got != "hl-c1" {
		t.Errorf("comment class = %q, want hl-c1", got)
	}
}

func TestPlainTextHasNoClass(t *testing.T) {
	for _, l := range Lines("func main() {}\n<b>", "plaintext") {
		for _, s := range l {
			if s.Class != "" {
				t.Errorf("plain text span %+v has a class", s)
			}
		}
	}
}

// Every language offered by the form must name a real chroma lexer, otherwise it
// would silently fall back to plain text.
func TestFormLanguagesHaveLexers(t *testing.T) {
	for _, l := range paste.Languages {
		if lexers.Get(l.Key) == nil {
			t.Errorf("no chroma lexer named %q", l.Key)
		}
	}
}

func TestCSS(t *testing.T) {
	for _, name := range []string{lightStyle, darkStyle} {
		if styles.Get(name).Name != name {
			t.Errorf("chroma has no style %q", name)
		}
	}
	css, err := CSS()
	if err != nil {
		t.Fatal(err)
	}
	s := string(css)
	for _, want := range []string{".hl-chroma .hl-k ", ".hl-chroma .hl-c1 ", "@media (prefers-color-scheme: dark) {"} {
		if !strings.Contains(s, want) {
			t.Errorf("CSS lacks %q", want)
		}
	}
}
