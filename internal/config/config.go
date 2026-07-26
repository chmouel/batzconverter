package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Timezone struct {
	Name  string `toml:"name"`
	Zone  string `toml:"zone"`
	Emoji string `toml:"emoji"`
	Icon  string `toml:"icon"`
}

type Config struct {
	DateFormat           string     `toml:"date_format"`
	DateFormatPadding    int        `toml:"date_format_padding"`
	DefaultTimezoneEmoji string     `toml:"default_timezone_emoji"`
	UseGum               bool       `toml:"use_gum"`
	ScreenshotTheme      string     `toml:"screenshot_theme"`
	Timezones            []Timezone `toml:"timezone"`
}

func Defaults() Config {
	theme := os.Getenv("BATZ_SCREENSHOT_THEME")
	if theme == "" {
		theme = "light"
	}
	return Config{
		DateFormat:           "%c",
		DefaultTimezoneEmoji: "🌍 ",
		ScreenshotTheme:      theme,
		Timezones: []Timezone{
			{Name: "Bangalore", Zone: "Asia/Calcutta", Emoji: "🇮🇳 "},
			{Name: "Brisbane", Zone: "Australia/Brisbane", Emoji: "🇦🇺 "},
			{Name: "Paris", Zone: "Europe/Paris", Emoji: "🇫🇷 "},
			{Name: "Boston", Zone: "America/New_York", Emoji: "🇺🇸 "},
			{Name: "California", Zone: "America/Los_Angeles", Emoji: "🐻 "},
		},
	}
}

func Path() (string, error) {
	if path := os.Getenv("BATZ_CONFIG_FILE"); path != "" {
		return path, nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(root, "batz", "config.toml"), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	cfg := Defaults()
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if os.Getenv("BATZ_CONFIG_FILE") != "" {
			return Config{}, fmt.Errorf("configured file %q does not exist", path)
		}
		return cfg, validate(cfg)
	} else if err != nil {
		return Config{}, fmt.Errorf("inspect config %q: %w", path, err)
	}

	metadata, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", path, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		return Config{}, fmt.Errorf("invalid config %q: unknown keys: %s", path, strings.Join(keys, ", "))
	}
	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.DateFormat == "" {
		return errors.New("date_format must not be empty")
	}
	if cfg.DateFormatPadding < 0 {
		return errors.New("date_format_padding must not be negative")
	}
	if cfg.ScreenshotTheme != "light" && cfg.ScreenshotTheme != "dark" {
		return errors.New("screenshot_theme must be light or dark")
	}
	if len(cfg.Timezones) == 0 {
		return errors.New("at least one [[timezone]] is required")
	}
	names := make(map[string]struct{}, len(cfg.Timezones))
	for i, zone := range cfg.Timezones {
		if strings.TrimSpace(zone.Name) == "" || strings.TrimSpace(zone.Zone) == "" {
			return fmt.Errorf("timezone %d requires name and zone", i+1)
		}
		if _, found := names[zone.Name]; found {
			return fmt.Errorf("duplicate timezone name %q", zone.Name)
		}
		names[zone.Name] = struct{}{}
		if _, err := time.LoadLocation(zone.Zone); err != nil {
			return fmt.Errorf("timezone %q has invalid zone %q: %w", zone.Name, zone.Zone, err)
		}
	}
	return nil
}
