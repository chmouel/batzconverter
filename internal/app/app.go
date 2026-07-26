package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/chmouel/batzconverter/internal/cli"
	"github.com/chmouel/batzconverter/internal/command"
	"github.com/chmouel/batzconverter/internal/config"
	"github.com/chmouel/batzconverter/internal/dateengine"
	"github.com/chmouel/batzconverter/internal/output"
	"github.com/chmouel/batzconverter/internal/screenshot"
	tzutil "github.com/chmouel/batzconverter/internal/timezone"
)

var (
	executablePath = os.Executable
	currentTime    = time.Now
)

var ErrNoTimezoneSelected = errors.New("no timezone selected")

type SelectionError struct {
	ExitCode int
	Stderr   string
}

func (err *SelectionError) Error() string {
	if err.Stderr == "" {
		return fmt.Sprintf("fzf selection failed with exit code %d", err.ExitCode)
	}
	return fmt.Sprintf("fzf selection failed with exit code %d: %s", err.ExitCode, err.Stderr)
}

const help = `batz - timezone converter

Usage:
  batz [flags] [+Area/Location ...] [TIME [GNU DATE EXPRESSION...]]

Examples:
  batz
  batz 10h30
  batz 10h00 tomorrow
  batz 19h00 Monday 17 June
  batz 10h30 next week
  batz -t UTC 11:00 next Thursday
  batz +America/Chicago +UTC 10h00 tomorrow

Flags:
  -t ZONE  use another timezone as the base
  -j       emit Alfred-compatible JSON
  -n       disable colors
  -C       force colors
  -E       disable emojis
  -f       select timezones with fzf
  -s       render and copy a screenshot
  -S THEME screenshot theme: light or dark
  -h       show help

The first date argument must be a time using h or :, followed by any expression
accepted by GNU date. The base zone defaults to the local IANA timezone; -t
accepts either a configured name or an IANA identifier.

Configuration:
  ~/.config/batz/config.toml
  BATZ_CONFIG_FILE overrides that path.

Requirements:
  GNU coreutils date (gdate on macOS).
  -f additionally requires fzf and zdump.
  Screenshot output uses native macOS tools or rsvg-convert/ImageMagick and
  wl-copy on Linux Wayland.
`

