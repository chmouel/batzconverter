package command

import (
	"slices"
	"testing"
)

func TestMergeEnvironmentReplacesKeys(t *testing.T) {
	got := mergeEnvironment(
		[]string{"TZ=Europe/Paris", "PATH=/bin", "LANG=en_US"},
		[]string{"TZ=UTC", "LANG=C"},
	)
	want := []string{"PATH=/bin", "TZ=UTC", "LANG=C"}
	if !slices.Equal(got, want) {
		t.Fatalf("environment = %#v, want %#v", got, want)
	}
}
