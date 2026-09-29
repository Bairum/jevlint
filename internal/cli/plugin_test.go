package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"jevlint/internal/config"
	"jevlint/internal/evals"
)

func TestPluginInstallWritesPin(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for plugin install tests")
	}

	packRepo := writeGitPack(t)
	root := t.TempDir()
	writeBareProject(t, root)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"plugin", "install",
			"--config", filepath.Join(root, "jevlint.json"),
			packRepo,
		},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("install exit = %d; stderr = %q", exitCode, stderr.String())
	}

	cfg, err := config.Load(filepath.Join(root, "jevlint.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Packs) != 1 || cfg.Packs[0].ID != "database-joins" || cfg.Packs[0].SHA == "" {
		t.Fatalf("packs = %#v", cfg.Packs)
	}
}

func TestPluginUpdateRejectsChangedPackID(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for plugin update tests")
	}

	packRepo := writeGitPack(t)
	root := t.TempDir()
	writeBareProject(t, root)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	configPath := filepath.Join(root, "jevlint.json")

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		context.Background(),
		[]string{"plugin", "install", "--config", configPath, packRepo},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("install exit = %d; stderr = %q", code, stderr.String())
	}
	before, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Packs) != 1 {
		t.Fatalf("packs = %#v", before.Packs)
	}
	oldSHA := before.Packs[0].SHA

	if err := os.WriteFile(filepath.Join(packRepo, "pack.json"), []byte(`{
		"version": 1,
		"id": "sql-database-joins",
		"languages": ["go"]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, packRepo, "add", ".")
	runGit(t, packRepo, "commit", "-m", "rename pack")

	stdout.Reset()
	stderr.Reset()
	code := runCLI(
		context.Background(),
		[]string{"plugin", "update", "--config", configPath},
		&stdout,
		&stderr,
	)
	if code == 0 {
		t.Fatalf("update exit = 0, want failure; stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "changed its id") {
		t.Fatalf("stderr = %q, want a pack id change error", stderr.String())
	}

	after, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Packs) != 1 ||
		after.Packs[0].ID != "database-joins" ||
		after.Packs[0].SHA != oldSHA {
		t.Fatalf("config changed: %#v", after.Packs)
	}
}

func TestCheckUsesPackRuleAndIgnoresPackEvals(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for plugin install tests")
	}

	packRepo := writeGitPack(t)
	root := t.TempDir()
	writeBareProject(t, root)
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		context.Background(),
		[]string{"plugin", "install", "--config", filepath.Join(root, "jevlint.json"), packRepo},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("install exit = %d; stderr = %q", code, stderr.String())
	}

	cfg, err := config.Load(filepath.Join(root, "jevlint.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Rules = nil
	if err := config.Write(filepath.Join(root, "jevlint.json"), cfg); err != nil {
		t.Fatal(err)
	}

	server := passingRulesServer(t, "database-joins")
	defer server.Close()
	t.Setenv("TYPESAFE_API_KEY", "sk-test")
	t.Setenv("TYPESAFE_BASE_URL", server.URL)
	t.Setenv("TYPESAFE_DEFAULT_MODEL", "jev-test")

	stdout.Reset()
	stderr.Reset()
	exitCode := runCLI(
		context.Background(),
		[]string{"check", "--config", filepath.Join(root, "jevlint.json"), "."},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("check exit = %d; stderr = %q stdout = %q", exitCode, stderr.String(), stdout.String())
	}
}

func TestEvalPacksFlagRunsPackEvals(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for plugin install tests")
	}

	packRepo := writeGitPack(t)
	root := t.TempDir()
	writeBareProject(t, root)
	if err := os.WriteFile(filepath.Join(root, evals.DefaultFile), []byte(`{
		"version": 1,
		"cases": [{"rule": "local-rule", "file": "sample.go", "expect": "pass"}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	if code := runCLI(
		context.Background(),
		[]string{"plugin", "install", "--config", filepath.Join(root, "jevlint.json"), packRepo},
		&stdout,
		&stderr,
	); code != 0 {
		t.Fatalf("install exit = %d; stderr = %q", code, stderr.String())
	}

	server := passingRulesServer(t, "local-rule", "database-joins")
	defer server.Close()
	t.Setenv("TYPESAFE_API_KEY", "sk-test")
	t.Setenv("TYPESAFE_BASE_URL", server.URL)
	t.Setenv("TYPESAFE_DEFAULT_MODEL", "jev-test")

	stdout.Reset()
	stderr.Reset()
	withoutPacks := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"},
		&stdout,
		&stderr,
	)
	if withoutPacks != 0 {
		t.Fatalf("eval exit = %d; stderr = %q", withoutPacks, stderr.String())
	}
	var report evals.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Total != 1 {
		t.Fatalf("eval without --packs total = %d", report.Total)
	}

	stdout.Reset()
	stderr.Reset()
	withPacks := runCLI(
		context.Background(),
		[]string{"eval", "--packs", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"},
		&stdout,
		&stderr,
	)
	if withPacks != 0 {
		t.Fatalf("eval --packs exit = %d; stderr = %q stdout = %q", withPacks, stderr.String(), stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 {
		t.Fatalf("eval --packs total = %d, want 2; stdout = %s", report.Total, stdout.String())
	}
}

func writeBareProject(t *testing.T, root string) {
	t.Helper()
	config := `{
		"languages": {"go": {}},
		"rules": [{
			"id": "local-rule",
			"description": "A local rule.",
			"severity": "info"
		}]
	}`
	if err := os.WriteFile(filepath.Join(root, "jevlint.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeGitPack(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(`{
		"version": 1,
		"id": "database-joins",
		"languages": ["go"]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rules.json"), []byte(`{
		"rules": [{
			"id": "database-joins",
			"description": "Join related records in the database.",
			"severity": "error"
		}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, evals.DefaultFile), []byte(`{
		"version": 1,
		"cases": [{"rule": "database-joins", "file": "fixture.go", "expect": "pass"}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init", "--initial-branch=main")
	runGit(t, dir, "config", "user.email", "jevlint@example.com")
	runGit(t, dir, "config", "user.name", "jevlint")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "pack")
	return dir
}

func passingRulesServer(t *testing.T, ids ...string) *httptest.Server {
	t.Helper()
	answers := make([]string, 0, len(ids))
	for _, id := range ids {
		answers = append(answers, fmt.Sprintf(
			`%q: {"type":"choice","choice":"pass","confidence":1}`,
			id,
		))
	}
	body := fmt.Sprintf(`{"model":"jev-test","answers":{%s}}`, strings.Join(answers, ","))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestEvalDefaultFileIsJeVlintEvalsJSON(t *testing.T) {
	t.Parallel()
	if evals.DefaultFile != "jevlint-evals.json" {
		t.Fatalf("DefaultFile = %q", evals.DefaultFile)
	}
	if !strings.Contains(usage, "jevlint-evals.json") {
		t.Fatalf("usage missing jevlint-evals.json: %s", usage)
	}
}
