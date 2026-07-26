package timezone

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const Zoneinfo = "/usr/share/zoneinfo"

var (
	countriesOnce   sync.Once
	countriesByZone map[string]string
)

func Validate(zone string) error {
	if zone == "" || strings.ContainsAny(zone, "\"\n\r\x00") {
		return fmt.Errorf("invalid timezone %q", zone)
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", zone, err)
	}
	return nil
}

func Local() (string, error) {
	if zone := os.Getenv("TZ"); zone != "" && !strings.HasPrefix(zone, ":") {
		if err := Validate(zone); err == nil {
			return zone, nil
		}
	}
	target, err := filepath.EvalSymlinks("/etc/localtime")
	if err == nil {
		if marker := string(filepath.Separator) + "zoneinfo" + string(filepath.Separator); strings.Contains(target, marker) {
			zone := strings.SplitN(target, marker, 2)[1]
			if err := Validate(zone); err == nil {
				return zone, nil
			}
		}
	}
	data, readErr := os.ReadFile("/etc/timezone")
	if readErr == nil {
		zone := strings.TrimSpace(string(data))
		if err := Validate(zone); err == nil {
			return zone, nil
		}
	}
	return "", fmt.Errorf("could not determine local IANA timezone; set TZ or use -t")
}

func List() ([]string, error) {
	var zones []string
	err := filepath.WalkDir(Zoneinfo, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "posix", "right", "SystemV":
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(Zoneinfo, path)
		if err != nil || strings.Contains(relative, ".") || !strings.Contains(relative, "/") {
			return nil
		}
		if err := Validate(relative); err == nil {
			zones = append(zones, relative)
		}
		return nil
	})
	sort.Strings(zones)
	return zones, err
}

func Countries() map[string]string {
	countriesOnce.Do(loadCountries)
	result := make(map[string]string, len(countriesByZone))
	for zone, country := range countriesByZone {
		result[zone] = country
	}
	return result
}

func Country(zone string) string {
	countriesOnce.Do(loadCountries)
	if country := countriesByZone[zone]; country != "" {
		return country
	}
	target, err := filepath.EvalSymlinks(filepath.Join(Zoneinfo, filepath.FromSlash(zone)))
	if err != nil {
		return ""
	}
	canonical, err := filepath.Rel(Zoneinfo, target)
	if err != nil || strings.HasPrefix(canonical, "..") {
		return ""
	}
	if country := countriesByZone[filepath.ToSlash(canonical)]; country != "" {
		return country
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		return ""
	}
	for candidate, country := range countriesByZone {
		candidateInfo, err := os.Stat(filepath.Join(Zoneinfo, filepath.FromSlash(candidate)))
		if err == nil && os.SameFile(targetInfo, candidateInfo) {
			return country
		}
	}
	return ""
}

func Emoji(zone string) string {
	if zone == "UTC" || zone == "GMT" || strings.HasPrefix(zone, "Etc/") {
		return "🌍 "
	}
	country := Country(zone)
	if len(country) != 2 {
		return ""
	}
	first, second := country[0], country[1]
	if first < 'A' || first > 'Z' || second < 'A' || second > 'Z' {
		return ""
	}
	const regionalIndicatorOffset = rune(0x1F1E6 - 'A')
	return string([]rune{rune(first) + regionalIndicatorOffset, rune(second) + regionalIndicatorOffset}) + " "
}

func loadCountries() {
	result := make(map[string]string)
	for _, name := range []string{"zone.tab", "zone1970.tab"} {
		file, err := os.Open(filepath.Join(Zoneinfo, name))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 3 && !strings.HasPrefix(fields[0], "#") {
				if _, exists := result[fields[2]]; !exists {
					result[fields[2]] = strings.Split(fields[0], ",")[0]
				}
			}
		}
		_ = file.Close()
	}
	type alias struct {
		canonical string
		name      string
	}
	var aliases []alias
	if file, err := os.Open(filepath.Join(Zoneinfo, "tzdata.zi")); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) == 3 && fields[0] == "L" {
				aliases = append(aliases, alias{canonical: fields[1], name: fields[2]})
			}
		}
		_ = file.Close()
	}
	for range aliases {
		changed := false
		for _, alias := range aliases {
			if result[alias.name] == "" && result[alias.canonical] != "" {
				result[alias.name] = result[alias.canonical]
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	countriesByZone = result
}
