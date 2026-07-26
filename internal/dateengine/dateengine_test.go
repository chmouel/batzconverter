package dateengine

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type call struct {
	name string
	args []string
	env  []string
}

type fakeRunner struct {
	paths map[string]string
	calls []call
	fail  bool
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	path, found := f.paths[name]
	if !found {
		return "", errors.New("not found")
	}
	return path, nil
}

func (f *fakeRunner) Run(_ context.Context, name string, args, env []string, _ io.Reader) ([]byte, []byte, error) {
	f.calls = append(f.calls, call{name: name, args: append([]string(nil), args...), env: append([]string(nil), env...)})
	if len(args) == 1 && args[0] == "--version" {
		return []byte("date (GNU coreutils) 9.8\n"), nil, nil
	}
	if f.fail {
		return nil, []byte("date: invalid date"), errors.New("exit status 1")
	}
	return []byte("result\n"), nil, nil
}

func TestDiscoverAndFormat(t *testing.T) {
	runner := &fakeRunner{paths: map[string]string{"gdate": "/opt/gdate", "date": "/bin/date"}}
	engine, err := Discover(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	value, err := engine.Format(context.Background(), "America/New_York", "Europe/Paris", "10:00 tomorrow", "%c")
	if err != nil {
		t.Fatal(err)
	}
	if value != "result" {
		t.Fatalf("value = %q", value)
	}
	got := runner.calls[len(runner.calls)-1]
	wantArgs := []string{`--date=TZ="Europe/Paris" 10:00 tomorrow`, "+%c"}
	if !reflect.DeepEqual(got.args, wantArgs) || !reflect.DeepEqual(got.env, []string{"TZ=America/New_York"}) {
		t.Fatalf("call = %#v, want args %#v", got, wantArgs)
	}
}

func TestFormatIncludesGNUError(t *testing.T) {
	runner := &fakeRunner{fail: true}
	engine := &GNUDate{path: "/bin/date", runner: runner}
	_, err := engine.Format(context.Background(), "UTC", "UTC", "nonsense", "%c")
	if err == nil || !strings.Contains(err.Error(), "invalid date") || !strings.Contains(err.Error(), "nonsense") {
		t.Fatalf("error = %v", err)
	}
}

func TestDiscoverRejectsMissingGNUDate(t *testing.T) {
	_, err := Discover(context.Background(), &fakeRunner{paths: map[string]string{}})
	if err == nil || !strings.Contains(err.Error(), "GNU date is required") {
		t.Fatalf("error = %v", err)
	}
}
