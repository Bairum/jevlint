package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestResolveIgnoresUnverifiedLegacyPackCache(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack cache tests")
	}
	const manifest = `{"version":1,"id":"owner/pack"}`
	repo := writePackRepo(t, manifest, validRulesJSON)
	pin, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	legacy := filepath.Join(cache, "jevlint", "packs", pin, "owner", "pack")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, ManifestFile), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	unverified := strings.ReplaceAll(validRulesJSON, "Join related records.", "UNVERIFIED-CACHE-CONTENT")
	if err := os.WriteFile(filepath.Join(legacy, defaultRulesFile), []byte(unverified), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Resolve([]config.PackRef{{ID: "owner/pack", Source: repo, SHA: pin}}, func() (string, error) { return cache, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Rules[0].Description != "Join related records." || loaded[0].Dir == legacy {
		t.Fatalf("Resolve() trusted legacy cache instead of fetching pinned content: %#v", loaded)
	}
	contents, err := os.ReadFile(filepath.Join(legacy, defaultRulesFile))
	if err != nil || string(contents) != unverified {
		t.Fatalf("legacy cache changed: %q, %v", contents, err)
	}
}
