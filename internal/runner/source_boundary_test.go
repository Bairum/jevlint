package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestRunnerRejectsSourcesOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.go"), "package secret\nfunc ExternalSecret() {}\n")
	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(root, "link.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(root, filepath.Join(outside, "secret.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"link.go", "linked/secret.go", "linked", relative, filepath.Join(outside, "secret.go")} {
		t.Run(path, func(t *testing.T) {
			evaluator := &capturingEvaluator{}
			_, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
				context.Background(), sourceBoundaryConfig(), Options{Root: root, Paths: []string{path}, Concurrency: 1},
			)
			if len(evaluator.batches) != 0 {
				t.Fatalf("external source reached evaluator: %#v", evaluator.batches)
			}
			if err == nil {
				t.Fatal("outside-root source was not rejected")
			}
		})
	}
}

func TestRunnerConfinesReadsAfterDiscovery(t *testing.T) {
	for _, swapDirectory := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[swapDirectory], func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(root, "nested", "sample.go"), "package sample\nfunc Internal() {}\n")
			writeFile(t, filepath.Join(outside, "sample.go"), "package secret\nfunc ExternalSecret() {}\n")
			boundary, err := os.OpenRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer boundary.Close()
			runner := Runner{Extractor: testGoExtractor(t)}
			files, err := discover(context.Background(), boundary, []string{"."}, runner.Extractor)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "nested", "sample.go")
			target := filepath.Join(outside, "sample.go")
			if swapDirectory {
				path = filepath.Join(root, "nested")
				target = outside
			}
			if err := os.Rename(path, path+".original"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			_, jobs, err := runner.planEvaluations(context.Background(), sourceBoundaryConfig(), boundary, files, nil, nil)
			if len(jobs) != 0 {
				t.Fatalf("replaced external source became an evaluation job: %#v", jobs)
			}
			if err == nil {
				t.Fatal("escaping replacement was not rejected")
			}
		})
	}
}

func TestRunnerPreservesConfinedSourcesAndOverlays(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "nested", "sample.go"), "package sample\nfunc Internal() {}\n")
	if err := os.Symlink("nested/sample.go", filepath.Join(root, "link.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nested", filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"nested/sample.go", filepath.Join(root, "nested", "sample.go"), "link.go", "linked"} {
		t.Run(path, func(t *testing.T) {
			evaluator := &capturingEvaluator{}
			report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
				context.Background(), sourceBoundaryConfig(), Options{Root: root, Paths: []string{path}, Concurrency: 1},
			)
			if err != nil {
				t.Fatal(err)
			}
			if report.ScannedFiles != 1 || len(evaluator.batches) != 1 || evaluator.batches[0].CodeUnit.Name != "Internal" {
				t.Fatalf("confined source was not evaluated: %#v, %#v", report, evaluator.batches)
			}
		})
	}
	evaluator := &capturingEvaluator{}
	_, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(), sourceBoundaryConfig(), Options{
			Root: root, Paths: []string{"nested/sample.go"}, Concurrency: 1,
			SourceOverlay: map[string][]byte{"nested/sample.go": []byte("package sample\nfunc Overlay() {}\n")},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluator.batches) != 1 || evaluator.batches[0].CodeUnit.Name != "Overlay" {
		t.Fatalf("overlay was not preserved: %#v", evaluator.batches)
	}
}

func TestRunnerConfinesCalleeContextReads(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(root, "caller.go"), "package sample\nfunc Caller() { helper() }\n")
	writeFile(t, filepath.Join(root, "helper.go"), "package sample\nfunc helper() {}\n")
	writeFile(t, filepath.Join(outside, "helper.go"), "package sample\nfunc helper() { println(\"ExternalSecret\") }\n")
	boundary, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer boundary.Close()
	runner := Runner{Extractor: testGoExtractor(t)}
	files, err := discover(context.Background(), boundary, []string{"."}, runner.Extractor)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "helper.go")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "helper.go"), path); err != nil {
		t.Fatal(err)
	}
	cfg := sourceBoundaryConfig()
	cfg.Rules[0].Include = []string{"caller.go"}
	cfg.Rules[0].Context.Callees = true
	evaluator := &capturingEvaluator{}
	_, jobs, err := runner.planEvaluations(context.Background(), cfg, boundary, files, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		if _, err := evaluateJob(context.Background(), evaluator, job, cfg); err != nil {
			t.Fatal(err)
		}
	}
	if len(evaluator.batches) != 1 || evaluator.batches[0].CodeUnit.Name != "Caller" {
		t.Fatalf("confined caller was not evaluated: %#v", evaluator.batches)
	}
	if len(evaluator.batches[0].CodeUnit.Callees) != 0 {
		t.Fatalf("external callee reached evaluator: %#v", evaluator.batches[0].CodeUnit.Callees)
	}
}

func sourceBoundaryConfig() config.Config {
	return config.Config{Rules: []config.Rule{{
		ID: "naming", Description: "Names describe the work.", Severity: config.SeverityWarning,
		Kinds: []config.TargetKind{config.TargetKindFunction},
	}}}
}
