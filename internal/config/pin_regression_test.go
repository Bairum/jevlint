package config

import (
	"strings"
	"testing"
)

func TestConfigRequiresFullCommitPackPins(t *testing.T) {
	for _, pin := range []string{"HEAD", "main", "v1", "abc123", strings.Repeat("a", 39), strings.Repeat("a", 41), strings.Repeat("a", 63), strings.Repeat("a", 65), strings.Repeat("g", 40), strings.Repeat("a", 40) + "^"} {
		t.Run(pin, func(t *testing.T) {
			cfg := Config{Languages: map[string]Language{"go": {}}, Packs: []PackRef{{ID: "owner/pack", Source: "local", SHA: pin}}}
			if err := cfg.Validate(); err == nil {
				t.Fatalf("Validate() accepted non-commit pack pin %q", pin)
			}
		})
	}
	for _, pin := range []string{strings.Repeat("a", 40), strings.Repeat("b", 64), strings.Repeat("A", 40)} {
		cfg := Config{Languages: map[string]Language{"go": {}}, Packs: []PackRef{{ID: "owner/pack", Source: "local", SHA: pin}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() rejected full commit pin %q: %v", pin, err)
		}
	}
}
