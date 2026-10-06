package evals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
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

// UnitDecision is one raw rule answer for one code unit.
type UnitDecision struct {
	Name            string                               `json:"name"`
	Kind            parsing.CodeKind                     `json:"kind"`
	StartLine       uint                                 `json:"startLine"`
	EndLine         uint                                 `json:"endLine"`
	Status          evaluation.Status                    `json:"status"`
	FailProbability float64                              `json:"failProbability"`
	Score           float64                              `json:"score"`
	Checks          float64                              `json:"checks"`
	SubjectGate     bool                                 `json:"subjectGate"`
	Answers         map[string]evaluation.QuestionAnswer `json:"answers,omitempty"`
	Reported        bool                                 `json:"reported"`
}

// Result is the outcome of scoring one eval case.
type Result struct {
	Rule            string           `json:"rule"`
	Name            string           `json:"name,omitempty"`
	File            string           `json:"file"`
	Expected        Expect           `json:"expected"`
	Actual          Outcome          `json:"actual"`
	Matched         bool             `json:"matched"`
	FailProbability float64          `json:"failProbability"`
	Floor           float64          `json:"floor"`
	Decisions       runner.Decisions `json:"decisions"`
	Units           []UnitDecision   `json:"units,omitempty"`
}

// Report is the outcome of scoring every case.
type Report struct {
	Total              int                      `json:"total"`
	Matched            int                      `json:"matched"`
	Mismatched         int                      `json:"mismatched"`
	Inconclusive       int                      `json:"inconclusive"`
	ReportedFailures   int                      `json:"reportedFailures"`
	BelowFloorFailures int                      `json:"belowFloorFailures"`
	Models             []string                 `json:"models"`
	Usage              evaluation.Usage         `json:"usage"`
	Oversized          []evaluation.Oversized   `json:"oversized,omitempty"`
	ContextDropped     []evaluation.ContextDrop `json:"contextDropped,omitempty"`
	Cases              []Result                 `json:"cases"`
}

