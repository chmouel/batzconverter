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
	return ValidateAt(Zoneinfo, zone)
}

func ValidateAt(root, zone string) error {
	if zone == "" || zone == "Local" || filepath.IsAbs(zone) || strings.Contains(zone, `\`) {
		return fmt.Errorf("invalid timezone %q", zone)
	}
	for _, character := range zone {
		if character > 127 ||
			!((character >= 'a' && character <= 'z') ||
				(character >= 'A' && character <= 'Z') ||
				(character >= '0' && character <= '9') ||
				strings.ContainsRune("_-+./", character)) {
			return fmt.Errorf("invalid timezone %q", zone)
		}
	}
	for _, segment := range strings.Split(zone, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasPrefix(segment, "-") {
			return fmt.Errorf("invalid timezone %q", zone)
		}
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve zoneinfo root: %w", err)
	}
	target, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, filepath.FromSlash(zone)))
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", zone, err)
	}
	relative, err := filepath.Rel(resolvedRoot, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid timezone %q: target escapes zoneinfo root", zone)
	}
	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("invalid timezone %q: not a regular zone file", zone)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", zone, err)
	}
	if _, err := time.LoadLocationFromTZData(zone, data); err != nil {
		return fmt.Errorf("invalid timezone %q: not TZif data", zone)
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
	return ListAt(Zoneinfo)
}

func ListAt(root string) ([]string, error) {
	zones, err := listFromMetadata(root)
	if err == nil && len(zones) > 0 {
		return zones, nil
	}
	return walkZones(root)
}

func listFromMetadata(root string) ([]string, error) {
	set := map[string]struct{}{"UTC": {}, "GMT": {}}
	foundCanonical := false
	for _, name := range []string{"zone.tab", "zone1970.tab"} {
		file, err := os.Open(filepath.Join(root, name))
		if err != nil {
			if name == "zone.tab" {
				return nil, err
			}
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 3 {
				_ = file.Close()
				return nil, fmt.Errorf("malformed %s", name)
			}
			zone := fields[2]
			if ValidateAt(root, zone) == nil {
				set[zone] = struct{}{}
				foundCanonical = true
			}
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	if !foundCanonical {
		return nil, fmt.Errorf("timezone metadata contains no zones")
	}
	if file, err := os.Open(filepath.Join(root, "tzdata.zi")); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) == 3 && fields[0] == "L" && ValidateAt(root, fields[2]) == nil {
				set[fields[2]] = struct{}{}
			}
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return nil, err
		}
		_ = file.Close()
	}
	zones := make([]string, 0, len(set))
	for zone := range set {
		if ValidateAt(root, zone) == nil {
			zones = append(zones, zone)
		}
	}
	sort.Strings(zones)
	return zones, nil
}

func walkZones(root string) ([]string, error) {
	var zones []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		zone := filepath.ToSlash(relative)
		switch zone {
		case "zone.tab", "zone1970.tab", "tzdata.zi", "iso3166.tab", "leap-seconds.list",
			"leapseconds", "localtime", "posixrules":
			return nil
		}
		if err := ValidateAt(root, zone); err == nil {
			zones = append(zones, zone)
		}
		return nil
	})
	sort.Strings(zones)
	if err == nil && len(zones) == 0 {
		err = fmt.Errorf("no timezones found in %s", root)
	}
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
		if sameZoneFile(zone, "UTC") || sameZoneFile(zone, "GMT") {
			return "🌍 "
		}
		return ""
	}

	first, second := country[0], country[1]
	if first < 'A' || first > 'Z' || second < 'A' || second > 'Z' {
		return ""
	}
	const regionalIndicatorOffset = rune(0x1F1E6 - 'A')
	return string([]rune{rune(first) + regionalIndicatorOffset, rune(second) + regionalIndicatorOffset}) + " "
}

func sameZoneFile(first, second string) bool {
	firstInfo, firstErr := os.Stat(filepath.Join(Zoneinfo, filepath.FromSlash(first)))
	secondInfo, secondErr := os.Stat(filepath.Join(Zoneinfo, filepath.FromSlash(second)))
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
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
