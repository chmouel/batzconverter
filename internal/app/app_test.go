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
	"time"
)

type fakeRunner struct {
	paths map[string]string
}

type selectionRunner struct {
	stdout []byte
	stderr []byte
	err    error
}

func (selectionRunner) LookPath(name string) (string, error) {
	return "/bin/" + name, nil
}

func (runner selectionRunner) Run(context.Context, string, []string, []string, io.Reader) ([]byte, []byte, error) {
	return runner.stdout, runner.stderr, runner.err
}

type exitError int

func (code exitError) Error() string { return "exit" }
func (code exitError) ExitCode() int { return int(code) }

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

func TestShellQuote(t *testing.T) {
	if got, want := shellQuote("/tmp/a b/c'd"), `'/tmp/a b/c'"'"'d'`; got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}

func TestNextTransition(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	data := `Europe/Paris  Sun Mar 29 00:59:59 2026 UT = Sun Mar 29 01:59:59 2026 CET isdst=0 gmtoff=3600
Europe/Paris  Sun Mar 29 01:00:00 2026 UT = Sun Mar 29 03:00:00 2026 CEST isdst=1 gmtoff=7200
`
	got, err := nextTransition(data, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "01:00:00 2026 UT") {
		t.Fatalf("next transition = %q", got)
	}
}

func TestNextTransitionRejectsMalformedPairs(t *testing.T) {
	_, err := nextTransition("Europe/Paris  bad\n", time.Now())
	if err == nil {
		t.Fatal("nextTransition unexpectedly succeeded")
	}
}

func TestIsTerminalRejectsRegularDevices(t *testing.T) {
	file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if isTerminal(file) {
		t.Fatal("os.DevNull detected as a terminal")
	}
}

func TestSelectZonesCancellation(t *testing.T) {
	for _, runner := range []selectionRunner{{}, {err: exitError(1)}, {err: exitError(130)}} {
		_, err := selectZones(context.Background(), runner)
		if !errors.Is(err, ErrNoTimezoneSelected) {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestSelectZonesRejectsUnofferedAndDuplicateOutput(t *testing.T) {
	for _, output := range [][]byte{[]byte("Not/A_Zone\n"), []byte("UTC\nUTC\n"), []byte("UTC\r\n")} {
		_, err := selectZones(context.Background(), selectionRunner{stdout: output})
		var selectionErr *SelectionError
		if !errors.As(err, &selectionErr) {
			t.Fatalf("output %q error = %v", output, err)
		}
	}
}

func TestSelectZonesDiscardsPartialOutputOnFailure(t *testing.T) {
	_, err := selectZones(context.Background(), selectionRunner{
		stdout: []byte("UTC\n"),
		stderr: []byte("boom\n"),
		err:    exitError(2),
	})
	var selectionErr *SelectionError
	if !errors.As(err, &selectionErr) || selectionErr.ExitCode != 2 || selectionErr.Stderr != "boom" {
		t.Fatalf("error = %#v", err)
	}
}

func TestSelectZonesContextCancellationWins(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := selectZones(ctx, selectionRunner{err: exitError(2)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
