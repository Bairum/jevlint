package evals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"jevlint/internal/config"
	"jevlint/internal/evaluation"
	"jevlint/internal/parsing"
)

type fixedEvaluator struct {
	status     evaluation.Status
	confidence float64
}

func TestLoadRejectsUnknownRule(t *testing.T) {
	t.Parallel()

	root := writeEvalDir(t, `{
		"version": 1,
		"cases": [{"rule": "missing-rule", "file": "sample.go", "expect": "fail"}]
	}`, "package sample\n\nfunc Ready() {}\n")
	_, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err == nil || !strings.Contains(err.Error(), "unknown eval rule") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsMissingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(validEvalsJSON("missing.go")), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err == nil || !strings.Contains(err.Error(), "eval fixture") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(validEvalsJSON("notes.txt")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err == nil || !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsInvalidExpect(t *testing.T) {
	t.Parallel()

	root := writeEvalDir(t, `{
		"version": 1,
		"cases": [{
			"rule": "database-joins",
			"file": "sample.go",
			"expect": "maybe"
		}]
	}`, "package sample\n\nfunc Ready() {}\n")
	_, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err == nil || !strings.Contains(err.Error(), "expect must be") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsDuplicateRuleAndFile(t *testing.T) {
	t.Parallel()

	root := writeEvalDir(t, `{
		"version": 1,
		"cases": [
			{"name": "first", "rule": "database-joins", "file": "sample.go", "expect": "pass"},
			{"name": "second", "rule": "database-joins", "file": "./sample.go", "expect": "fail"}
		]
	}`, "package sample\n\nfunc Ready() {}\n")
	_, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err == nil || !strings.Contains(err.Error(), "duplicate eval case") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadResolvesFixturesRelativeToEvalFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	casesDir := filepath.Join(root, "cases")
	if err := os.Mkdir(casesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(casesDir, DefaultFile), []byte(validEvalsJSON("join.go")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(casesDir, "join.go"), []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := Load(filepath.Join(casesDir, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if document.Cases[0].AbsolutePath() != filepath.Join(casesDir, "join.go") {
		t.Fatalf("abs = %q", document.Cases[0].AbsolutePath())
	}
}

func TestRunPassAndFailCases(t *testing.T) {
	t.Parallel()

	root := writeEvalDir(t, `{
		"version": 1,
		"cases": [
			{"name": "should-pass", "rule": "database-joins", "file": "sample.go", "expect": "pass"},
			{"name": "should-fail", "rule": "database-joins", "file": "other.go", "expect": "fail"}
		]
	}`, "package sample\n\nfunc Ready() {}\n")
	if err := os.WriteFile(filepath.Join(root, "other.go"), []byte("package sample\n\nfunc Other() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	passReport, err := Run(
		context.Background(),
		document,
		sampleConfig(nil),
		testExtractor(t),
		fixedEvaluator{status: evaluation.StatusPass, confidence: 0.9},
		Options{Root: root, Concurrency: 1},
	)
	if err != nil {
		t.Fatalf("Run() pass error = %v", err)
	}
	if passReport.Matched != 1 || passReport.Mismatched != 1 {
		t.Fatalf("pass report = %#v", passReport)
	}
	if passReport.Cases[0].Confidence != nil {
		t.Fatalf("pass confidence = %v", passReport.Cases[0].Confidence)
	}

	failReport, err := Run(
		context.Background(),
		document,
		sampleConfig(nil),
		testExtractor(t),
		fixedEvaluator{status: evaluation.StatusFail, confidence: 0.91},
		Options{Root: root, Concurrency: 1},
	)
	if err != nil {
		t.Fatalf("Run() fail error = %v", err)
	}
	if failReport.Matched != 1 || failReport.Mismatched != 1 {
		t.Fatalf("fail report = %#v", failReport)
	}
	if failReport.Cases[1].Confidence == nil || *failReport.Cases[1].Confidence != 0.91 {
		t.Fatalf("fail confidence = %v", failReport.Cases[1].Confidence)
	}
}

func TestRunEvaluatesExcludedFixture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fixtures := filepath.Join(root, "fixtures")
	if err := os.Mkdir(fixtures, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(validEvalsJSON("fixtures/sample.go")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtures, "sample.go"), []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := sampleConfig([]string{"fixtures/**"})
	document, err := Load(filepath.Join(root, DefaultFile), cfg, testExtractor(t))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	report, err := Run(
		context.Background(),
		document,
		cfg,
		testExtractor(t),
		fixedEvaluator{status: evaluation.StatusFail, confidence: 1},
		Options{Root: root, Concurrency: 1},
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Cases[0].Matched || report.Cases[0].Actual != OutcomeFail {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunRequiresApplicableUnits(t *testing.T) {
	t.Parallel()

	root := writeEvalDir(t, validEvalsJSON("sample.go"), "package sample\n")
	document, err := Load(filepath.Join(root, DefaultFile), sampleConfig(nil), testExtractor(t))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	_, err = Run(
		context.Background(),
		document,
		sampleConfig(nil),
		testExtractor(t),
		fixedEvaluator{status: evaluation.StatusPass, confidence: 1},
		Options{Root: root, Concurrency: 1},
	)
	if !errors.Is(err, ErrNoApplicableUnits) {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestResultJSONIncludesConfidence(t *testing.T) {
	t.Parallel()

	confidence := 0.88
	data, err := json.Marshal(Result{
		Rule:       "database-joins",
		File:       "sample.go",
		Expected:   ExpectFail,
		Actual:     OutcomeFail,
		Matched:    true,
		Confidence: &confidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"confidence":0.88`) {
		t.Fatalf("json = %s", data)
	}
}

func (evaluator fixedEvaluator) Evaluate(
	_ context.Context,
	batch evaluation.Batch,
) (map[string]evaluation.Result, error) {
	results := make(map[string]evaluation.Result, len(batch.Rules))
	for _, rule := range batch.Rules {
		results[rule.ID] = evaluation.Result{
			Status:     evaluator.status,
			Confidence: evaluator.confidence,
		}
	}
	return results, nil
}

func sampleConfig(exclude []string) config.Config {
	return config.Config{
		Languages: map[string]config.Language{"go": {}},
		Rules: []config.Rule{{
			ID:          "database-joins",
			Description: "Join related records in the database.",
			Severity:    config.SeverityError,
			Include:     []string{"src/**/*.go"},
			Exclude:     exclude,
		}},
	}
}

func testExtractor(t *testing.T) *parsing.Extractor {
	t.Helper()
	extractor, err := parsing.NewExtractor(map[string]config.Language{"go": {}})
	if err != nil {
		t.Fatalf("NewExtractor() error = %v", err)
	}
	return extractor
}

func writeEvalDir(t *testing.T, evalsJSON string, source string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(evalsJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func validEvalsJSON(file string) string {
	return `{
		"version": 1,
		"cases": [{
			"rule": "database-joins",
			"file": "` + file + `",
			"expect": "fail"
		}]
	}`
}

// oneCase builds a document with a single case for sample.go.
func oneCase(t *testing.T, expect Expect) (Document, string) {
	t.Helper()

	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	if err := os.WriteFile(path, []byte("package sample\n\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	document := Document{Version: currentVersion, Cases: []Case{{
		Rule:   "database-joins",
		File:   "sample.go",
		Expect: expect,
		abs:    path,
	}}}
	return document, root
}

func TestRunClassifiesRawDecisions(t *testing.T) {
	t.Parallel()

	floor := 0.8
	tests := []struct {
		name         string
		expect       Expect
		status       evaluation.Status
		confidence   float64
		want         Outcome
		matched      bool
		inconclusive bool
	}{
		{"explicit pass matches pass", ExpectPass, evaluation.StatusPass, 0.9, OutcomePass, true, false},
		{"reported fail mismatches pass", ExpectPass, evaluation.StatusFail, 0.9, OutcomeFail, false, false},
		{"hidden fail is inconclusive", ExpectPass, evaluation.StatusFail, 0.4, OutcomeInconclusive, false, true},
		{"skip is inconclusive", ExpectPass, evaluation.StatusSkip, 0.9, OutcomeInconclusive, false, true},
		{"abstain is inconclusive", ExpectPass, evaluation.StatusAbstain, 0.9, OutcomeInconclusive, false, true},
		{"reported fail matches fail", ExpectFail, evaluation.StatusFail, 0.9, OutcomeFail, true, false},
		{"hidden fail cannot match fail", ExpectFail, evaluation.StatusFail, 0.4, OutcomeInconclusive, false, true},
		{"pass mismatches fail", ExpectFail, evaluation.StatusPass, 0.9, OutcomePass, false, false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			document, root := oneCase(t, test.expect)
			cfg := sampleConfig(nil)
			cfg.MinConfidence = &floor
			report, err := Run(
				context.Background(),
				document,
				cfg,
				testExtractor(t),
				fixedEvaluator{status: test.status, confidence: test.confidence},
				Options{Root: root, Concurrency: 1},
			)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			result := report.Cases[0]
			if result.Actual != test.want {
				t.Fatalf("actual = %v, want %v", result.Actual, test.want)
			}
			if result.Matched != test.matched {
				t.Fatalf("matched = %v, want %v", result.Matched, test.matched)
			}
			if (report.Inconclusive == 1) != test.inconclusive {
				t.Fatalf("inconclusive = %d, want %v", report.Inconclusive, test.inconclusive)
			}
			if report.Matched+report.Mismatched+report.Inconclusive != report.Total {
				t.Fatalf("report counts = %#v", report)
			}
			if result.Floor != floor {
				t.Fatalf("floor = %v, want %v", result.Floor, floor)
			}
			if len(result.Units) != 1 {
				t.Fatalf("units = %#v", result.Units)
			}
			unit := result.Units[0]
			wantReported := test.status == evaluation.StatusFail && test.confidence >= floor
			if unit.Reported != wantReported {
				t.Fatalf("unit reported = %v, want %v", unit.Reported, wantReported)
			}
			if unit.Status != test.status || unit.Confidence != test.confidence {
				t.Fatalf("unit = %#v", unit)
			}
		})
	}
}

type perUnitEvaluator struct {
	results map[string]evaluation.Result
}

func (evaluator perUnitEvaluator) Evaluate(
	_ context.Context,
	batch evaluation.Batch,
) (map[string]evaluation.Result, error) {
	results := make(map[string]evaluation.Result, len(batch.Rules))
	for _, rule := range batch.Rules {
		result, ok := evaluator.results[batch.CodeUnit.Name]
		if !ok {
			result = evaluation.Result{Status: evaluation.StatusPass, Confidence: 1}
		}
		results[rule.ID] = result
	}
	return results, nil
}

func TestRunAggregatesDecisionsAcrossCodeUnits(t *testing.T) {
	t.Parallel()

	floor := 0.8
	tests := []struct {
		name       string
		alpha      evaluation.Result
		beta       evaluation.Result
		want       Outcome
		pass       int
		skip       int
		reported   int
		belowFloor int
	}{
		{
			name:  "one pass and one skip passes",
			alpha: evaluation.Result{Status: evaluation.StatusPass, Confidence: 1},
			beta:  evaluation.Result{Status: evaluation.StatusSkip, Confidence: 1},
			want:  OutcomePass,
			pass:  1,
			skip:  1,
		},
		{
			name:  "all skip is inconclusive",
			alpha: evaluation.Result{Status: evaluation.StatusSkip, Confidence: 1},
			beta:  evaluation.Result{Status: evaluation.StatusSkip, Confidence: 1},
			want:  OutcomeInconclusive,
			skip:  2,
		},
		{
			name:       "below-floor fail makes it inconclusive",
			alpha:      evaluation.Result{Status: evaluation.StatusFail, Confidence: 0.4},
			beta:       evaluation.Result{Status: evaluation.StatusPass, Confidence: 1},
			want:       OutcomeInconclusive,
			pass:       1,
			belowFloor: 1,
		},
		{
			name:     "reported fail wins",
			alpha:    evaluation.Result{Status: evaluation.StatusFail, Confidence: 0.9},
			beta:     evaluation.Result{Status: evaluation.StatusPass, Confidence: 1},
			want:     OutcomeFail,
			pass:     1,
			reported: 1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := filepath.Join(root, "sample.go")
			source := "package sample\n\nfunc Alpha() {}\n\nfunc Beta() {}\n"
			if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			document := Document{Version: currentVersion, Cases: []Case{{
				Rule:   "database-joins",
				File:   "sample.go",
				Expect: ExpectPass,
				abs:    path,
			}}}
			cfg := sampleConfig(nil)
			cfg.MinConfidence = &floor
			report, err := Run(
				context.Background(),
				document,
				cfg,
				testExtractor(t),
				perUnitEvaluator{results: map[string]evaluation.Result{
					"Alpha": test.alpha,
					"Beta":  test.beta,
				}},
				Options{Root: root, Concurrency: 1},
			)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			result := report.Cases[0]
			if result.Actual != test.want {
				t.Fatalf("actual = %v, want %v", result.Actual, test.want)
			}
			if result.Decisions.Pass != test.pass ||
				result.Decisions.Skip != test.skip ||
				result.Decisions.Reported != test.reported ||
				result.Decisions.BelowFloor != test.belowFloor {
				t.Fatalf("decisions = %#v", result.Decisions)
			}
			if len(result.Units) != 2 ||
				result.Units[0].Name != "Alpha" ||
				result.Units[1].Name != "Beta" {
				t.Fatalf("units = %#v", result.Units)
			}
		})
	}
}

// barrierEvaluator holds every call until all expected calls have started, so
// concurrent recording overlaps.
type barrierEvaluator struct {
	mu      sync.Mutex
	started int
	total   int
	release chan struct{}
}

func (evaluator *barrierEvaluator) Evaluate(
	_ context.Context,
	batch evaluation.Batch,
) (map[string]evaluation.Result, error) {
	evaluator.mu.Lock()
	evaluator.started++
	if evaluator.started == evaluator.total {
		close(evaluator.release)
	}
	evaluator.mu.Unlock()
	<-evaluator.release

	results := make(map[string]evaluation.Result, len(batch.Rules))
	for _, rule := range batch.Rules {
		results[rule.ID] = evaluation.Result{Status: evaluation.StatusPass, Confidence: 1}
	}
	return results, nil
}

func TestRunConcurrentUnitsAreRaceFree(t *testing.T) {
	t.Parallel()

	const units = 8
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	var source strings.Builder
	source.WriteString("package sample\n\n")
	for index := 0; index < units; index++ {
		fmt.Fprintf(&source, "func F%d() {}\n\n", index)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	document := Document{Version: currentVersion, Cases: []Case{{
		Rule:   "database-joins",
		File:   "sample.go",
		Expect: ExpectPass,
		abs:    path,
	}}}
	report, err := Run(
		context.Background(),
		document,
		sampleConfig(nil),
		testExtractor(t),
		&barrierEvaluator{total: units, release: make(chan struct{})},
		Options{Root: root, Concurrency: units},
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Matched != 1 || report.Cases[0].Decisions.Pass != units {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunEmitsUnitDecisions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	source := "package sample\n\nfunc Alpha() {}\n\nfunc Beta() {}\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	document := Document{Version: currentVersion, Cases: []Case{{
		Rule:   "database-joins",
		File:   "sample.go",
		Expect: ExpectPass,
		abs:    path,
	}}}

	var mu sync.Mutex
	seen := map[string]UnitDecision{}
	_, err := Run(
		context.Background(),
		document,
		sampleConfig(nil),
		testExtractor(t),
		perUnitEvaluator{results: map[string]evaluation.Result{
			"Alpha": {Status: evaluation.StatusPass, Confidence: 1},
			"Beta":  {Status: evaluation.StatusFail, Confidence: 0.9},
		}},
		Options{
			Root:        root,
			Concurrency: 2,
			OnUnit: func(_ Case, unit UnitDecision) {
				mu.Lock()
				seen[unit.Name] = unit
				mu.Unlock()
			},
		},
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("emitted units = %#v", seen)
	}
	beta, ok := seen["Beta"]
	if !ok || beta.Status != evaluation.StatusFail || !beta.Reported {
		t.Fatalf("Beta unit = %#v", beta)
	}
}