// Options holds the settings for an eval run.
type Options struct {
	// Root confines programmatically constructed cases; loaded cases use their document directory.
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
	inner  evaluation.Evaluator
	onUnit func(UnitDecision)
	mu     sync.Mutex
	floor  float64
	units  map[string][]UnitDecision
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
	report := Report{Cases: make([]Result, 0, len(document.Cases)), Models: []string{}}
	for _, evalCase := range document.Cases {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		result, checkReport, err := runCase(ctx, evalCase, cfg, extractor, evaluator, options)
		if err != nil {
			return Report{}, err
		}
		mergeEvalMeta(&report, checkReport)
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
) (Result, runner.Report, error) {
	rule, ok := ruleByID(cfg, evalCase.Rule)
	if !ok {
		return Result{}, runner.Report{}, fmt.Errorf("unknown eval rule %q", evalCase.Rule)
	}
	floor := cfg.FailProbabilityFloor(rule)
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
	root := evalCase.root
	if root == "" {
		root = options.Root
	}
	checkReport, err := check.Evaluate(ctx, evalConfig(cfg, rule), runner.Options{
		Root:        root,
		Paths:       []string{evalCase.AbsolutePath()},
		Concurrency: options.Concurrency,
	})
	if err != nil {
		return Result{}, runner.Report{}, fmt.Errorf("evaluate %s: %w", caseLabel(evalCase), err)
	}
	if checkReport.Evaluations == 0 && len(checkReport.Oversized) == 0 {
		return Result{}, checkReport, fmt.Errorf("%s: %w", caseLabel(evalCase), ErrNoApplicableUnits)
	}

	decisions := checkReport.Rules[evalCase.Rule].Decisions
	actual := classifyOutcome(decisions)
	result := Result{
		Rule:            evalCase.Rule,
		Name:            evalCase.Name,
		File:            evalCase.File,
		Expected:        evalCase.Expect,
		Actual:          actual,
		Matched:         actual != OutcomeInconclusive && actual == expectedOutcome(evalCase.Expect),
		FailProbability: recorder.maxFailProbability(evalCase.Rule),
		Floor:           floor,
		Decisions:       decisions,
		Units:           recorder.unitsFor(evalCase.Rule),
	}
	return result, checkReport, nil
}

func mergeEvalMeta(report *Report, check runner.Report) {
	report.Models = unionModels(report.Models, check.Models)
	report.Usage.Requests += check.Usage.Requests
	report.Usage.InputTokens += check.Usage.InputTokens
	report.Usage.OutputTokens += check.Usage.OutputTokens
	report.Oversized = append(report.Oversized, check.Oversized...)
	report.ContextDropped = append(report.ContextDropped, check.ContextDropped...)
}

func unionModels(current, added []string) []string {
	seen := make(map[string]struct{}, len(current)+len(added))
	for _, name := range current {
		seen[name] = struct{}{}
	}
	for _, name := range added {
		seen[name] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// classifyOutcome turns a rule's raw decisions into a truthful case outcome.
//
// A reported failure makes the case fail. Without one, a below-floor failure,
// or no explicit pass at all, makes the case inconclusive. Otherwise the case
// passes.
func classifyOutcome(decisions runner.Decisions) Outcome {
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
		Languages:          cfg.Languages,
		MinFailProbability: cfg.MinFailProbability,
		Rules:              []config.Rule{rule},
	}
}

// Evaluate runs the batch and records the raw decisions per rule.
func (recorder *recordingEvaluator) Evaluate(
	ctx context.Context,
	batch evaluation.Batch,
) (map[string]evaluation.Result, error) {
	results, err := recorder.inner.Evaluate(ctx, neutralizeFixturePath(batch))
	if err != nil {
		return nil, err
	}
	// Localization answers refine a finding; they are not raw decisions
	// about the fixture's primary code units.
	if batch.CodeUnit.Kind == parsing.CodeKindRegion || len(batch.Regions) > 0 {
		return results, nil
	}
	recorder.mu.Lock()
	if recorder.units == nil {
		recorder.units = make(map[string][]UnitDecision)
	}
	emitted := make([]UnitDecision, 0, len(results))
	for id, result := range results {
		unit := UnitDecision{
			Name:            batch.CodeUnit.Name,
			Kind:            batch.CodeUnit.Kind,
			StartLine:       batch.CodeUnit.StartLine,
			EndLine:         batch.CodeUnit.EndLine,
			Status:          result.Status,
			FailProbability: result.FailProbability,
			Score:           result.Score,
			Checks:          result.Checks,
			SubjectGate:     result.SubjectGate,
			Answers:         result.Answers,
			Reported:        result.FailProbability >= recorder.floor,
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

// neutralizeFixturePath hides the fixture's file name from the evaluator.
// Fixture names such as bad/, fail.rs or real-x-bug.rs encode the expected
// verdict, so the request carries a stable opaque name with the same extension,
// including in unit names derived from the file name.
func neutralizeFixturePath(batch evaluation.Batch) evaluation.Batch {
	original := batch.CodeUnit.Path
	sum := sha256.Sum256([]byte(original))
	neutral := "case-" + hex.EncodeToString(sum[:6]) + path.Ext(original)
	unit := batch.CodeUnit
	unit.Path = neutral
	unit.Name = strings.ReplaceAll(unit.Name, path.Base(original), neutral)
	unit.Callees = append([]parsing.CalleeContext(nil), unit.Callees...)
	for index := range unit.Callees {
		if unit.Callees[index].Path == original {
			unit.Callees[index].Path = neutral
		}
	}
	unit.RelatedTypes = append([]parsing.TypeDeclaration(nil), unit.RelatedTypes...)
	for index := range unit.RelatedTypes {
		if unit.RelatedTypes[index].Path == original {
			unit.RelatedTypes[index].Path = neutral
		}
	}
	batch.CodeUnit = unit
	return batch
}

// CacheStats returns the cache counts from the wrapped client.
func (recorder *recordingEvaluator) CacheStats() evaluation.CacheStats {
	provider, ok := recorder.inner.(evaluation.CacheStatsProvider)
	if !ok {
		return evaluation.CacheStats{}
	}
	return provider.CacheStats()
}

// RunMeta returns models and usage from the wrapped client.
func (recorder *recordingEvaluator) RunMeta() evaluation.RunMeta {
	provider, ok := recorder.inner.(evaluation.RunMetaProvider)
	if !ok {
		return evaluation.RunMeta{Models: []string{}}
	}
	return provider.RunMeta()
}

// unitsFor returns one rule's raw per-unit decisions in source order.
func (recorder *recordingEvaluator) unitsFor(ruleID string) []UnitDecision {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	recorded := recorder.units[ruleID]
	if len(recorded) == 0 {
		return nil
	}
	units := append([]UnitDecision(nil), recorded...)
	sort.SliceStable(units, func(i, j int) bool {
		if units[i].StartLine != units[j].StartLine {
			return units[i].StartLine < units[j].StartLine
		}
		return units[i].Name < units[j].Name
	})
	return units
}

// maxFailProbability returns the strongest fail probability across a case's units.
func (recorder *recordingEvaluator) maxFailProbability(ruleID string) float64 {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	max := 0.0
	for _, unit := range recorder.units[ruleID] {
		if unit.FailProbability > max {
			max = unit.FailProbability
		}
	}
	return max
}
