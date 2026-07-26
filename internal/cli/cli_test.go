package cli

import (
	"strings"
	"testing"
)

func TestParseExpressionAndFlags(t *testing.T) {
	options, err := Parse([]string{"+UTC", "-t", "Europe/Paris", "-j", "10h3", "tomorrow"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Expression != "10:03 tomorrow" {
		t.Fatalf("expression = %q", options.Expression)
	}
	if options.BaseZone != "Europe/Paris" || !options.JSON {
		t.Fatalf("unexpected options: %#v", options)
	}
	if len(options.AddedZones) != 1 || options.AddedZones[0] != "UTC" {
		t.Fatalf("added zones = %#v", options.AddedZones)
	}
}

func TestParseRejectsInvalidCombinations(t *testing.T) {
	for _, args := range [][]string{{"-s", "-j"}, {"tomorrow"}, {"-S", "blue"}, {"-x"}} {
		if _, err := Parse(args); err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", args)
		}
	}
}

func TestParseKeepsExpressionWords(t *testing.T) {
	options, err := Parse(strings.Fields("19h00 Monday 17 June 2027"))
	if err != nil {
		t.Fatal(err)
	}
	if options.Expression != "19:00 Monday 17 June 2027" {
		t.Fatalf("expression = %q", options.Expression)
	}
}

func TestParseClusteredAndAttachedOptions(t *testing.T) {
	options, err := Parse([]string{"-jn", "-tUTC", "-Sdark", "10h00"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.JSON || !options.NoColor || options.BaseZone != "UTC" || options.ScreenshotTheme != "dark" {
		t.Fatalf("unexpected options: %#v", options)
	}
}
