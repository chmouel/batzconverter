package screenshot

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chmouel/batzconverter/internal/command"
	"github.com/chmouel/batzconverter/internal/output"
)

type Row struct {
	output.Row
	Date    string
	Time    string
	Abbr    string
	Country string
}

type palette struct {
	background, panel, panelStroke, title, muted, row, rowStroke       string
	accent, local, localStroke, time, badge, badgeText, zone, zoneText string
}

func colors(theme string) palette {
	if theme == "dark" {
		return palette{
			background: "#090b18", panel: "#141426", panelStroke: "#2a293d",
			title: "#f5f3ff", muted: "#958faa", row: "#1c1c30", rowStroke: "#302f43",
			accent: "#a78bfa", local: "#252044", localStroke: "#8b5cf6",
			time: "#ffffff", badge: "#302351", badgeText: "#c4b5fd",
			zone: "#102f38", zoneText: "#67e8f9",
		}
	}
	return palette{
		background: "#f2f6ff", panel: "#ffffff", panelStroke: "#cbd5e1",
		title: "#0f172a", muted: "#64748b", row: "#f8fafc", rowStroke: "#e2e8f0",
		accent: "#7c3aed", local: "#f0ecff", localStroke: "#a78bfa",
		time: "#111827", badge: "#ede9fe", badgeText: "#5b21b6",
		zone: "#ecfeff", zoneText: "#0e7490",
	}
}

func SVG(rows []Row, header, theme string, avatars, emoji bool) []byte {
	p := colors(theme)
	height := 210 + len(rows)*84
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="%d" viewBox="0 0 1200 %d">`, height, height)
	fmt.Fprintf(&b, `<rect width="1200" height="%d" fill="%s"/>`, height, p.background)
	fmt.Fprintf(&b, `<rect x="32" y="28" width="1136" height="%d" rx="28" fill="%s" stroke="%s"/>`, height-56, p.panel, p.panelStroke)
	fmt.Fprintf(&b, `<rect x="60" y="28" width="1080" height="4" rx="2" fill="%s"/>`, p.accent)
	fmt.Fprintf(&b, `<text x="72" y="82" fill="%s" font-family="Inter,Arial,sans-serif" font-size="29" font-weight="700">World Clock</text>`, p.title)
	fmt.Fprintf(&b, `<text x="72" y="105" fill="%s" font-family="Inter,Arial,sans-serif" font-size="13" letter-spacing="2.4">TIME AROUND THE WORLD</text>`, p.muted)
	fmt.Fprintf(&b, `<rect x="735" y="57" width="375" height="49" rx="24" fill="%s" stroke="%s"/>`, p.row, p.rowStroke)
	fmt.Fprintf(&b, `<circle cx="764" cy="81" r="5" fill="%s"/><text x="786" y="88" fill="%s" font-family="monospace" font-size="16">%s</text>`, p.accent, p.title, html.EscapeString(header))
	for i, row := range rows {
		y := 142 + i*84
		fill, stroke := p.row, p.rowStroke
		if row.IsBase || row.IsLocal {
			fill, stroke = p.local, p.localStroke
		}
		fmt.Fprintf(&b, `<rect x="58" y="%d" width="1084" height="72" rx="18" fill="%s" stroke="%s"/>`, y, fill, stroke)
		cityX := 82
		avatar := ""
		if avatars {
			if emoji && row.Emoji != "" {
				avatar = strings.TrimSpace(row.Emoji)
			} else {
				avatar = row.Country
				if avatar == "" {
					avatar = initials(row.Name)
				}
			}
		}
		if avatar != "" {
			cityX = 142
			fmt.Fprintf(&b, `<circle cx="92" cy="%d" r="24" fill="%s"/><text x="92" y="%d" fill="%s" font-family="Arial,sans-serif" font-size="15" font-weight="800" text-anchor="middle">%s</text>`, y+36, p.rowStroke, y+42, p.title, html.EscapeString(avatar))
		}
		fontSize := 26
		if len(row.Name) > 25 {
			fontSize = 18
		} else if len(row.Name) > 18 {
			fontSize = 21
		}
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s" font-family="Inter,Arial,sans-serif" font-size="%d" font-weight="600">%s</text>`, cityX, y+31, p.title, fontSize, html.EscapeString(row.Name))
		fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s" font-family="Inter,Arial,sans-serif" font-size="15">%s</text>`, cityX, y+55, p.muted, html.EscapeString(row.Date))
		badge := ""
		if row.IsBase {
			badge = "BASE"
		} else if row.IsLocal {
			badge = "LOCAL"
		}
		if badge != "" {
			fmt.Fprintf(&b, `<rect x="620" y="%d" width="82" height="28" rx="14" fill="%s"/><text x="661" y="%d" fill="%s" font-family="Arial,sans-serif" font-size="11" font-weight="700" text-anchor="middle">%s</text>`, y+21, p.badge, y+40, p.badgeText, badge)
		}
		fmt.Fprintf(&b, `<text x="935" y="%d" fill="%s" font-family="monospace" font-size="36" text-anchor="end">%s</text>`, y+48, p.time, html.EscapeString(row.Time))
		fmt.Fprintf(&b, `<rect x="960" y="%d" width="100" height="38" rx="19" fill="%s"/><text x="1010" y="%d" fill="%s" font-family="monospace" font-size="15" font-weight="600" text-anchor="middle">%s</text>`, y+17, p.zone, y+42, p.zoneText, html.EscapeString(row.Abbr))
	}
	b.WriteString("</svg>")
	return []byte(b.String())
}

