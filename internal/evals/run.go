package evals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"jevlint/internal/config"
	"jevlint/internal/evaluation"
	"jevlint/internal/parsing"
	"jevlint/internal/runner"
)

var ErrNoApplicableUnits = errors.New("no applicable code units")

// Outcome is the truthful result of one eval case after its rule's raw
// decisions are aggregated across the fixture's code units.
type Outcome int

const (
	OutcomeUnknown Outcome = iota
	OutcomePass
	OutcomeFail
	OutcomeInconclusive
)

// String returns the outcome name.
func (outcome Outcome) String() string {
	switch outcome {
	case OutcomePass:
		return "pass"
	case OutcomeFail:
		return "fail"
	case OutcomeInconclusive:
		return "inconclusive"
	default:
		return "unknown"
	}
}

// MarshalJSON writes the outcome as its name.
func (outcome Outcome) MarshalJSON() ([]byte, error) {
	return json.Marshal(outcome.String())
}

// UnmarshalJSON reads an outcome from its name.
func (outcome *Outcome) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch value {
	case "pass":
		*outcome = OutcomePass
	case "fail":
		*outcome = OutcomeFail
	case "inconclusive":
		*outcome = OutcomeInconclusive
	default:
		return fmt.Errorf("invalid outcome %q", value)
	}
	return nil
}

// Decisions counts the raw answers a rule gave across a fixture. Reportable
// failures are kept apart from failures hidden by the confidence floor.
type Decisions struct {
	Pass           int `json:"pass"`
	Fail           int `json:"fail"`
	Skip           int `json:"skip"`
	Abstain        int `json:"abstain"`
	ReportableFail int `json:"reportableFail"`
	HiddenFail     int `json:"hiddenFail"`
}

// Result is the outcome of scoring one eval case.
type Result struct {
	Rule       string    `json:"rule"`
	Name       string    `json:"name,omitempty"`
	File       string    `json:"file"`
	Expected   Expect    `json:"expected"`
	Actual     Outcome   `json:"actual"`
	Matched    bool      `json:"matched"`
	Confidence *float64  `json:"confidence,omitempty"`
	Decisions  Decisions `json:"decisions"`
}

// Report is the outcome of scoring every case.
type Report struct {
	Total        int      `json:"total"`
	Matched      int      `json:"matched"`
	Mismatched   int      `json:"mismatched"`
	Inconclusive int      `json:"inconclusive"`
	Cases        []Result `json:"cases"`
}

// Options holds the settings for an eval run.
type Options struct {
	Root        string
	Concurrency int
}

// recordingEvaluator wraps a client and records the raw decisions it returns.
type recordingEvaluator struct {
	inner     evaluation.Evaluator
	bestFail  map[string]float64
	decisions map[string]*Decisions
}

// Run scores every case and returns the report.
func Run(
	ctx context.Context,
	document Document,
	cfg config.Config,
	extractor *parsing.Extractor,
	evaluator evaluation.Evaluator,
	options Options,
) (Report, error) {
	report := Report{Cases: make([]Result, 0, len(document.Cases))}
	for _, evalCase := range document.Cases {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		result, err := runCase(ctx, evalCase, cfg, extractor, evaluator, options)
		if err != nil {
			return Report{}, err
		}
		report.Total++
		switch {
		case result.Matched:
			report.Matched++
		case result.Actual == OutcomeInconclusive:
			report.Inconclusive++
		default:
			report.Mismatched++
		}
		report.Cases = append(report.Cases, result)
	}
	return report, nil
}

// runCase scores one case.
func runCase(
	ctx context.Context,
	evalCase Case,
	cfg config.Config,
	extractor *parsing.Extractor,
	evaluator evaluation.Evaluator,
	options Options,
) (Result, error) {
	rule, ok := ruleByID(cfg, evalCase.Rule)
	if !ok {
		return Result{}, fmt.Errorf("unknown eval rule %q", evalCase.Rule)
	}
	recorder := &recordingEvaluator{inner: evaluator}
	check := runner.Runner{
		Extractor: extractor,
		Evaluator: recorder,
	}
	report, err := check.Evaluate(ctx, evalConfig(cfg, rule), runner.Options{
		Root:        options.Root,
		Paths:       []string{evalCase.AbsolutePath()},
		Concurrency: options.Concurrency,
	})
	if err != nil {
		return Result{}, fmt.Errorf("evaluate %s: %w", caseLabel(evalCase), err)
	}
	if report.Evaluations == 0 {
		return Result{}, fmt.Errorf("%s: %w", caseLabel(evalCase), ErrNoApplicableUnits)
	}

	decisions := recorder.decisionsFor(evalCase.Rule, len(report.Findings))
	actual := classifyOutcome(decisions)
	result := Result{
		Rule:       evalCase.Rule,
		Name:       evalCase.Name,
		File:       evalCase.File,
		Expected:   evalCase.Expect,
		Actual:     actual,
		Matched:    actual != OutcomeInconclusive && actual == expectedOutcome(evalCase.Expect),
		Confidence: recorder.reportableConfidence(evalCase.Rule, cfg.ConfidenceFloor(rule)),
		Decisions:  decisions,
	}
	return result, nil
}

