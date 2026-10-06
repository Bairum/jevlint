package runner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func privacyRule(id string) config.Rule {
	return config.Rule{
		ID:          id,
		Description: "Check source privacy.",
		Severity:    config.SeverityWarning,
		Kinds:       []config.TargetKind{config.TargetKindFunction},
		Context:     config.RuleContext{Callees: true},
	}
}

func privacyGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func privacyRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	root := t.TempDir()
	privacyGit(t, root, "init")
	return root
}

func TestBroadScanHonorsGitIgnoresBeforeCalleeIndexing(t *testing.T) {
	root := privacyRepo(t)
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".gitignore"), "ignored.go\ntracked.go\nnested/*.go\n")
	writeFile(t, filepath.Join(root, "nested/.gitignore"), "!allowed.go\n")
	writeFile(t, filepath.Join(root, ".git/info/exclude"), "local.go\n")
	global := filepath.Join(t.TempDir(), "ignore")
	writeFile(t, global, "global.go\n")
	privacyGit(t, root, "config", "core.excludesFile", global)
	for _, name := range []string{"ignored.go", "local.go", "global.go", "nested/hidden.go"} {
		writeFile(t, filepath.Join(root, name), "package sample\nfunc secret() { println(\"PRIVATE_IGNORED_SOURCE\") }\n")
	}
	writeFile(t, filepath.Join(root, "caller.go"), "package sample\nfunc caller() { secret(); tracked() }\n")
	writeFile(t, filepath.Join(root, "tracked.go"), "package sample\nfunc tracked() {}\n")
	writeFile(t, filepath.Join(root, "nested/allowed.go"), "package sample\nfunc allowed() {}\n")
	privacyGit(t, root, "add", "-f", "tracked.go")
	evaluator := &capturingEvaluator{}
	runner := Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}
	report, err := runner.Evaluate(context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{
		Root: root, Concurrency: 1,
		SourceOverlay: map[string][]byte{"ignored.go": []byte("package sample\nfunc secret() { println(\"PRIVATE_OVERLAY\") }\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(report.SourcePaths, ","); got != "caller.go,nested/allowed.go,tracked.go" {
		t.Fatalf("scanned paths = %s", got)
	}
	payload, err := json.Marshal(evaluator.batches)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "PRIVATE_") {
		t.Fatalf("ignored source reached evaluator: %s", payload)
	}
	caller, ok := batchFor(evaluator.batches, "caller", "privacy")
	if !ok || strings.Join(calleeNames(caller.CodeUnit.Callees), ",") != "tracked" {
		t.Fatalf("tracked callee missing: %#v", caller)
	}
	evaluator.batches = nil
	_, err = runner.Evaluate(context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{
		Root: root, Paths: []string{".", "ignored.go"}, Concurrency: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err = json.Marshal(evaluator.batches)
	if err != nil || !strings.Contains(string(payload), "PRIVATE_IGNORED_SOURCE") {
		t.Fatalf("explicit ignored file was not evaluated: %s, %v", payload, err)
	}
}

func TestBroadScanGitWorktreeSubproject(t *testing.T) {
	root := privacyRepo(t)
	writeFile(t, filepath.Join(root, "tracked.go"), "package sample\nfunc tracked() {}\n")
	privacyGit(t, root, "add", "tracked.go")
	privacyGit(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "test")
	worktree := filepath.Join(t.TempDir(), "worktree")
	privacyGit(t, root, "worktree", "add", "--detach", worktree)
	subproject := filepath.Join(worktree, "project")
	if err := os.Mkdir(subproject, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(worktree, ".gitignore"), "project/secret.go\n")
	writeFile(t, filepath.Join(subproject, "secret.go"), "package sample\nfunc secret() {}\n")
	writeFile(t, filepath.Join(subproject, "allowed.go"), "package sample\nfunc allowed() {}\n")
	report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: &capturingEvaluator{}}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{Root: subproject, Concurrency: 1},
	)
	if err != nil || strings.Join(report.SourcePaths, ",") != "allowed.go" {
		t.Fatalf("subproject paths = %v, error = %v", report.SourcePaths, err)
	}
}

func TestBroadScanGitFailureDoesNotFallBack(t *testing.T) {
	for _, mode := range []string{"missing", "broken"} {
		t.Run(mode, func(t *testing.T) {
			root := privacyRepo(t)
			writeFile(t, filepath.Join(root, ".gitignore"), "secret.go\n")
			writeFile(t, filepath.Join(root, "secret.go"), "package sample\nfunc secret() {}\n")
			if mode == "missing" {
				t.Setenv("PATH", t.TempDir())
			} else {
				writeFile(t, filepath.Join(root, ".git/HEAD"), "invalid metadata\n")
			}
			evaluator := &capturingEvaluator{}
			_, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
				context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{Root: root, Concurrency: 1},
			)
			if err == nil || len(evaluator.batches) != 0 {
				t.Fatalf("Git failure leaked source: error = %v, batches = %#v", err, evaluator.batches)
			}
		})
	}
}

