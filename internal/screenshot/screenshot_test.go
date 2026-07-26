package screenshot

import (
	"strings"
	"testing"

	"github.com/chmouel/batzconverter/internal/output"
)

func TestSVGEscapesValuesAndIncludesTheme(t *testing.T) {
	rows := []Row{{
		Row:  output.Row{Name: "A&B", Emoji: "🌍", IsBase: true},
		Date: "Mon <today>", Time: "10:00", Abbr: "UTC", Country: "GB",
	}}
	svg := string(SVG(rows, "Header & now", "dark", true, true))
	for _, expected := range []string{"A&amp;B", "Mon &lt;today&gt;", "Header &amp; now", "#090b18", "BASE"} {
		if !strings.Contains(svg, expected) {
			t.Fatalf("SVG does not contain %q", expected)
		}
	}
	if strings.Contains(svg, "A&B") {
		t.Fatal("SVG contains unescaped value")
	}
}

func TestSVGUsesCountryFallbackWithoutEmoji(t *testing.T) {
	rows := []Row{{
		Row:     output.Row{Name: "Paris", Emoji: "🇫🇷"},
		Date:    "Monday",
		Time:    "10:00",
		Abbr:    "CET",
		Country: "FR",
	}}
	svg := string(SVG(rows, "Header", "light", true, false))
	if !strings.Contains(svg, ">FR</text>") || strings.Contains(svg, "🇫🇷") {
		t.Fatalf("unexpected fallback SVG: %s", svg)
	}
}