func Run(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "--preview-zone" {
		if len(args) != 2 {
			return fmt.Errorf("--preview-zone requires exactly one timezone")
		}
		return previewZone(ctx, runner, args[1], stdout)
	}
	options, err := cli.Parse(args)
	if err != nil {
		return err
	}
	if options.Help {
		_, err := io.WriteString(stdout, help)
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if options.ScreenshotTheme != "" {
		cfg.ScreenshotTheme = options.ScreenshotTheme
	}
	zones := append([]config.Timezone(nil), cfg.Timezones...)
	for _, zone := range options.AddedZones {
		if err := tzutil.Validate(zone); err != nil {
			return err
		}
		zones = upsert(zones, config.Timezone{Name: filepath.Base(zone), Zone: zone})
	}

	localZone, localErr := tzutil.Local()
	baseZone := options.BaseZone
	if baseZone == "" {
		if localErr != nil {
			return localErr
		}
		baseZone = localZone
	} else {
		resolved := false
		for _, zone := range zones {
			if zone.Name == baseZone {
				baseZone = zone.Zone
				resolved = true
				break
			}
		}
		if !resolved {
			if err := tzutil.Validate(baseZone); err != nil {
				return err
			}
			zones = upsert(zones, config.Timezone{Name: options.BaseZone, Zone: baseZone})
		}
	}
	if err := tzutil.Validate(baseZone); err != nil {
		return err
	}

	if options.FZF {
		selected, err := selectZones(ctx, runner)
		if err != nil {
			return err
		}
		for _, zone := range selected {
			zones = upsert(zones, config.Timezone{Name: filepath.Base(zone), Zone: zone})
		}
	}
	if len(zones) == 0 {
		return fmt.Errorf("no timezones selected")
	}

	engine, err := dateengine.Discover(ctx, runner)
	if err != nil {
		return err
	}
	rows := make([]output.Row, 0, len(zones))
	for _, zone := range zones {
		var value string
		if options.Expression == "" {
			value, err = engine.Now(ctx, zone.Zone, cfg.DateFormat)
		} else {
			value, err = engine.Format(ctx, zone.Zone, baseZone, options.Expression, cfg.DateFormat)
		}
		if err != nil {
			return err
		}
		emoji := zone.Emoji
		if emoji == "" {
			emoji = tzutil.Emoji(zone.Zone)
			if emoji == "" {
				emoji = cfg.DefaultTimezoneEmoji
			}
		}
		rows = append(rows, output.Row{
			Name: zone.Name, Zone: zone.Zone, Value: value, Emoji: emoji, Icon: zone.Icon,
			IsBase:  options.BaseZone != "" && zone.Zone == baseZone,
			IsLocal: options.BaseZone == "" && zone.Zone == localZone,
		})
	}

	switch {
	case options.JSON:
		return output.JSON(stdout, rows)
	case options.Screenshot:
		return runScreenshot(ctx, runner, engine, rows, baseZone, options.Expression, cfg.ScreenshotTheme, !options.NoEmoji, screenshot.SupportsEmoji(runner), stderr)
	default:
		interactive := isTerminal(stdout)
		color := interactive && (options.ForceColor || (!options.NoColor && os.Getenv("NO_COLOR") == ""))
		return output.Terminal(stdout, rows, interactive, color, !options.NoEmoji)
	}
}

func upsert(zones []config.Timezone, added config.Timezone) []config.Timezone {
	for i := range zones {
		if zones[i].Name == added.Name {
			zones[i] = added
			return zones
		}
	}
	return append(zones, added)
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(file.Fd())
}

func selectZones(ctx context.Context, runner command.Runner) ([]string, error) {
	zones, err := tzutil.List()
	if err != nil {
		return nil, fmt.Errorf("list timezones: %w", err)
	}
	path, err := runner.LookPath("fzf")
	if err != nil {
		return nil, fmt.Errorf("fzf is required for -f")
	}
	if _, err := runner.LookPath("zdump"); err != nil {
		return nil, fmt.Errorf("zdump is required for -f previews")
	}
	executable, err := executablePath()
	if err != nil {
		return nil, fmt.Errorf("resolve current executable: %w", err)
	}
	input := strings.NewReader(strings.Join(zones, "\n") + "\n")
	args := []string{
		"--prompt=Select timezone: ", "-s", "-m",
		"--preview=" + shellQuote(executable) + " --preview-zone {}",
	}
	stdout, stderr, err := runner.Run(ctx, path, args, nil, input)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		exitCode := -1
		if exitError, ok := err.(interface{ ExitCode() int }); ok {
			exitCode = exitError.ExitCode()
		}
		if exitCode == 1 || exitCode == 130 {
			return nil, ErrNoTimezoneSelected
		}
		return nil, &SelectionError{ExitCode: exitCode, Stderr: strings.TrimSpace(string(stderr))}
	}
	offered := make(map[string]struct{}, len(zones))
	for _, zone := range zones {
		offered[zone] = struct{}{}
	}
	raw := string(stdout)
	if raw == "" {
		return nil, ErrNoTimezoneSelected
	}
	raw = strings.TrimSuffix(raw, "\n")
	if raw == "" {
		return nil, ErrNoTimezoneSelected
	}
	var selected []string
	seen := make(map[string]struct{})
	for _, zone := range strings.Split(raw, "\n") {
		if zone == "" || strings.ContainsRune(zone, '\r') {
			return nil, &SelectionError{ExitCode: 0, Stderr: "fzf returned malformed output"}
		}
		if _, found := offered[zone]; !found {
			return nil, &SelectionError{ExitCode: 0, Stderr: fmt.Sprintf("fzf returned unoffered timezone %q", zone)}
		}
		if _, duplicate := seen[zone]; duplicate {
			return nil, &SelectionError{ExitCode: 0, Stderr: fmt.Sprintf("fzf returned duplicate timezone %q", zone)}
		}
		seen[zone] = struct{}{}
		selected = append(selected, zone)
	}
	if len(selected) == 0 {
		return nil, ErrNoTimezoneSelected
	}
	return selected, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func previewZone(ctx context.Context, runner command.Runner, zone string, writer io.Writer) error {
	if err := tzutil.Validate(zone); err != nil {
		return err
	}
	zdump, err := runner.LookPath("zdump")
	if err != nil {
		return fmt.Errorf("zdump is required for timezone previews")
	}
	const outputLimit = 1 << 20
	current, currentErr, err := runner.Run(ctx, zdump, []string{zone}, []string{"LC_ALL=C"}, nil)
	if err != nil {
		return fmt.Errorf("zdump current time failed: %s", strings.TrimSpace(string(currentErr)))
	}
	if len(current) > outputLimit {
		return fmt.Errorf("zdump current output exceeds 1 MiB")
	}
	now := currentTime()
	rangeArg := fmt.Sprintf("%d,%d", now.Year(), now.Year()+7)
	transitions, transitionErr, err := runner.Run(ctx, zdump, []string{"-V", "-c", rangeArg, zone}, []string{"LC_ALL=C"}, nil)
	if err != nil {
		return fmt.Errorf("zdump transitions failed: %s", strings.TrimSpace(string(transitionErr)))
	}
	if len(transitions) > outputLimit {
		return fmt.Errorf("zdump transition output exceeds 1 MiB")
	}
	next, err := nextTransition(string(transitions), now)
	if err != nil {
		return err
	}
	if _, err := writer.Write(bytes.TrimSuffix(current, []byte("\n"))); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, "\n\nNext Transition:\n"+next+"\n"); err != nil {
		return err
	}
	return nil
}

