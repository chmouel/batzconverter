package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWithoutFile(t *testing.T) {
	t.Setenv("BATZ_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DateFormat != "%c" || len(cfg.Timezones) != 5 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestLoadTOMLPreservesOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := `date_format = "%Y-%m-%d %H:%M %Z"
date_format_padding = 25
screenshot_theme = "dark"

[[timezone]]
name = "UTC"
zone = "UTC"

[[timezone]]
name = "Paris"
zone = "Europe/Paris"
emoji = "FR"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BATZ_CONFIG_FILE", path)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Timezones) != 2 || cfg.Timezones[0].Name != "UTC" || cfg.Timezones[1].Name != "Paris" {
		t.Fatalf("timezone order = %#v", cfg.Timezones)
	}
	if cfg.ScreenshotTheme != "dark" || cfg.DateFormatPadding != 25 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoadRejectsDuplicateAndInvalidZones(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate": `[[timezone]]
name = "UTC"
zone = "UTC"
[[timezone]]
name = "UTC"
zone = "Europe/Paris"
`,
		"invalid": `[[timezone]]
name = "Bad"
zone = "Not/A_Zone"
`,
		"unknown key": `date_formt = "%c"
[[timezone]]
name = "UTC"
zone = "UTC"
`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("BATZ_CONFIG_FILE", path)
			if _, err := Load(); err == nil {
				t.Fatal("Load unexpectedly succeeded")
			}
		})
	}
}

func TestScreenshotThemeEnvironmentIsFallback(t *testing.T) {
	t.Setenv("BATZ_SCREENSHOT_THEME", "dark")
	t.Setenv("BATZ_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ScreenshotTheme != "dark" {
		t.Fatalf("theme = %q", cfg.ScreenshotTheme)
	}
}
