package evals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/evaluation"
	"github.com/codegirl-007/jevlint/internal/parsing"
	"github.com/codegirl-007/jevlint/internal/runner"
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

// Decisions counts the raw answers a rule gave across a fixture. Reported
// failures reached the confidence floor; below-floor failures did not.
type Decisions struct {
	Pass       int `json:"pass"`
	Fail       int `json:"fail"`
	Skip       int `json:"skip"`
	Abstain    int `json:"abstain"`
	Reported   int `json:"reported"`
	BelowFloor int `json:"belowFloor"`
}

// UnitDecision is one raw rule answer for one code unit.
type UnitDecision struct {
	Name       string            `json:"name"`
	Kind       parsing.CodeKind  `json:"kind"`
	StartLine  uint              `json:"startLine"`
	EndLine    uint              `json:"endLine"`
	Status     evaluation.Status `json:"status"`
	Confidence float64           `json:"confidence"`
	Reported   bool              `json:"reported"`
}

// Result is the outcome of scoring one eval case.
type Result struct {
	Rule       string         `json:"rule"`
	Name       string         `json:"name,omitempty"`
	File       string         `json:"file"`
	Expected   Expect         `json:"expected"`
	Actual     Outcome        `json:"actual"`
	Matched    bool           `json:"matched"`
	Confidence *float64       `json:"confidence,omitempty"`
	Floor      float64        `json:"floor"`
	Decisions  Decisions      `json:"decisions"`
	Units      []UnitDecision `json:"units,omitempty"`
}

// Report is the outcome of scoring every case.
type Report struct {
	Total              int      `json:"total"`
	Matched            int      `json:"matched"`
	Mismatched         int      `json:"mismatched"`
	Inconclusive       int      `json:"inconclusive"`
	ReportedFailures   int      `json:"reportedFailures"`
	BelowFloorFailures int      `json:"belowFloorFailures"`
	Cases              []Result `json:"cases"`
}

// Options holds the settings for an eval run.
type Options struct {
	Root        string
	Concurrency int
	// OnUnit, when set, is called with each unit decision as it is scored.
	OnUnit func(Case, UnitDecision)
	// OnResult, when set, is called with each case result as it is scored.
	OnResult func(Result)
}

// recordingEvaluator wraps a client and records the raw decisions it returns.
// The runner evaluates units concurrently, so its maps are guarded.
type recordingEvaluator struct {
	inner     evaluation.Evaluator
	onUnit    func(UnitDecision)
	mu        sync.Mutex
	floor     float64
	bestFail  map[string]float64
	decisions map[string]*Decisions
	units     map[string][]UnitDecision
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
		if options.OnResult != nil {
			options.OnResult(result)
		}
		report.Total++
		report.ReportedFailures += result.Decisions.Reported
		report.BelowFloorFailures += result.Decisions.BelowFloor
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
	floor := cfg.ConfidenceFloor(rule)
	recorder := &recordingEvaluator{inner: evaluator, floor: floor}
	if options.OnUnit != nil {
		recorder.onUnit = func(unit UnitDecision) {
			options.OnUnit(evalCase, unit)
		}
	}
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
		Confidence: recorder.bestFailConfidence(evalCase.Rule),
		Floor:      floor,
		Decisions:  decisions,
		Units:      recorder.unitsFor(evalCase.Rule, floor),
	}
	return result, nil
}

// classifyOutcome turns a rule's raw decisions into a truthful case outcome.
//
// A reported failure makes the case fail. Without one, a below-floor failure,
// or no explicit pass at all, makes the case inconclusive. Otherwise the case
// passes.
func classifyOutcome(decisions Decisions) Outcome {
	switch {
	case decisions.Reported > 0:
		return OutcomeFail
	case decisions.BelowFloor > 0:
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
	recorder.mu.Lock()
	if recorder.bestFail == nil {
		recorder.bestFail = make(map[string]float64)
	}
	if recorder.decisions == nil {
		recorder.decisions = make(map[string]*Decisions)
	}
	if recorder.units == nil {
		recorder.units = make(map[string][]UnitDecision)
	}
	emitted := make([]UnitDecision, 0, len(results))
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
		unit := UnitDecision{
			Name:       batch.CodeUnit.Name,
			Kind:       batch.CodeUnit.Kind,
			StartLine:  batch.CodeUnit.StartLine,
			EndLine:    batch.CodeUnit.EndLine,
			Status:     result.Status,
			Confidence: result.Confidence,
			Reported: result.Status == evaluation.StatusFail &&
				result.Confidence >= recorder.floor,
		}
		recorder.units[id] = append(recorder.units[id], unit)
		emitted = append(emitted, unit)
	}
	recorder.mu.Unlock()

	if recorder.onUnit != nil {
		for _, unit := range emitted {
			recorder.onUnit(unit)
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

// decisionsFor returns a copy of a rule's raw decisions, with reported and
// below-floor failures filled in from the findings the runner reported.
func (recorder *recordingEvaluator) decisionsFor(ruleID string, reported int) Decisions {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	var decisions Decisions
	if recorded := recorder.decisions[ruleID]; recorded != nil {
		decisions = *recorded
	}
	decisions.Reported = reported
	decisions.BelowFloor = decisions.Fail - reported
	if decisions.BelowFloor < 0 {
		decisions.BelowFloor = 0
	}
	return decisions
}

// unitsFor returns one rule's per-unit decisions in source order, marking the
// failures that reach the confidence floor.
func (recorder *recordingEvaluator) unitsFor(ruleID string, floor float64) []UnitDecision {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	recorded := recorder.units[ruleID]
	if len(recorded) == 0 {
		return nil
	}
	units := append([]UnitDecision(nil), recorded...)
	for index := range units {
		units[index].Reported = units[index].Status == evaluation.StatusFail &&
			units[index].Confidence >= floor
	}
	sort.SliceStable(units, func(i, j int) bool {
		if units[i].StartLine != units[j].StartLine {
			return units[i].StartLine < units[j].StartLine
		}
		return units[i].Name < units[j].Name
	})
	return units
}

// bestFailConfidence returns the strongest fail score for a rule, whether or
// not it reaches the confidence floor.
func (recorder *recordingEvaluator) bestFailConfidence(ruleID string) *float64 {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	confidence, ok := recorder.bestFail[ruleID]
	if !ok {
		return nil
	}
	value := confidence
	return &value
}
