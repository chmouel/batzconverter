package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestJSONEscapesAndOmitsEmptyIcon(t *testing.T) {
	rows := []Row{{Name: `A"B`, Value: "<date>"}, {Name: "Paris", Value: "now", Icon: "/tmp/icon.png"}}
	var buffer bytes.Buffer
	if err := JSON(&buffer, rows); err != nil {
		t.Fatal(err)
	}
	var document Alfred
	if err := json.Unmarshal(buffer.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Items[0].Title != `A"B` || document.Items[0].Icon != nil {
		t.Fatalf("unexpected first item: %#v", document.Items[0])
	}
	if document.Items[1].Icon == nil || document.Items[1].Icon.Path != "/tmp/icon.png" {
		t.Fatalf("unexpected second item: %#v", document.Items[1])
	}
}

func TestTerminalPlainAlignment(t *testing.T) {
	rows := mixedRows()
	var output bytes.Buffer
	if err := Terminal(&output, rows, false, false, true); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b") || strings.ContainsAny(output.String(), "╭╮╰╯│") {
		t.Fatalf("plain output contains terminal styling: %q", output.String())
	}
	assertColumnAlignment(t, output.String(), true)
	if !strings.Contains(output.String(), "🏠") {
		t.Fatalf("plain output is missing local marker: %q", output.String())
	}
	if strings.HasSuffix(strings.TrimSuffix(output.String(), "\n"), " ") {
		t.Fatalf("plain output has trailing whitespace: %q", output.String())
	}
}

func TestTerminalRoundedAlignment(t *testing.T) {
	var output bytes.Buffer
	if err := Terminal(&output, mixedRows(), true, true, true); err != nil {
		t.Fatal(err)
	}
	for _, border := range []string{"╭", "╮", "╰", "╯", "│"} {
		if !strings.Contains(output.String(), border) {
			t.Fatalf("rounded output is missing %q:\n%s", border, output.String())
		}
	}
	assertColumnAlignment(t, output.String(), true)
}

func TestTerminalWithoutEmojiHasTwoColumns(t *testing.T) {
	var output bytes.Buffer
	if err := Terminal(&output, mixedRows(), false, false, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "🇮🇳") || strings.Contains(output.String(), "🏠") {
		t.Fatalf("emoji-disabled output contains emoji: %q", output.String())
	}
	assertColumnAlignment(t, output.String(), false)
}

func TestTerminalEmptyRows(t *testing.T) {
	var output bytes.Buffer
	if err := Terminal(&output, nil, true, true, true); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("empty output = %q", output.String())
	}
}

func mixedRows() []Row {
	return []Row{
		{Name: "Bangalore", Value: "Sun 26 Jul 2026 14:00:00 IST", Emoji: "🇮🇳 ", IsLocal: true},
		{Name: "Brisbane", Value: "Sun 26 Jul 2026 18:30:00 AEST", Emoji: "🇦🇺 "},
		{Name: "Paris", Value: "Sun 26 Jul 2026 10:30:00 CEST", Emoji: "🇫🇷 "},
		{Name: "Boston", Value: "Sun 26 Jul 2026 04:30:00 EDT", Emoji: "🇺🇸 "},
		{Name: "California", Value: "Sun 26 Jul 2026 01:30:00 PDT", Emoji: "🐻 "},
		{Name: "Cairo", Value: "Sun 26 Jul 2026 11:30:00 EEST", Emoji: "🇪🇬 "},
	}
}

func assertColumnAlignment(t *testing.T, rendered string, expectEmoji bool) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
	datePosition := -1
	emojiPosition := -1
	rows := 0
	emojis := []string{"🇮🇳", "🇦🇺", "🇫🇷", "🇺🇸", "🐻", "🇪🇬"}
	for _, line := range lines {
		plain := ansi.Strip(line)
		dateIndex := strings.Index(plain, "Sun 26 Jul")
		if dateIndex < 0 {
			continue
		}
		position := ansi.StringWidth(plain[:dateIndex])
		if datePosition == -1 {
			datePosition = position
		} else if datePosition != position {
			t.Fatalf("date starts at columns %d and %d:\n%s", datePosition, position, rendered)
		}
		if expectEmoji {
			emojiIndex := strings.Index(plain, emojis[rows])
			if emojiIndex < 0 {
				t.Fatalf("%q is absent from row %q", emojis[rows], plain)
			}
			position = ansi.StringWidth(plain[:emojiIndex])
			if emojiPosition == -1 {
				emojiPosition = position
			} else if emojiPosition != position {
				t.Fatalf("emoji starts at columns %d and %d:\n%s", emojiPosition, position, rendered)
			}
		}
		rows++
	}
	if rows != len(mixedRows()) {
		t.Fatalf("checked %d rows, want %d:\n%s", rows, len(mixedRows()), rendered)
	}
}