func nextTransition(data string, now time.Time) (string, error) {
	lines := strings.Split(strings.TrimSuffix(data, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "No transition in the next six years", nil
	}
	if len(lines)%2 != 0 {
		return "", fmt.Errorf("malformed zdump output: unpaired transition")
	}
	horizon := now.AddDate(6, 0, 0)
	for index := 0; index < len(lines); index += 2 {
		before, err := transitionInstant(lines[index])
		if err != nil {
			return "", err
		}
		after, err := transitionInstant(lines[index+1])
		if err != nil {
			return "", err
		}
		if !after.Equal(before.Add(time.Second)) {
			return "", fmt.Errorf("malformed zdump output: transition records are not adjacent")
		}
		if after.After(now) && !after.After(horizon) {
			return lines[index+1], nil
		}
	}
	return "No transition in the next six years", nil
}

func transitionInstant(line string) (time.Time, error) {
	start := strings.Index(line, "  ")
	end := strings.Index(line, " UT =")
	if start < 0 || end <= start+2 {
		return time.Time{}, fmt.Errorf("malformed zdump transition %q", line)
	}
	value := line[start+2 : end]
	instant, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", value, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse zdump transition %q: %w", line, err)
	}
	return instant, nil
}

func runScreenshot(ctx context.Context, runner command.Runner, engine dateengine.Engine, rows []output.Row, baseZone, expression, theme string, avatars, emoji bool, stderr io.Writer) error {
	result := make([]screenshot.Row, 0, len(rows))
	for _, row := range rows {
		format := func(format string) (string, error) {
			if expression == "" {
				return engine.Now(ctx, row.Zone, format)
			}
			return engine.Format(ctx, row.Zone, baseZone, expression, format)
		}
		date, err := format("%a, %d %B")
		if err != nil {
			return err
		}
		clock, err := format("%H:%M")
		if err != nil {
			return err
		}
		abbr, err := format("%Z")
		if err != nil {
			return err
		}
		result = append(result, screenshot.Row{Row: row, Date: date, Time: clock, Abbr: abbr, Country: tzutil.Country(row.Zone)})
	}
	header, err := engine.Now(ctx, baseZone, "%a, %d %B · %H:%M %Z")
	if err != nil {
		return err
	}
	svg := screenshot.SVG(result, header, theme, avatars, emoji)
	if err := screenshot.RenderAndCopy(ctx, runner, svg, theme); err != nil {
		return err
	}
	_, err = io.Copy(stderr, bytes.NewBufferString("Screenshot copied to the clipboard.\n"))
	return err
}
