package evals

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/codegirl-007/jevlint/internal/evaluation"
)

type fixtureBoundaryEvaluator struct {
	calls atomic.Int32
}

func (evaluator *fixtureBoundaryEvaluator) Evaluate(ctx context.Context, batch evaluation.Batch) (map[string]evaluation.Result, error) {
	evaluator.calls.Add(1)
	return fixedEvaluator{status: evaluation.StatusFail, confidence: 1}.Evaluate(ctx, batch)
}

func TestLoadRejectsFixtureEscapesBeforeEvaluation(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"parent", "nested-parent", "absolute", "file-symlink", "directory-symlink"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			parent := t.TempDir()
			root := filepath.Join(parent, "evals")
			if err := os.MkdirAll(filepath.Join(root, "nested"), 0o700); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(parent, "outside.go")
			if err := os.WriteFile(outside, []byte("package sample\n\nfunc Secret() {}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var fixture string
			switch name {
			case "parent":
				fixture = "../outside.go"
			case "nested-parent":
				fixture = "nested/../../outside.go"
			case "absolute":
				fixture = outside
			case "file-symlink":
				fixture = "linked.go"
				if err := os.Symlink(outside, filepath.Join(root, fixture)); err != nil {
					t.Fatal(err)
				}
			case "directory-symlink":
				fixture = "linked/outside.go"
				if err := os.Symlink(parent, filepath.Join(root, "linked")); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(validEvalsJSON(filepath.ToSlash(fixture))), 0o600); err != nil {
				t.Fatal(err)
			}
			extractor := testExtractor(t)
			document, loadErr := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), extractor)
			evaluator := &fixtureBoundaryEvaluator{}
			if loadErr == nil {
				_, _ = Run(context.Background(), document, sampleConfig(nil), extractor, evaluator, Options{Root: root, Concurrency: 1})
				t.Error("Load() accepted an escaping fixture")
			}
			if calls := evaluator.calls.Load(); calls != 0 {
				t.Fatalf("evaluator called %d times for an escaping fixture", calls)
			}
		})
	}
}

func TestRunRejectsFixtureSymlinkSwap(t *testing.T) {
	t.Parallel()

	root := writeEvalDir(t, validEvalsJSON("sample.go"), "package sample\n\nfunc Ready() {}\n")
	extractor := testExtractor(t)
	document, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), extractor)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.go")
	if err := os.WriteFile(outside, []byte("package sample\n\nfunc Secret() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(root, "sample.go")
	if err := os.Remove(fixture); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, fixture); err != nil {
		t.Fatal(err)
	}
	evaluator := &fixtureBoundaryEvaluator{}
	_, err = Run(context.Background(), document, sampleConfig(nil), extractor, evaluator, Options{Root: root, Concurrency: 1})
	if err == nil {
		t.Error("Run() accepted a fixture replaced with an escaping symlink")
	}
	if calls := evaluator.calls.Load(); calls != 0 {
		t.Fatalf("evaluator called %d times after escaping symlink swap", calls)
	}
}

func TestRunLoadedNestedFixtureUsesDocumentRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "fixtures"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fixtures", "sample.go"), []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("fixtures", filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(validEvalsJSON("linked/sample.go")), 0o600); err != nil {
		t.Fatal(err)
	}
	extractor := testExtractor(t)
	document, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), extractor)
	if err != nil {
		t.Fatal(err)
	}
	document, err = (Document{Version: 1}).Concat(document).FilterRule("database-joins")
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &fixtureBoundaryEvaluator{}
	report, err := Run(context.Background(), document, sampleConfig(nil), extractor, evaluator, Options{Root: t.TempDir(), Concurrency: 1})
	if err != nil {
		t.Fatalf("Run() with separate project root: %v", err)
	}
	if report.Matched != 1 || evaluator.calls.Load() != 1 {
		t.Fatalf("report = %#v; evaluator calls = %d", report, evaluator.calls.Load())
	}
}