func initials(name string) string {
	var result []rune
	for _, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result = append(result, r)
		}
		if len(result) == 2 {
			break
		}
	}
	if len(result) == 0 {
		for _, r := range strings.ToUpper(name) {
			if r >= 'A' && r <= 'Z' {
				result = append(result, r)
			}
			if len(result) == 2 {
				break
			}
		}
	}
	if len(result) == 0 {
		return "TZ"
	}
	return string(result)
}

func RenderAndCopy(ctx context.Context, runner command.Runner, svg []byte, theme string) error {
	dir, err := os.MkdirTemp("", "batz-go-*")
	if err != nil {
		return fmt.Errorf("create screenshot directory: %w", err)
	}
	defer os.RemoveAll(dir)
	svgPath, pngPath := filepath.Join(dir, "batz.svg"), filepath.Join(dir, "batz.png")
	if err := os.WriteFile(svgPath, svg, 0o600); err != nil {
		return fmt.Errorf("write screenshot SVG: %w", err)
	}
	background := colors(theme).background
	if err := rasterize(ctx, runner, svgPath, pngPath, background); err != nil {
		return err
	}
	png, err := os.ReadFile(pngPath)
	if err != nil {
		return fmt.Errorf("read rendered screenshot: %w", err)
	}
	return copyPNG(ctx, runner, pngPath, png)
}

func SupportsEmoji(runner command.Runner) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := runner.LookPath("swift")
	return err == nil
}

func rasterize(ctx context.Context, runner command.Runner, svgPath, pngPath, background string) error {
	if runtime.GOOS == "darwin" {
		if path, err := runner.LookPath("swift"); err == nil {
			_, stderr, err := runner.Run(ctx, path, []string{"-", svgPath, pngPath, background}, nil, strings.NewReader(swiftRenderer))
			if err != nil {
				return fmt.Errorf("Swift/AppKit failed: %s", strings.TrimSpace(string(stderr)))
			}
			return nil
		}
	}
	type candidate struct {
		name string
		args []string
	}
	candidates := []candidate{
		{"rsvg-convert", []string{"--background-color", background, svgPath, "--output", pngPath}},
		{"magick", []string{"-background", background, svgPath, "-alpha", "remove", pngPath}},
		{"convert", []string{"-background", background, svgPath, "-alpha", "remove", pngPath}},
	}
	for _, candidate := range candidates {
		path, err := runner.LookPath(candidate.name)
		if err != nil {
			continue
		}
		_, stderr, err := runner.Run(ctx, path, candidate.args, nil, nil)
		if err != nil {
			return fmt.Errorf("%s failed: %s", candidate.name, strings.TrimSpace(string(stderr)))
		}
		return nil
	}
	return fmt.Errorf("screenshot output needs rsvg-convert or ImageMagick")
}

const swiftRenderer = `import AppKit

func fail(_ message: String) -> Never {
  FileHandle.standardError.write(Data(("batz-go: " + message + "\n").utf8))
  exit(1)
}

let source = CommandLine.arguments[1]
let destination = CommandLine.arguments[2]
let backgroundHex = CommandLine.arguments[3]
var value: UInt64 = 0
let scanner = Scanner(string: backgroundHex.hasPrefix("#") ? String(backgroundHex.dropFirst()) : backgroundHex)
guard scanner.scanHexInt64(&value) else { fail("invalid background color " + backgroundHex) }
let background = NSColor(
  red: CGFloat((value >> 16) & 0xFF) / 255,
  green: CGFloat((value >> 8) & 0xFF) / 255,
  blue: CGFloat(value & 0xFF) / 255,
  alpha: 1
)
guard let image = NSImage(contentsOfFile: source) else { fail("could not load the generated SVG") }
let width = Int(image.size.width)
let height = Int(image.size.height)
guard let bitmap = NSBitmapImageRep(
  bitmapDataPlanes: nil, pixelsWide: width, pixelsHigh: height,
  bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true,
  isPlanar: false, colorSpaceName: .deviceRGB,
  bytesPerRow: 0, bitsPerPixel: 0
) else { fail("could not allocate the PNG bitmap") }
NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: bitmap)
background.setFill()
NSBezierPath(rect: NSRect(x: 0, y: 0, width: width, height: height)).fill()
image.draw(in: NSRect(x: 0, y: 0, width: width, height: height))
NSGraphicsContext.restoreGraphicsState()
guard let data = bitmap.representation(using: .png, properties: [:]) else { fail("could not encode the PNG") }
try data.write(to: URL(fileURLWithPath: destination))
`

func copyPNG(ctx context.Context, runner command.Runner, pngPath string, png []byte) error {
	switch runtime.GOOS {
	case "darwin":
		path, err := runner.LookPath("osascript")
		if err != nil {
			return fmt.Errorf("osascript is required to copy PNG images on macOS")
		}
		script := `on run argv
set imageFile to POSIX file (item 1 of argv)
set the clipboard to (read imageFile as «class PNGf»)
end run`
		_, stderr, err := runner.Run(ctx, path, []string{"-", pngPath}, nil, strings.NewReader(script))
		if err != nil {
			return fmt.Errorf("copy PNG to macOS clipboard: %s", strings.TrimSpace(string(stderr)))
		}
	case "linux":
		if os.Getenv("WAYLAND_DISPLAY") == "" {
			return fmt.Errorf("screenshot clipboard output on Linux is only supported on Wayland")
		}
		path, err := runner.LookPath("wl-copy")
		if err != nil {
			return fmt.Errorf("wl-copy is required to copy PNG images on Wayland")
		}
		_, stderr, err := runner.Run(ctx, path, []string{"--type", "image/png"}, nil, bytes.NewReader(png))
		if err != nil {
			return fmt.Errorf("copy PNG to Wayland clipboard: %s", strings.TrimSpace(string(stderr)))
		}
	default:
		return fmt.Errorf("screenshot clipboard output is supported on macOS and Wayland")
	}
	return nil
}
