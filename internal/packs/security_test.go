package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
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

func TestValidatePackID(t *testing.T) {
	t.Parallel()

	valid := []string{"owner/name", "codegirl-007/database-joins", "a.b/c-d_e", "A1/B2"}
	for _, id := range valid {
		if err := validatePackID(id); err != nil {
			t.Fatalf("validatePackID(%q) error = %v", id, err)
		}
	}
	invalid := []string{"", "name", "/name", "owner/", "owner/name/extra", "../x", "a b/c", "owner//name"}
	for _, id := range invalid {
		if err := validatePackID(id); err == nil {
			t.Fatalf("validatePackID(%q) = nil, want an error", id)
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
		{"missing owner", "abc123", "database-joins"},
		{"too many slashes", "abc123", "owner/name/extra"},
		{"empty name", "abc123", "owner/"},
		{"traversing sha", "../../outside", "codegirl-007/database-joins"},
	} {
		if got, err := CacheDir(userCache, test.sha, test.id); err == nil {
			t.Fatalf("CacheDir(%q, %q) = %q, want an error", test.sha, test.id, got)
		}
	}

	got, err := CacheDir(userCache, "abc123", "codegirl-007/database-joins")
	if err != nil {
		t.Fatalf("CacheDir() error = %v", err)
	}
	want := filepath.Join(userCache, "jevlint", "packs", "abc123", "codegirl-007", "database-joins")
	if got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}

func TestLoadDirRejectsManifestTraversal(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"traversing id":    `{"version": 1, "id": "../../outside"}`,
		"absolute rules":   `{"version": 1, "id": "codegirl-007/joins", "rules": "/etc/passwd"}`,
		"traversing rules": `{"version": 1, "id": "codegirl-007/joins", "rules": "../outside.json"}`,
		"traversing evals": `{"version": 1, "id": "codegirl-007/joins", "evals": "../../secret.json"}`,
		"absolute evals":   `{"version": 1, "id": "codegirl-007/joins", "evals": "/etc/passwd"}`,
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

	repo := writePackRepo(t, `{"version": 1, "id": "codegirl-007/database-joins", "languages": ["go"]}`, validRulesJSON)
	cache := t.TempDir()
	_, err := Install(repo+"#../outside", func() (string, error) { return cache, nil })
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("Install() error = %v, want an escape error", err)
	}
}

func TestLoadDirRejectsSymlinks(t *testing.T) {
	t.Parallel()

	const manifestJSON = `{"version": 1, "id": "codegirl-007/joins", "languages": ["go"]}`
	tests := map[string]struct {
		manifest string
		link     string
	}{
		"symlinked manifest": {manifestJSON, ManifestFile},
		"symlinked rules":    {manifestJSON, defaultRulesFile},
		"symlinked evals":    {`{"version": 1, "id": "codegirl-007/joins", "evals": "evals.json"}`, "evals.json"},
		"symlinked fixture":  {manifestJSON, "fixture.go"},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outside := filepath.Join(t.TempDir(), "outside.json")
			if err := os.WriteFile(outside, []byte(validRulesJSON), 0o600); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if test.link != ManifestFile {
				if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(test.manifest), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.link != ManifestFile && test.link != defaultRulesFile {
				if err := os.WriteFile(filepath.Join(dir, defaultRulesFile), []byte(validRulesJSON), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(outside, filepath.Join(dir, test.link)); err != nil {
				t.Fatal(err)
			}

			_, err := LoadDir(dir)
			if err == nil || !strings.Contains(err.Error(), "symbolic link") {
				t.Fatalf("LoadDir() error = %v, want a symbolic link error", err)
			}
		})
	}
}

func TestEvalPathRejectsSymlink(t *testing.T) {
	t.Parallel()

	outside := filepath.Join(t.TempDir(), "secret-evals.json")
	if err := os.WriteFile(outside, []byte(`{"version": 1, "cases": []}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "evals.json")); err != nil {
		t.Fatal(err)
	}
	loaded := Loaded{Manifest: Manifest{Evals: "evals.json"}, Dir: dir}
	if _, err := loaded.EvalPath(); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("EvalPath() error = %v, want a symbolic link error", err)
	}
}

func TestCopyDirRejectsSymlink(t *testing.T) {
	t.Parallel()

	contents := "SYMLINK-TARGET-CONTENTS"
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(source, "a-link.txt")); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "dest")

	err := copyDir(source, dest)
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("copyDir() error = %v, want a symbolic link error", err)
	}
	assertNoFileContains(t, dest, contents)
}

func TestReplaceDirRejectsSymlinkBeforeClearingDest(t *testing.T) {
	t.Parallel()

	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(source, "link.txt")); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "cache")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dest, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := replaceDir(dest, source); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("replaceDir() error = %v, want a symbolic link error", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("replaceDir() cleared the destination before validating: %v", statErr)
	}
}

func TestInstallRejectsSymlinkWithoutCopyingTarget(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack install tests")
	}

	contents := "SYMLINK-TARGET-CONTENTS"
	outsideDir := t.TempDir()
	secret := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secret, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ManifestFile), []byte(`{
		"version": 1,
		"id": "codegirl-007/database-joins",
		"languages": ["go"]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, defaultRulesFile), []byte(validRulesJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(repo, "fixture.go")); err != nil {
		t.Fatal(err)
	}
	packsRunGit(t, repo, "init", "--initial-branch=main")
	packsRunGit(t, repo, "config", "user.email", "jevlint@example.com")
	packsRunGit(t, repo, "config", "user.name", "jevlint")
	packsRunGit(t, repo, "config", "commit.gpgsign", "false")
	packsRunGit(t, repo, "add", ".")
	packsRunGit(t, repo, "commit", "-m", "pack")

	cache := t.TempDir()
	_, err := Install(repo, func() (string, error) { return cache, nil })
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Install() error = %v, want a symbolic link error", err)
	}

	data, readErr := os.ReadFile(secret)
	if readErr != nil || string(data) != contents {
		t.Fatalf("outside file changed: %q, %v", data, readErr)
	}
	entries, readErr := os.ReadDir(outsideDir)
	if readErr != nil || len(entries) != 1 {
		t.Fatalf("outside directory changed: %#v, %v", entries, readErr)
	}
	assertNoFileContains(t, cache, contents)
}

func assertNoFileContains(t *testing.T, root string, needle string) {
	t.Helper()
	// A missing cache directory is a valid outcome (nothing was copied), but
	// any other error means the check did not actually run.
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("stat %s: %v", root, err)
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), needle) {
			t.Fatalf("found symlink target %q in %s", needle, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
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
