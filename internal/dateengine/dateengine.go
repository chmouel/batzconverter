package dateengine

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chmouel/batzconverter/internal/command"
)

type Engine interface {
	Format(ctx context.Context, targetZone, baseZone, expression, format string) (string, error)
	Now(ctx context.Context, zone, format string) (string, error)
}

type GNUDate struct {
	path   string
	runner command.Runner
}

func Discover(ctx context.Context, runner command.Runner) (*GNUDate, error) {
	for _, candidate := range []string{"gdate", "date"} {
		path, err := runner.LookPath(candidate)
		if err != nil {
			continue
		}
		stdout, _, err := runner.Run(ctx, path, []string{"--version"}, nil, nil)
		if err != nil || !strings.Contains(string(stdout), "GNU coreutils") {
			continue
		}
		if _, _, err := runner.Run(ctx, path, []string{"--date=tomorrow", "+%s"}, []string{"TZ=UTC"}, nil); err == nil {
			return &GNUDate{path: path, runner: runner}, nil
		}
	}
	return nil, fmt.Errorf("GNU date is required; install coreutils (Homebrew: brew install coreutils)")
}

func (g *GNUDate) Format(ctx context.Context, targetZone, baseZone, expression, format string) (string, error) {
	dateExpression := fmt.Sprintf(`TZ="%s" %s`, baseZone, expression)
	return g.run(ctx, targetZone, []string{"--date=" + dateExpression, "+" + format}, expression)
}

func (g *GNUDate) Now(ctx context.Context, zone, format string) (string, error) {
	return g.run(ctx, zone, []string{"+" + format}, "now")
}

func (g *GNUDate) run(ctx context.Context, zone string, args []string, expression string) (string, error) {
	stdout, stderr, err := g.runner.Run(ctx, g.path, args, []string{"TZ=" + zone}, io.Reader(nil))
	if err != nil {
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("GNU date could not parse %q: %s", expression, detail)
	}
	return strings.TrimSuffix(string(stdout), "\n"), nil
}
