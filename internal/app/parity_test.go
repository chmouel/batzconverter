package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chmouel/batzconverter/internal/command"
	"github.com/chmouel/batzconverter/internal/output"
)

func TestBashJSONParity(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the system Bash on macOS is older than the script requirement")
	}
	if _, err := exec.LookPath("date"); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	goConfig := filepath.Join(dir, "config.toml")
	bashConfig := filepath.Join(dir, "config.sh")
	if err := os.WriteFile(goConfig, []byte(`date_format = "%Y-%m-%dT%H:%M:%S%z"
screenshot_theme = "light"
[[timezone]]
name = "Paris"
zone = "Europe/Paris"
[[timezone]]
name = "NewYork"
zone = "America/New_York"
[[timezone]]
name = "UTC"
zone = "UTC"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bashConfig, []byte(`TIME_ZONES=(
  ["Paris"]="Europe/Paris"
  ["NewYork"]="America/New_York"
  ["UTC"]="UTC"
)
DATE_FORMAT="%Y-%m-%dT%H:%M:%S%z"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	expressions := [][]string{
		{"10h00", "tomorrow"},
		{"11:00", "next", "Thursday"},
		{"09:30", "yesterday"},
		{"19h00", "Monday", "17", "June", "2027"},
		{"02:30", "25", "October", "2026"},
	}
	for _, expression := range expressions {
		t.Run(expression[0]+"_"+expression[len(expression)-1], func(t *testing.T) {
			t.Setenv("BATZ_CONFIG_FILE", goConfig)
			t.Setenv("TZ", "UTC")
			t.Setenv("LANG", "C")
			t.Setenv("LC_ALL", "C")
			t.Setenv("NO_COLOR", "")
			t.Setenv("BATZ_SCREENSHOT_THEME", "")
			var goOutput bytes.Buffer
			args := append([]string{"-j", "-t", "UTC"}, expression...)
			if err := Run(context.Background(), command.OSRunner{}, args, &goOutput, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}

			script := filepath.Join("..", "..", "batz.sh")
			cmd := exec.Command("bash", append([]string{script, "-j", "-t", "UTC"}, expression...)...)
			cmd.Env = cleanEnvironment(map[string]string{
				"BATZ_CONFIG_FILE": bashConfig, "LANG": "C", "LC_ALL": "C", "TZ": "UTC",
			}, "NO_COLOR", "BATZ_SCREENSHOT_THEME")
			bashOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("batz.sh failed: %v\n%s", err, bashOutput)
			}
			goItems, bashItems := decodeItems(t, goOutput.Bytes()), decodeItems(t, bashOutput)
			if len(goItems) != len(bashItems) {
				t.Fatalf("item count differs: Go=%d Bash=%d", len(goItems), len(bashItems))
			}
			for title, value := range bashItems {
				if goItems[title] != value {
					t.Errorf("%s differs: Go=%q Bash=%q", title, goItems[title], value)
				}
			}
		})
	}
}

func TestBashAndGoCurrentTimeBounds(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the system Bash on macOS is older than the script requirement")
	}
	dir := t.TempDir()
	goConfig := filepath.Join(dir, "current.toml")
	bashConfig := filepath.Join(dir, "current.sh")
	if err := os.WriteFile(goConfig, []byte("date_format = \"%s\"\n[[timezone]]\nname = \"UTC\"\nzone = \"UTC\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bashConfig, []byte("TIME_ZONES=([\"UTC\"]=\"UTC\")\nDATE_FORMAT=\"%s\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BATZ_CONFIG_FILE", goConfig)
	t.Setenv("TZ", "UTC")
	t.Setenv("LC_ALL", "C")
	var goOutput bytes.Buffer
	before := time.Now().Unix()
	if err := Run(context.Background(), command.OSRunner{}, []string{"-j"}, &goOutput, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	assertEpochBounds(t, decodeItems(t, goOutput.Bytes()), before, time.Now().Unix())

	cmd := exec.Command("bash", filepath.Join("..", "..", "batz.sh"), "-j")
	cmd.Env = cleanEnvironment(map[string]string{
		"BATZ_CONFIG_FILE": bashConfig, "LANG": "C", "LC_ALL": "C", "TZ": "UTC",
	}, "NO_COLOR", "BATZ_SCREENSHOT_THEME")
	before = time.Now().Unix()
	bashOutput, err := cmd.CombinedOutput()
	after := time.Now().Unix()
	if err != nil {
		t.Fatalf("batz.sh failed: %v\n%s", err, bashOutput)
	}
	assertEpochBounds(t, decodeItems(t, bashOutput), before, after)
}

func decodeItems(t *testing.T, data []byte) map[string]string {
	t.Helper()
	var document output.Alfred
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, data)
	}
	result := make(map[string]string, len(document.Items))
	for _, item := range document.Items {
		result[item.Title] = item.Arg
	}
	return result
}

func cleanEnvironment(values map[string]string, remove ...string) []string {
	replaced := make(map[string]struct{}, len(values)+len(remove))
	for key := range values {
		replaced[key] = struct{}{}
	}
	for _, key := range remove {
		replaced[key] = struct{}{}
	}
	environment := make([]string, 0, len(os.Environ())+len(values))
	for _, item := range os.Environ() {
		key, _, found := strings.Cut(item, "=")
		if _, skip := replaced[key]; found && skip {
			continue
		}
		environment = append(environment, item)
	}
	for key, value := range values {
		environment = append(environment, key+"="+value)
	}
	return environment
}

func assertEpochBounds(t *testing.T, items map[string]string, before, after int64) {
	t.Helper()
	for name, value := range items {
		epoch, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			t.Fatalf("%s epoch %q: %v", name, value, err)
		}
		if epoch < before || epoch > after {
			t.Errorf("%s epoch %d outside [%d, %d]", name, epoch, before, after)
		}
	}
}
