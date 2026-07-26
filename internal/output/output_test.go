package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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

func TestTerminalAndCSV(t *testing.T) {
	rows := []Row{{Name: "UTC", Value: "2026-01-02", Emoji: "🌍 ", IsBase: true}}
	var terminal bytes.Buffer
	if err := Terminal(&terminal, rows, 0, false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(terminal.String(), "UTC") || !strings.Contains(terminal.String(), "🌍") {
		t.Fatalf("terminal output = %q", terminal.String())
	}
	csv, err := CSV(rows, true)
	if err != nil {
		t.Fatal(err)
	}
	if csv != "Timezone,Date\n🌍 UTC,2026-01-02\n" {
		t.Fatalf("CSV = %q", csv)
	}
}
