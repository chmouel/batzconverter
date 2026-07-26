package timezone

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, zone := range []string{"UTC", "Europe/Paris", "America/New_York"} {
		if err := Validate(zone); err != nil {
			t.Errorf("Validate(%q): %v", zone, err)
		}
	}
	for _, zone := range []string{"", "Not/A_Zone", "UTC\nBAD", `UTC" BAD`} {
		if err := Validate(zone); err == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", zone)
		}
	}
}

func TestValidateRejectsUnsafeIdentifiers(t *testing.T) {
	for _, zone := range []string{"Local", "/UTC", "../UTC", "Etc//UTC", `Etc\UTC`, "Etc/-UTC", "Europé/Paris", "zone.tab", "iso3166.tab"} {
		if err := Validate(zone); err == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", zone)
		}
	}
}

func TestLocalHonorsTZ(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	zone, err := Local()
	if err != nil {
		t.Fatal(err)
	}
	if zone != "Europe/Paris" {
		t.Fatalf("zone = %q", zone)
	}
}

func TestCountriesContainsKnownZone(t *testing.T) {
	if country := Countries()["Europe/Paris"]; country != "FR" {
		t.Fatalf("country = %q", country)
	}
}

func TestCountryAndEmoji(t *testing.T) {
	tests := map[string]struct {
		country string
		emoji   string
	}{
		"Europe/Paris":  {country: "FR", emoji: "🇫🇷 "},
		"Asia/Calcutta": {country: "IN", emoji: "🇮🇳 "},
		"UTC":           {emoji: "🌍 "},
		"Etc/GMT+4":     {emoji: "🌍 "},
	}
	for zone, want := range tests {
		t.Run(zone, func(t *testing.T) {
			if country := Country(zone); country != want.country {
				t.Errorf("Country(%q) = %q, want %q", zone, country, want.country)
			}
			if emoji := Emoji(zone); emoji != want.emoji {
				t.Errorf("Emoji(%q) = %q, want %q", zone, emoji, want.emoji)
			}
		})
	}
}

func TestAllListedZonesHaveEmoji(t *testing.T) {
	zones, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, zone := range zones {
		if emoji := Emoji(zone); emoji == "" {
			t.Errorf("Emoji(%q) is empty", zone)
		}
	}
}

func TestListAtMetadataAndFallback(t *testing.T) {
	root := t.TempDir()
	zoneData, err := os.ReadFile(filepath.Join(Zoneinfo, "UTC"))
	if err != nil {
		t.Fatal(err)
	}
	for _, zone := range []string{"Area/City", "Area/Alias", "UTC", "GMT"} {
		path := filepath.Join(root, filepath.FromSlash(zone))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, zoneData, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "zone.tab"), []byte("AA\t+0000\tArea/City\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tzdata.zi"), []byte("L Area/City Area/Alias\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zones, err := ListAt(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, zone := range []string{"Area/Alias", "Area/City", "GMT", "UTC"} {
		if !slices.Contains(zones, zone) {
			t.Errorf("zones does not contain %q: %#v", zone, zones)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "zone.tab"), []byte("malformed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zones, err = ListAt(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(zones, "Area/City") {
		t.Fatalf("fallback zones = %#v", zones)
	}
}
