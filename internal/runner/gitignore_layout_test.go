package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestBroadScanIncludesTrackedSubmoduleAndHonorsItsIgnores(t *testing.T) {
	source := privacyRepo(t)
	writeFile(t, filepath.Join(source, "helper.go"), "package sample\nfunc helper() {}\n")
	writeFile(t, filepath.Join(source, ".gitignore"), "secret.go\n")
	privacyGit(t, source, "add", "helper.go", ".gitignore")
	privacyGit(t, source, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "test")
	root := privacyRepo(t)
	writeFile(t, filepath.Join(root, "ordinary.go"), "package sample\nfunc ordinary() {}\n")
	privacyGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", source, "module")
	writeFile(t, filepath.Join(root, "module/secret.go"), "package sample\nfunc secret() { println(\"PRIVATE_SUBMODULE\") }\n")
	evaluator := &capturingEvaluator{}
	report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{Root: root, Concurrency: 1},
	)
	if err != nil || strings.Join(report.SourcePaths, ",") != "module/helper.go,ordinary.go" {
		t.Fatalf("submodule scan paths = %v, error = %v", report.SourcePaths, err)
	}
	payload, err := json.Marshal(evaluator.batches)
	if err != nil || strings.Contains(string(payload), "PRIVATE_SUBMODULE") {
		t.Fatalf("submodule ignored source reached evaluator: %s, error = %v", payload, err)
	}
}

func TestBroadScanConfinedDirectoryAliasHonorsTargetIgnores(t *testing.T) {
	root := privacyRepo(t)
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".gitignore"), "nested/secret.go\n")
	writeFile(t, filepath.Join(root, "nested/allowed.go"), "package sample\nfunc allowed() {}\n")
	writeFile(t, filepath.Join(root, "nested/secret.go"), "package sample\nfunc secret() { println(\"PRIVATE_ALIAS\") }\n")
	if err := os.Symlink("nested", filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	evaluator := &capturingEvaluator{}
	report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}},
		Options{Root: root, Paths: []string{"linked"}, Concurrency: 1},
	)
	if err != nil || strings.Join(report.SourcePaths, ",") != "linked/allowed.go" {
		t.Fatalf("directory alias scan paths = %v, error = %v", report.SourcePaths, err)
	}
	payload, err := json.Marshal(evaluator.batches)
	if err != nil || strings.Contains(string(payload), "PRIVATE_ALIAS") {
		t.Fatalf("aliased ignored source reached evaluator: %s, error = %v", payload, err)
	}
}

func TestBroadScanParentIgnoreProtectsNestedRepository(t *testing.T) {
	root := privacyRepo(t)
	writeFile(t, filepath.Join(root, ".gitignore"), "private/\n")
	writeFile(t, filepath.Join(root, "ordinary.go"), "package sample\nfunc ordinary() {}\n")
	nested := filepath.Join(root, "private")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	privacyGit(t, nested, "init")
	writeFile(t, filepath.Join(nested, "secret.go"), "package sample\nfunc secret() { println(\"PRIVATE_NESTED_REPO\") }\n")
	evaluator := &capturingEvaluator{}
	report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(), config.Config{Rules: []config.Rule{privacyRule("privacy")}}, Options{Root: root, Concurrency: 1},
	)
	if err != nil || strings.Join(report.SourcePaths, ",") != "ordinary.go" {
		t.Fatalf("nested repository scan paths = %v, error = %v", report.SourcePaths, err)
	}
	payload, err := json.Marshal(evaluator.batches)
	if err != nil || strings.Contains(string(payload), "PRIVATE_NESTED_REPO") {
		t.Fatalf("parent-ignored repository leaked source: %s, error = %v", payload, err)
	}
}
