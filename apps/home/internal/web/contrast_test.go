package web

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// color is an sRGB color with an alpha channel, components from 0 to 1.
type color struct{ r, g, b, a float64 }

var (
	hexColor = regexp.MustCompile(`^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$`)
	rgbColor = regexp.MustCompile(`^rgb\((\d+) (\d+) (\d+) / ([0-9.]+)\)$`)
	cssVar   = regexp.MustCompile(`--([a-z0-9-]+):\s*([^;]+);`)
)

func parseColor(t *testing.T, s string) color {
	t.Helper()
	if m := hexColor.FindStringSubmatch(s); m != nil {
		var c [3]float64
		for i := range c {
			n, _ := strconv.ParseUint(m[i+1], 16, 8)
			c[i] = float64(n) / 255
		}
		return color{c[0], c[1], c[2], 1}
	}
	if m := rgbColor.FindStringSubmatch(s); m != nil {
		var c [4]float64
		for i := range c {
			c[i], _ = strconv.ParseFloat(m[i+1], 64)
		}
		return color{c[0] / 255, c[1] / 255, c[2] / 255, c[3]}
	}
	t.Fatalf("cannot parse color %q (use #rrggbb or rgb(r g b / a))", s)
	return color{}
}

// over composites a translucent color over an opaque one.
func (c color) over(bg color) color {
	return color{
		c.r*c.a + bg.r*(1-c.a),
		c.g*c.a + bg.g*(1-c.a),
		c.b*c.a + bg.b*(1-c.a),
		1,
	}
}

// luminance is the relative luminance of WCAG 2.
func (c color) luminance() float64 {
	lin := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.r) + 0.7152*lin(c.g) + 0.0722*lin(c.b)
}

func contrast(a, b color) float64 {
	la, lb := a.luminance(), b.luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// themeVars returns the variables of the light and dark :root blocks of home.css.
func themeVars(t *testing.T) (light, dark map[string]string) {
	t.Helper()
	data, err := os.ReadFile("static/assets/home.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	before, after, ok := strings.Cut(css, "@media (prefers-color-scheme: dark)")
	if !ok {
		t.Fatal("no dark mode block in home.css")
	}
	block := func(s string) map[string]string {
		start := strings.Index(s, ":root {")
		if start < 0 {
			t.Fatal("no :root block in home.css")
		}
		end := strings.Index(s[start:], "}")
		vars := map[string]string{}
		for _, m := range cssVar.FindAllStringSubmatch(s[start:start+end], -1) {
			vars[m[1]] = strings.TrimSpace(m[2])
		}
		return vars
	}
	return block(before), block(after)
}

// CA17 (NF1, NF4): every text color reaches AA (4.5:1) on every color the page
// background goes through, and on the cards (at rest and hovered) laid over it.
func TestContrastAA(t *testing.T) {
	light, dark := themeVars(t)
	for mode, vars := range map[string]map[string]string{"light": light, "dark": dark} {
		get := func(name string) color {
			v, ok := vars[name]
			if !ok {
				t.Fatalf("%s: --%s is not defined", mode, name)
			}
			return parseColor(t, v)
		}
		backgrounds := map[string]color{"bg-0": get("bg-0"), "bg-1": get("bg-1")}
		for name, bg := range backgrounds {
			for _, card := range []string{"card", "card-hover"} {
				backgrounds[card+" over "+name] = get(card).over(bg)
			}
		}
		// Text on the page: title, paragraph, section title. On the cards: name,
		// description, domain (and the link back of the 404 page).
		pairs := map[string][]string{
			"text":        {"bg-0", "bg-1", "card over bg-0", "card over bg-1", "card-hover over bg-0", "card-hover over bg-1"},
			"muted":       {"bg-0", "bg-1", "card over bg-0", "card over bg-1", "card-hover over bg-0", "card-hover over bg-1"},
			"accent-text": {"bg-0", "bg-1", "card over bg-0", "card over bg-1", "card-hover over bg-0", "card-hover over bg-1"},
		}
		for text, bgs := range pairs {
			for _, bg := range bgs {
				if r := contrast(get(text), backgrounds[bg]); r < 4.5 {
					t.Errorf("%s: --%s on %s: contrast %.2f, want ≥ 4.5", mode, text, bg, r)
				}
			}
		}
	}
}

func TestContrastHelper(t *testing.T) {
	black, white := parseColor(t, "#000000"), parseColor(t, "#ffffff")
	if r := contrast(black, white); math.Abs(r-21) > 0.01 {
		t.Errorf("contrast(black, white) = %.2f, want 21", r)
	}
	half := parseColor(t, "rgb(255 255 255 / 0.5)").over(black)
	if math.Abs(half.r-0.5) > 0.001 {
		t.Errorf("50%% white over black = %v", half)
	}
}
