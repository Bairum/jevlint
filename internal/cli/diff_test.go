package cli

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/runner"
)

func TestCheckBaseSelectsOnlyOverlappingUnit(t *testing.T) {
	root := initTrunkProject(t)
	writeProjectFile(t, root, "sample.go", "package sample\n\nfunc Alpha() {}\n\nfunc Beta() {}\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "base")
	base := gitRev(t, root, "HEAD")
	writeProjectFile(t, root, "sample.go", "package sample\n\nfunc Alpha() {}\n\nfunc Beta() {\n\tprintln(\"join\")\n}\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "edit beta")
	server := passingJevServer(t)
	t.Setenv("TYPESAFE_API_KEY", "sk-test")
	t.Setenv("TYPESAFE_BASE_URL", server.URL)
	t.Setenv("TYPESAFE_DEFAULT_MODEL", "jev-test")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	report, stderr := runScoped(t, root, "--base", base)
	if stderr != "" && !strings.Contains(stderr, "jevlint:") {
		t.Fatalf("stderr = %q", stderr)
	}
	if report.CodeUnits != 1 || report.Evaluations != 1 || report.ScannedFiles != 1 {
		t.Fatalf("units=%d evals=%d files=%d", report.CodeUnits, report.Evaluations, report.ScannedFiles)
	}
	if report.Scope == nil || report.Scope.Base != base || report.Scope.Units != 1 || report.Scope.Files != 1 || report.Scope.MergeBase == "" {
		t.Fatalf("scope = %#v", report.Scope)
	}
}

func TestCheckChangedMatchesBaseHEAD(t *testing.T) {
	root := initTrunkProject(t)
	writeProjectFile(t, root, "sample.go", "package sample\n\nfunc Alpha() {}\n\nfunc Beta() {}\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "base")
	writeProjectFile(t, root, "sample.go", "package sample\n\nfunc Alpha() {}\n\nfunc Beta() {\n\tprintln(\"join\")\n}\n")
	server := passingJevServer(t)
	t.Setenv("TYPESAFE_API_KEY", "sk-test")
	t.Setenv("TYPESAFE_BASE_URL", server.URL)
	t.Setenv("TYPESAFE_DEFAULT_MODEL", "jev-test")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	changed, _ := runScoped(t, root, "--changed")
	based, _ := runScoped(t, root, "--base", "HEAD")
	if changed.CodeUnits != 1 || based.CodeUnits != changed.CodeUnits || based.Evaluations != changed.Evaluations || based.ScannedFiles != changed.ScannedFiles {
		t.Fatalf("changed units=%d base units=%d", changed.CodeUnits, based.CodeUnits)
	}
	if changed.Scope == nil || based.Scope == nil || changed.Scope.MergeBase != based.Scope.MergeBase || changed.Scope.Units != based.Scope.Units {
		t.Fatalf("scopes changed=%#v base=%#v", changed.Scope, based.Scope)
	}
}

func TestCheckRejectsChangedWithBase(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(context.Background(), []string{"check", "--changed", "--base", "HEAD"}, &stdout, &stderr)
	if exitCode != 2 || !strings.Contains(stderr.String(), "--changed and --base cannot be used together") {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
}

func TestCheckUnknownBaseSuggestsFetch(t *testing.T) {
	root := initTrunkProject(t)
	writeProjectFile(t, root, "sample.go", "package sample\n\nfunc Alpha() {}\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "base")

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(context.Background(), []string{
		"check", "--base", "no-such-ref", "--config", filepath.Join(root, "jevlint.json"),
	}, &stdout, &stderr)
	if exitCode != 2 || !strings.Contains(stderr.String(), "git fetch") {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
}

func initTrunkProject(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required")
	}
	root := t.TempDir()
	writeProjectConfig(t, root)
	runGit(t, root, "init", "--initial-branch=trunk")
	runGit(t, root, "config", "user.email", "jevlint@example.com")
	runGit(t, root, "config", "user.name", "jevlint")
	runGit(t, root, "config", "commit.gpgsign", "false")
	return root
}

func gitRev(t *testing.T, dir string, ref string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", ref)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatalf("rev-parse %s: %v", ref, err)
	}
	return strings.TrimSpace(string(output))
}

func runScoped(t *testing.T, root string, flags ...string) (runner.Report, string) {
	t.Helper()
	args := append([]string{"check"}, flags...)
	args = append(args, "--config", filepath.Join(root, "jevlint.json"), "--format", "json")
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(context.Background(), args, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
	return decodeReport(t, stdout.Bytes()), stderr.String()
}
