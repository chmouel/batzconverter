package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

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
	}
	for _, expression := range expressions {
		t.Run(expression[0]+"_"+expression[len(expression)-1], func(t *testing.T) {
			t.Setenv("BATZ_CONFIG_FILE", goConfig)
			t.Setenv("TZ", "UTC")
			var goOutput bytes.Buffer
			args := append([]string{"-j", "-t", "UTC"}, expression...)
			if err := Run(context.Background(), command.OSRunner{}, args, &goOutput, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}

			script := filepath.Join("..", "..", "batz.sh")
			cmd := exec.Command("bash", append([]string{script, "-j", "-t", "UTC"}, expression...)...)
			cmd.Env = append(os.Environ(), "BATZ_CONFIG_FILE="+bashConfig, "TZ=UTC")
			bashOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("batz.sh failed: %v\n%s", err, bashOutput)
			}

			goItems := decodeItems(t, goOutput.Bytes())
			bashItems := decodeItems(t, bashOutput)
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
