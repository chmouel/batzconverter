package dateengine

import (
	"context"
	"os"
	"testing"

	"github.com/chmouel/batzconverter/internal/command"
)

func TestGNUDateDocumentedExpressions(t *testing.T) {
	engine, err := Discover(context.Background(), command.OSRunner{})
	if err != nil {
		t.Skip(err)
	}
	tests := []string{
		"10:30",
		"10:00 tomorrow",
		"10:30 next week",
		"11:00 next Thursday",
		"09:30 yesterday",
		"19:00 Monday 17 June 2027",
	}
	for _, expression := range tests {
		t.Run(expression, func(t *testing.T) {
			value, err := engine.Format(context.Background(), "America/New_York", "Europe/Paris", expression, "%Y-%m-%dT%H:%M:%S%z")
			if err != nil {
				t.Fatal(err)
			}
			if value == "" {
				t.Fatal("empty value")
			}
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
