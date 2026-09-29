package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"jevlint/internal/config"
)

const validRulesJSON = `{
	"rules": [{
		"id": "database-joins",
		"description": "Join related records.",
		"severity": "error"
	}]
}`

func TestSafeJoinRejectsEscapes(t *testing.T) {
	t.Parallel()

	root := filepath.Join(string(filepath.Separator), "cache", "packs")
	allowed := map[string]string{
		"rules.json":        filepath.Join(root, "rules.json"),
		"nested/evals.json": filepath.Join(root, "nested", "evals.json"),
		"./rules.json":      filepath.Join(root, "rules.json"),
		"a/b/../c.json":     filepath.Join(root, "a", "c.json"),
	}
	for input, want := range allowed {
		got, err := safeJoin(root, input)
		if err != nil {
			t.Fatalf("safeJoin(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("safeJoin(%q) = %q, want %q", input, got, want)
		}
	}

	rejected := []string{
		"",
		"..",
		"../outside",
		"../../outside",
		"nested/../../outside",
		"/etc/passwd",
		filepath.Join(string(filepath.Separator), "abs", "path"),
	}
	for _, input := range rejected {
		if got, err := safeJoin(root, input); err == nil {
			t.Fatalf("safeJoin(%q) = %q, want an error", input, got)
		}
	}
}

func TestCacheDirRejectsTraversal(t *testing.T) {
	t.Parallel()

	userCache := t.TempDir()
	for _, test := range []struct {
		name string
		sha  string
		id   string
	}{
		{"traversing id", "abc123", "../../outside"},
		{"absolute id", "abc123", "/etc"},
		{"empty id", "abc123", ""},
		{"traversing sha", "../../outside", "database-joins"},
		{"slash in id", "abc123", "pack/bad"},
	} {
		if got, err := CacheDir(userCache, test.sha, test.id); err == nil {
			t.Fatalf("CacheDir(%q, %q) = %q, want an error", test.sha, test.id, got)
		}
	}

	got, err := CacheDir(userCache, "abc123", "database-joins")
	if err != nil {
		t.Fatalf("CacheDir() error = %v", err)
	}
	want := filepath.Join(userCache, "jevlint", "packs", "abc123", "database-joins")
	if got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}

func TestLoadDirRejectsManifestTraversal(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"traversing id":    `{"version": 1, "id": "../../outside"}`,
		"absolute rules":   `{"version": 1, "id": "joins", "rules": "/etc/passwd"}`,
		"traversing rules": `{"version": 1, "id": "joins", "rules": "../outside.json"}`,
		"traversing evals": `{"version": 1, "id": "joins", "evals": "../../secret.json"}`,
		"absolute evals":   `{"version": 1, "id": "joins", "evals": "/etc/passwd"}`,
	}
	for name, manifest := range tests {
		name, manifest := name, manifest
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadDir(dir); err == nil {
				t.Fatalf("LoadDir() error = nil, want an error")
			}
		})
	}
}

func TestEvalPathRejectsTraversal(t *testing.T) {
	t.Parallel()

	loaded := Loaded{Manifest: Manifest{Evals: "../../secret.json"}, Dir: t.TempDir()}
	if _, err := loaded.EvalPath(); err == nil {
		t.Fatal("EvalPath() error = nil, want an error")
	}
}

func TestResolveRejectsMaliciousPin(t *testing.T) {
	t.Parallel()

	cache := t.TempDir()
	_, err := Resolve(
		[]config.PackRef{{
			ID:     "../../outside",
			Source: "local",
			SHA:    "abc123",
		}},
		func() (string, error) { return cache, nil },
	)
	if err == nil {
		t.Fatal("Resolve() error = nil, want an error")
	}
}

func TestInstallRejectsManifestIDTraversal(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack install tests")
	}

	repo := writePackRepo(
		t,
		`{"version": 1, "id": "../../outside", "languages": ["go"]}`,
		validRulesJSON,
	)
	cache := t.TempDir()
	// A naive join of "../../outside" under jevlint/packs/<sha> resolves here,
	// so RemoveAll on the destination would delete this directory.
	sentinel := filepath.Join(cache, "jevlint", "outside")
	if err := os.MkdirAll(sentinel, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(sentinel, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Install(repo, func() (string, error) { return cache, nil })
	if err == nil {
		t.Fatal("Install() error = nil, want an error")
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("Install() removed a path outside the pack cache: %v", statErr)
	}
}

func TestInstallRejectsPathTraversal(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack install tests")
	}

	repo := writePackRepo(t, `{"version": 1, "id": "database-joins", "languages": ["go"]}`, validRulesJSON)
	cache := t.TempDir()
	_, err := Install(repo+"#../outside", func() (string, error) { return cache, nil })
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Install() error = %v, want an escape error", err)
	}
}

func writePackRepo(t *testing.T, manifest string, rules string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, defaultRulesFile), []byte(rules), 0o600); err != nil {
		t.Fatal(err)
	}
	packsRunGit(t, dir, "init", "--initial-branch=main")
	packsRunGit(t, dir, "config", "user.email", "jevlint@example.com")
	packsRunGit(t, dir, "config", "user.name", "jevlint")
	packsRunGit(t, dir, "config", "commit.gpgsign", "false")
	packsRunGit(t, dir, "add", ".")
	packsRunGit(t, dir, "commit", "-m", "pack")
	return dir
}

func packsRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
