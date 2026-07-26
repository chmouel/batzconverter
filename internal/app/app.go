package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chmouel/batzconverter/internal/cli"
	"github.com/chmouel/batzconverter/internal/command"
	"github.com/chmouel/batzconverter/internal/config"
	"github.com/chmouel/batzconverter/internal/dateengine"
	"github.com/chmouel/batzconverter/internal/output"
	"github.com/chmouel/batzconverter/internal/screenshot"
	tzutil "github.com/chmouel/batzconverter/internal/timezone"
)

const help = `batz-go - timezone converter

Usage:
  batz-go [flags] [+Area/Location ...] [TIME [GNU DATE EXPRESSION...]]

Examples:
  batz-go
  batz-go 10h30
  batz-go 10h00 tomorrow
  batz-go -t UTC 11:00 next Thursday
  batz-go +America/Chicago +UTC 10h00 tomorrow

Flags:
  -t ZONE  use another timezone as the base
  -j       emit Alfred-compatible JSON
  -n       disable colors
  -C       force colors
  -E       disable emojis
  -f       select timezones with fzf
  -g       format output with gum
  -s       render and copy a screenshot
  -S THEME screenshot theme: light or dark
  -h       show help
`

func Run(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) error {
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
	options.Gum = options.Gum || cfg.UseGum

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
	case options.Gum:
		return runGum(ctx, runner, rows, !options.NoEmoji, stdout)
	default:
		color := options.ForceColor || (!options.NoColor && os.Getenv("NO_COLOR") == "" && isTerminal(stdout))
		return output.Terminal(stdout, rows, cfg.DateFormatPadding, color, !options.NoEmoji)
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
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
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
	input := strings.NewReader(strings.Join(zones, "\n") + "\n")
	args := []string{"--prompt=Select timezone: ", "-s", "-m", "--preview=zdump {}"}
	stdout, stderr, err := runner.Run(ctx, path, args, nil, input)
	if err != nil {
		return nil, fmt.Errorf("fzf selection failed: %s", strings.TrimSpace(string(stderr)))
	}
	var selected []string
	for _, zone := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
		if zone == "" {
			continue
		}
		if err := tzutil.Validate(zone); err != nil {
			return nil, err
		}
		selected = append(selected, zone)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no timezone selected")
	}
	return selected, nil
}

func runGum(ctx context.Context, runner command.Runner, rows []output.Row, emoji bool, stdout io.Writer) error {
	path, err := runner.LookPath("gum")
	if err != nil {
		return fmt.Errorf("gum is required for -g")
	}
	data, err := output.CSV(rows, emoji)
	if err != nil {
		return fmt.Errorf("encode gum input: %w", err)
	}
	result, commandErr, err := runner.Run(ctx, path, []string{"table", "-p"}, nil, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("gum table failed: %s", strings.TrimSpace(string(commandErr)))
	}
	_, err = stdout.Write(result)
	return err
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