func TestBroadScanNonGitRetainsFilesystemDiscovery(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "allowed.go\n")
	writeFile(t, filepath.Join(root, "allowed.go"), "package sample\nfunc allowed() {}\n")
	t.Setenv("PATH", t.TempDir())
	report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: &capturingEvaluator{}}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{Root: root, Concurrency: 1},
	)
	if err != nil || strings.Join(report.SourcePaths, ",") != "allowed.go" {
		t.Fatalf("non-Git paths = %v, error = %v", report.SourcePaths, err)
	}
}

func TestRuleExcludesCalleeSourceInMixedBatchesAndLocalization(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "caller.go"), "package sample\nfunc caller() { secret(); allowed() }\n")
	writeFile(t, filepath.Join(root, "secret.go"), "package sample\nfunc secret() { println(\"PRIVATE_CALLEE\") }\n")
	writeFile(t, filepath.Join(root, "allowed.go"), "package sample\nfunc allowed() { println(\"ALLOWED_CALLEE\") }\n")
	restricted := privacyRule("restricted")
	restricted.Include = []string{"caller.go"}
	restricted.Exclude = []string{"secret.go"}
	restricted.Localize = []config.TargetKind{config.TargetKindStatement}
	equivalent := restricted
	equivalent.ID = "equivalent"
	equivalent.Exclude = []string{"**/secret.go", "absent/**"}
	permitted := restricted
	permitted.ID = "permitted"
	permitted.Exclude = nil
	ordinary := restricted
	ordinary.ID = "ordinary"
	ordinary.Context.Callees = false
	evaluator := &capturingEvaluator{failFunctions: true}
	_, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{restricted, equivalent, permitted, ordinary}}, Options{Root: root, Concurrency: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	localized := false
	for _, batch := range evaluator.batches {
		payload, err := json.Marshal(batch.CodeUnit)
		if err != nil {
			t.Fatal(err)
		}
		for _, rule := range batch.Rules {
			switch rule.ID {
			case "restricted", "equivalent":
				if strings.Contains(string(payload), "PRIVATE_CALLEE") || !strings.Contains(string(payload), "ALLOWED_CALLEE") {
					t.Fatalf("excluded source or missing allowed context in %s: %s", rule.ID, payload)
				}
				localized = localized || len(batch.Regions) > 0
			case "permitted":
				if !strings.Contains(string(payload), "PRIVATE_CALLEE") || !strings.Contains(string(payload), "ALLOWED_CALLEE") {
					t.Fatalf("permitted callee context missing: %s", payload)
				}
			case "ordinary":
				if len(batch.CodeUnit.Callees) != 0 {
					t.Fatal("ordinary rule received callee context")
				}
			}
		}
	}
	batch, ok := batchFor(evaluator.batches, "caller", "restricted")
	if _, grouped := ruleIDs(batch)["equivalent"]; !ok || !grouped || !localized {
		t.Fatalf("equivalent context batching or localization missing: %#v", evaluator.batches)
	}
}

func TestRuleExcludedCalleeDoesNotConsumeContextQuota(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "caller.go"), "package sample\nfunc caller() { secret(); allowed() }\n")
	writeFile(t, filepath.Join(root, "secret.go"), "package sample\nfunc secret() { println(\""+strings.Repeat("s", 15000)+"\") }\n")
	writeFile(t, filepath.Join(root, "allowed.go"), "package sample\nfunc allowed() { println(\""+strings.Repeat("a", 2000)+"\") }\n")
	rule := privacyRule("restricted")
	rule.Include = []string{"caller.go"}
	rule.Exclude = []string{"secret.go"}
	evaluator := &capturingEvaluator{}
	_, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{rule}}, Options{Root: root, Concurrency: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	batch, ok := batchFor(evaluator.batches, "caller", "restricted")
	if !ok || strings.Join(calleeNames(batch.CodeUnit.Callees), ",") != "allowed" {
		t.Fatalf("excluded callee consumed context quota: %#v", batch.CodeUnit.Callees)
	}
}

func TestExplicitIgnoredFileDoesNotRequireGit(t *testing.T) {
	root := privacyRepo(t)
	writeFile(t, filepath.Join(root, ".gitignore"), "secret.go\n")
	writeFile(t, filepath.Join(root, "secret.go"), "package sample\nfunc secret() {}\n")
	t.Setenv("PATH", t.TempDir())
	for _, paths := range [][]string{{"secret.go"}, {".", "secret.go"}} {
		report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: &capturingEvaluator{}}).Evaluate(
			context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}},
			Options{Root: root, Paths: paths, Concurrency: 1},
		)
		if err != nil || strings.Join(report.SourcePaths, ",") != "secret.go" {
			t.Fatalf("explicit paths %v: scanned %v, error = %v", paths, report.SourcePaths, err)
		}
	}
}
