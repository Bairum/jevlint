package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestResolveRejectsFullSHADefaultBranchImpersonation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack pin tests")
	}
	const manifest = `{"version": 1, "id": "owner/pack", "languages": ["go"]}`
	original := writePackRepo(t, manifest, validRulesJSON)
	pin, err := gitOutput(original, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	replacement := writePackRepo(t, manifest, strings.ReplaceAll(validRulesJSON, "Join related records.", "REPLACEMENT-CONTENT"))
	packsRunGit(t, replacement, "branch", "-m", pin)
	cache := t.TempDir()
	_, resolveErr := Resolve([]config.PackRef{{ID: "owner/pack", Source: replacement, SHA: pin}}, func() (string, error) { return cache, nil })
	entries, err := os.ReadDir(cache)
	if err != nil {
		t.Fatal(err)
	}
	if resolveErr == nil || len(entries) != 0 {
		t.Fatalf("Resolve() error = %v, cache entries = %v; want rejection with no cache population", resolveErr, entries)
	}
}

func TestPackPinsRejectSymbolicAndTruncatedValues(t *testing.T) {
	for _, pin := range []string{"HEAD", "main", "v1", "abc123", strings.Repeat("a", 39), strings.Repeat("a", 41), strings.Repeat("g", 40), strings.Repeat("a", 40) + "^"} {
		t.Run(pin, func(t *testing.T) {
			cache := t.TempDir()
			// A populated cache must not make an invalid pin acceptable.
			dir := filepath.Join(cache, "jevlint", "packs", pin, "owner", "pack")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{"version":1,"id":"owner/pack"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, defaultRulesFile), []byte(validRulesJSON), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := CacheDir(cache, pin, "owner/pack"); err == nil {
				t.Errorf("CacheDir() accepted non-commit pin %q", pin)
			}
			if _, err := Resolve([]config.PackRef{{ID: "owner/pack", Source: "unused", SHA: pin}}, func() (string, error) { return cache, nil }); err == nil {
				t.Errorf("Resolve() accepted cached non-commit pin %q", pin)
			}
		})
	}
}

func TestInstallRefsAndResolvePinnedCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack pin tests")
	}
	repo := writePackRepo(t, `{"version":1,"id":"owner/pack"}`, validRulesJSON)
	pin, err := gitOutput(repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	packsRunGit(t, repo, "tag", "-a", "v1", "-m", "release")
	for _, ref := range []string{"main", "v1", pin} {
		t.Run(ref, func(t *testing.T) {
			cache := t.TempDir()
			installed, err := Install(repo+"@"+ref, func() (string, error) { return cache, nil })
			if err != nil || installed.SHA != pin || installed.Ref != ref {
				t.Fatalf("Install() = %#v, %v; want pin %s and ref %s", installed, err, pin, ref)
			}
		})
	}
	if err := os.WriteFile(filepath.Join(repo, defaultRulesFile), []byte(strings.ReplaceAll(validRulesJSON, "Join related records.", "New default content.")), 0o600); err != nil {
		t.Fatal(err)
	}
	packsRunGit(t, repo, "add", ".")
	packsRunGit(t, repo, "commit", "-m", "advance main")
	cache := t.TempDir()
	loaded, err := Resolve([]config.PackRef{{ID: "owner/pack", Source: repo, Ref: "main", SHA: pin}}, func() (string, error) { return cache, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Rules[0].Description != "Join related records." {
		t.Fatalf("Resolve() did not retain pinned content: %#v", loaded)
	}
}
