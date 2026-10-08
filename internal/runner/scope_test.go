package runner

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestDiffScopeSkipsUntouchedUnitsAndKeepsCallees(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "caller.go"), "package sample\n\nfunc Caller() {\n\tloadUsers()\n}\n\nfunc Untouched() {}\n")
	writeFile(t, filepath.Join(root, "users.go"), "package sample\n\nfunc loadUsers() {}\n")
	evaluator := &capturingEvaluator{}
	report, err := (Runner{Extractor: testGoExtractor(t), Evaluator: evaluator}).Evaluate(
		context.Background(),
		config.Config{Rules: []config.Rule{{
			ID:          "database-joins",
			Description: "Join records in the database.",
			Kinds:       []config.TargetKind{config.TargetKindFunction},
			Context:     config.RuleContext{Callees: true},
		}}},
		Options{
			Root:        root,
			Concurrency: 1,
			Diff: &DiffScope{
				Base:      "HEAD",
				MergeBase: "abc",
				Includes: func(path string, start, end uint) bool {
					return path == "caller.go" && start <= 3 && end >= 3
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.CodeUnits != 1 || report.ScannedFiles != 1 || report.Scope == nil ||
		report.Scope.Units != 1 || report.Scope.Files != 1 || report.Scope.Base != "HEAD" {
		t.Fatalf("report scope = %#v units=%d files=%d", report.Scope, report.CodeUnits, report.ScannedFiles)
	}
	if len(evaluator.batches) != 1 || evaluator.batches[0].CodeUnit.Name != "Caller" {
		t.Fatalf("batches = %#v", evaluator.batches)
	}
	if got := strings.Join(calleeNames(evaluator.batches[0].CodeUnit.Callees), ","); got != "loadUsers" {
		t.Fatalf("callees = %q", got)
	}
}
