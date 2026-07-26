package dateengine

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestGNUDateDeterministicTimezoneAndDSTConversions(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	engine, err := Discover(context.Background(), command.OSRunner{})
	if err != nil {
		t.Skip(err)
	}
	tests := []struct {
		name       string
		targetZone string
		baseZone   string
		expression string
		want       string
		wantError  bool
	}{
		{
			name:       "timezone alias",
			targetZone: "UTC",
			baseZone:   "Asia/Calcutta",
			expression: "10:00 17 June 2027",
			want:       "2027-06-17T04:30:00+0000",
		},
		{
			name:       "European DST fold",
			targetZone: "Europe/Paris",
			baseZone:   "UTC",
			expression: "02:30 25 October 2026",
			want:       "2026-10-25T03:30:00+0100",
		},
		{
			name:       "New York DST fold",
			targetZone: "UTC",
			baseZone:   "America/New_York",
			expression: "01:30 1 November 2026",
			want:       "2026-11-01T05:30:00+0000",
		},
		{
			name:       "New York DST gap",
			targetZone: "UTC",
			baseZone:   "America/New_York",
			expression: "02:30 8 March 2026",
			wantError:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := engine.Format(
				context.Background(),
				test.targetZone,
				test.baseZone,
				test.expression,
				"%Y-%m-%dT%H:%M:%S%z",
			)
			if test.wantError {
				if err == nil || !strings.Contains(err.Error(), "could not parse") {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("Format() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGNUDateCurrentTimeBounds(t *testing.T) {
	engine, err := Discover(context.Background(), command.OSRunner{})
	if err != nil {
		t.Skip(err)
	}
	before := time.Now().Unix()
	value, err := engine.Now(context.Background(), "UTC", "%s")
	after := time.Now().Unix()
	if err != nil {
		t.Fatal(err)
	}
	epoch, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("parse epoch %q: %v", value, err)
	}
	if epoch < before || epoch > after {
		t.Fatalf("epoch %d outside [%d, %d]", epoch, before, after)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