// classifyOutcome turns a rule's raw decisions into a truthful case outcome.
//
// A reportable failure makes the case fail. Without one, a failure below the
// confidence floor, or no explicit pass at all, makes the case inconclusive.
// Otherwise the case passes.
func classifyOutcome(decisions Decisions) Outcome {
	switch {
	case decisions.ReportableFail > 0:
		return OutcomeFail
	case decisions.HiddenFail > 0:
		return OutcomeInconclusive
	case decisions.Pass > 0:
		return OutcomePass
	default:
		return OutcomeInconclusive
	}
}

// expectedOutcome maps a case's expectation to the outcome it wants.
func expectedOutcome(expect Expect) Outcome {
	if expect == ExpectFail {
		return OutcomeFail
	}
	return OutcomePass
}

// evalConfig builds a config with a single rule and no file filters.
func evalConfig(cfg config.Config, rule config.Rule) config.Config {
	rule.Include = nil
	rule.Exclude = nil
	return config.Config{
		Languages:     cfg.Languages,
		MinConfidence: cfg.MinConfidence,
		Rules:         []config.Rule{rule},
	}
}

// Evaluate runs the batch and records the raw decisions per rule.
func (recorder *recordingEvaluator) Evaluate(
	ctx context.Context,
	batch evaluation.Batch,
) (map[string]evaluation.Result, error) {
	results, err := recorder.inner.Evaluate(ctx, batch)
	if err != nil {
		return nil, err
	}
	// The localization pass re-checks regions with the same rule. Those
	// answers refine a finding; they are not decisions about the fixture's
	// code units, so keep them out of the raw counts.
	if batch.CodeUnit.Kind == parsing.CodeKindRegion {
		return results, nil
	}
	if recorder.bestFail == nil {
		recorder.bestFail = make(map[string]float64)
	}
	if recorder.decisions == nil {
		recorder.decisions = make(map[string]*Decisions)
	}
	for id, result := range results {
		decisions := recorder.decisions[id]
		if decisions == nil {
			decisions = &Decisions{}
			recorder.decisions[id] = decisions
		}
		switch result.Status {
		case evaluation.StatusPass:
			decisions.Pass++
		case evaluation.StatusFail:
			decisions.Fail++
		case evaluation.StatusSkip:
			decisions.Skip++
		case evaluation.StatusAbstain:
			decisions.Abstain++
		}
		if result.Status == evaluation.StatusFail {
			if current, exists := recorder.bestFail[id]; !exists || result.Confidence > current {
				recorder.bestFail[id] = result.Confidence
			}
		}
	}
	return results, nil
}

// CacheStats returns the cache counts from the wrapped client.
func (recorder *recordingEvaluator) CacheStats() evaluation.CacheStats {
	provider, ok := recorder.inner.(evaluation.CacheStatsProvider)
	if !ok {
		return evaluation.CacheStats{}
	}
	return provider.CacheStats()
}

// decisionsFor returns a copy of a rule's raw decisions, with reportable and
// hidden failures filled in from the findings the runner reported.
func (recorder *recordingEvaluator) decisionsFor(ruleID string, reportable int) Decisions {
	var decisions Decisions
	if recorded := recorder.decisions[ruleID]; recorded != nil {
		decisions = *recorded
	}
	decisions.ReportableFail = reportable
	decisions.HiddenFail = decisions.Fail - reportable
	if decisions.HiddenFail < 0 {
		decisions.HiddenFail = 0
	}
	return decisions
}

// reportableConfidence returns the fail score when it reaches the floor.
func (recorder *recordingEvaluator) reportableConfidence(
	ruleID string,
	floor float64,
) *float64 {
	confidence, ok := recorder.bestFail[ruleID]
	if !ok || confidence < floor {
		return nil
	}
	value := confidence
	return &value
}
