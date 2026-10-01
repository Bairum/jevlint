package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/packs"
)

func TestPluginInitScaffoldsPack(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "my-pack")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		[]string{"plugin", "init", "codegirl-007/my-pack", "--dir", dir},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("exit = %d; stderr = %q", code, stderr.String())
	}
	for _, name := range []string{
		"pack.json",
		"rules.json",
		"README.md",
		"jevlint-evals.json",
		filepath.Join("fixtures", "bad.go"),
		filepath.Join("fixtures", "good.go"),
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}

	loaded, err := packs.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v", err)
	}
	if loaded.Manifest.ID != "codegirl-007/my-pack" || len(loaded.Rules) != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
	if loaded.Rules[0].ID != "my-pack-example" {
		t.Fatalf("rule id = %q", loaded.Rules[0].ID)
	}
}

func TestPluginInitWithoutGoSkipsEvals(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "py-pack")
	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		[]string{"plugin", "init", "codegirl-007/py-pack", "--dir", dir, "--languages", "python"},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("exit = %d; stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "jevlint-evals.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no eval file, stat err = %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifest), "evals") || !strings.Contains(string(manifest), `"python"`) {
		t.Fatalf("pack.json = %s", manifest)
	}
}

func TestPluginInitRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"missing id":       {"plugin", "init"},
		"bare id":          {"plugin", "init", "bad"},
		"unknown language": {"plugin", "init", "owner/pack", "--languages", "bogus"},
	}
	for name, args := range tests {
		name, args := name, args
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			if code := runCLI(context.Background(), args, &stdout, &stderr); code == 0 {
				t.Fatalf("exit = 0, want a usage error")
			}
			if stderr.Len() == 0 {
				t.Fatal("stderr is empty")
			}
		})
	}
}

func TestPluginInitRefusesNonEmptyDirWithoutForce(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		context.Background(),
		[]string{"plugin", "init", "owner/pack", "--dir", dir},
		&stdout,
		&stderr,
	); code == 0 {
		t.Fatal("exit = 0, want a failure for a non-empty directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Fatalf("existing file was removed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCLI(
		context.Background(),
		[]string{"plugin", "init", "owner/pack", "--dir", dir, "--force"},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("--force exit = %d; stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "pack.json")); err != nil {
		t.Fatalf("pack.json missing after --force: %v", err)
	}
}
