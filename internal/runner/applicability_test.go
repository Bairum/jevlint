package runner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

func TestSourceMatchAndTestUnitsControlRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		patterns     []string
		includeTests bool
		want         []string
	}{
		{name: "no matching units", patterns: []string{`\babsent\b`}},
		{name: "matching production unit", patterns: []string{`\babsent\b`, `\bunsafe\b`}, want: []string{"relevant"}},
		{name: "include test units", patterns: []string{`\bunsafe\b`}, includeTests: true, want: []string{"relevant", "check"}},
		{name: "unfiltered skips tests", want: []string{"clean", "relevant"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "source.rs"), "fn clean() {}\nfn relevant() { unsafe {} }\n#[test]\nfn check() { unsafe {} }\n")
			cfg := config.Config{
				Languages: map[string]config.Language{"rust": {}},
				Rules: []config.Rule{{
					ID: "unsafe-subject", Description: "Check unsafe code.", Severity: config.SeverityWarning,
					Kinds: []config.TargetKind{config.TargetKindFunction}, SourceMatch: test.patterns,
					IncludeTests: test.includeTests,
				}},
			}
			evaluator := &capturingEvaluator{}
			report, err := (Runner{Extractor: testExtractor(t, "rust"), Evaluator: evaluator}).Evaluate(
				context.Background(), cfg, Options{Root: root, Concurrency: 1},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(evaluator.batches) != len(test.want) || report.Evaluations != len(test.want) {
				t.Fatalf("requests = %d, evaluations = %d, want %d", len(evaluator.batches), report.Evaluations, len(test.want))
			}
			for index, name := range test.want {
				if evaluator.batches[index].CodeUnit.Name != name {
					t.Fatalf("request %d unit = %q, want %q", index, evaluator.batches[index].CodeUnit.Name, name)
				}
			}
		})
	}
}

func TestJobsGroupIdenticalTypeAndCalleeContext(t *testing.T) {
	t.Parallel()

	unit := parsing.CodeUnit{
		Kind: parsing.CodeKindFunction, Name: "primary", Source: "fn primary() {}",
		RelatedTypes:   []parsing.TypeDeclaration{{Name: "Local", Path: "local.rs", Source: "struct Local;"}},
		TypeCandidates: []parsing.TypeDeclaration{{Name: "Extra", Path: "extra.rs", Source: "struct Extra;"}},
		Resolved:       []parsing.CalleeContext{{Name: "callee", Path: "callee.rs", Source: "fn callee() {}"}},
	}
	rules := []config.Rule{
		{ID: "types-a", Context: config.RuleContext{Types: true}},
		{ID: "types-b", Context: config.RuleContext{Types: true}},
		{ID: "both", Context: config.RuleContext{Types: true, Callees: true}},
		{ID: "restricted", Context: config.RuleContext{Types: true}, Exclude: []string{"extra.rs"}},
		{ID: "callees", Context: config.RuleContext{Callees: true}},
	}
	compiled, err := compileSourceMatches(rules)
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := jobsForUnits([]parsing.CodeUnit{unit}, rules, compiled)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 4 || len(jobs[0].rules) != 2 {
		t.Fatalf("jobs = %#v", jobs)
	}
	for _, job := range jobs {
		for _, rule := range job.rules {
			wantTypes := 2
			if rule.ID == "restricted" || rule.ID == "callees" {
				wantTypes = 1
			}
			if len(job.unit.RelatedTypes) != wantTypes || job.unit.RelatedTypes[0].Name != "Local" {
				t.Fatalf("%s type context = %#v", rule.ID, job.unit.RelatedTypes)
			}
			if got := len(job.unit.Callees) > 0; got != rule.Context.Callees {
				t.Fatalf("%s callee context = %#v", rule.ID, job.unit.Callees)
			}
		}
	}
	if len(unit.RelatedTypes) != 1 {
		t.Fatalf("planning changed original related types: %#v", unit.RelatedTypes)
	}
}
