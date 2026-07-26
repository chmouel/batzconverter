package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	paths map[string]string
}

func (f fakeRunner) LookPath(name string) (string, error) {
	path, ok := f.paths[name]
	if !ok {
		return "", errors.New("not found")
	}
	return path, nil
}

func (f fakeRunner) Run(_ context.Context, _ string, args, env []string, stdin io.Reader) ([]byte, []byte, error) {
	if len(args) == 1 && args[0] == "--version" {
		return []byte("date (GNU coreutils) 9.8\n"), nil, nil
	}
	if len(args) > 0 && strings.HasPrefix(args[0], "--prompt=") {
		return []byte("Asia/Tokyo\n"), nil, nil
	}
	if len(args) > 0 && args[0] == "table" {
		data, _ := io.ReadAll(stdin)
		return append([]byte("gum:\n"), data...), nil, nil
	}
	format := args[len(args)-1]
	zone := ""
	for _, value := range env {
		if strings.HasPrefix(value, "TZ=") {
			zone = strings.TrimPrefix(value, "TZ=")
		}
	}
	return []byte(zone + " " + format + "\n"), nil, nil
}

func writeConfig(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	data := `date_format = "%Y-%m-%d %H:%M %Z"
screenshot_theme = "light"

[[timezone]]
name = "Paris"
zone = "Europe/Paris"
emoji = "FR"

[[timezone]]
name = "UTC"
zone = "UTC"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BATZ_CONFIG_FILE", path)
	t.Setenv("TZ", "UTC")
}

func TestRunJSONResolvesBaseAlias(t *testing.T) {
	writeConfig(t)
	var stdout, stderr bytes.Buffer
	runner := fakeRunner{paths: map[string]string{"gdate": "/opt/gdate"}}
	err := Run(context.Background(), runner, []string{"-j", "-t", "Paris", "10h00", "tomorrow"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"title": "Paris"`) || !strings.Contains(stdout.String(), `Europe/Paris +%Y-%m-%d`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunAddsExplicitBaseZone(t *testing.T) {
	writeConfig(t)
	var stdout bytes.Buffer
	runner := fakeRunner{paths: map[string]string{"date": "/bin/date"}}
	err := Run(context.Background(), runner, []string{"-j", "-t", "Asia/Tokyo", "10h00"}, &stdout, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"title": "Asia/Tokyo"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunGum(t *testing.T) {
	writeConfig(t)
	var stdout bytes.Buffer
	runner := fakeRunner{paths: map[string]string{"date": "/bin/date", "gum": "/bin/gum"}}
	if err := Run(context.Background(), runner, []string{"-g"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout.String(), "gum:\nTimezone,Date\n") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFZFAddsSelection(t *testing.T) {
	writeConfig(t)
	var stdout bytes.Buffer
	runner := fakeRunner{paths: map[string]string{
		"date": "/bin/date", "fzf": "/bin/fzf", "zdump": "/bin/zdump",
	}}
	if err := Run(context.Background(), runner, []string{"-j", "-f"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{`"title": "Paris"`, `"title": "UTC"`, `"title": "Tokyo"`} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("stdout does not contain %s: %s", name, stdout.String())
		}
	}
}
