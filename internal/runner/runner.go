package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/evaluation"
	"github.com/codegirl-007/jevlint/internal/parsing"
	"github.com/codegirl-007/jevlint/internal/scoping"
)

const defaultConcurrency = 4

const (
	codeUnitRankType = iota
	codeUnitRankFunction
	codeUnitRankComment
	codeUnitRankField
	codeUnitRankOther
)

// Runner reads code and asks the evaluator about it.
type Runner struct {
	Extractor *parsing.Extractor
	Evaluator evaluation.Evaluator
}

// Options holds the settings for a run.
type Options struct {
	Root           string // Source reads are confined to this directory.
	Paths          []string
	Concurrency    int
	SourceOverlay  map[string][]byte
	ShowBelowFloor bool
	// Progress is called serially after each primary job completes.
	Progress func(done, total int)
}

// Decisions counts primary rule answers. Reported and belowFloor follow the
// fail-probability threshold, not the status alone.
type Decisions struct {
	Pass       int `json:"pass"`
	Fail       int `json:"fail"`
	Skip       int `json:"skip"`
	Abstain    int `json:"abstain"`
	Reported   int `json:"reported"`
	BelowFloor int `json:"belowFloor"`
}

// RuleReport describes a rule and its decisions in this run.
type RuleReport struct {
	Description string          `json:"description"`
	Severity    config.Severity `json:"severity"`
	Decisions   Decisions       `json:"decisions"`
}

// Report is the outcome of a run.
type Report struct {
	ScannedFiles   int                      `json:"scannedFiles"`
	CodeUnits      int                      `json:"codeUnits"`
	Evaluations    int                      `json:"evaluations"`
	Cache          *evaluation.CacheStats   `json:"cache,omitempty"`
	Models         []string                 `json:"models"`
	Usage          evaluation.Usage         `json:"usage"`
	Rules          map[string]RuleReport    `json:"rules"`
	Findings       []Finding                `json:"findings"`
	BelowFloor     []BelowFloorFinding      `json:"belowFloor,omitempty"`
	Oversized      []evaluation.Oversized   `json:"oversized,omitempty"`
	ContextDropped []evaluation.ContextDrop `json:"contextDropped,omitempty"`
	SourcePaths    []string                 `json:"-"`
}

// Finding is one reported or below-floor result for one piece of code.
type Finding struct {
	RuleID          string            `json:"ruleId"`
	Severity        config.Severity   `json:"severity"`
	Status          evaluation.Status `json:"status"`
	Path            string            `json:"path"`
	Language        string            `json:"language"`
	Kind            parsing.CodeKind  `json:"kind"`
	Name            string            `json:"name"`
	StartLine       uint              `json:"startLine"`
	EndLine         uint              `json:"endLine"`
	StartColumn     uint              `json:"startColumn"`
	EndColumn       uint              `json:"endColumn"`
	Snippet         string            `json:"snippet"`
	Locations       []Location        `json:"locations,omitempty"`
	FailProbability float64           `json:"failProbability"`
}

// BelowFloorFinding is a notable result that did not reach the reporting threshold.
type BelowFloorFinding struct {
	Finding
}

// Location is a smaller place inside a finding.
type Location struct {
	Category    string `json:"category"`
	Kind        string `json:"kind"`
	Source      string `json:"source"`
	StartLine   uint   `json:"startLine"`
	EndLine     uint   `json:"endLine"`
	StartColumn uint   `json:"startColumn"`
	EndColumn   uint   `json:"endColumn"`
}

// evaluationJob is one piece of code and the rules to check against it.
type evaluationJob struct {
	rules []config.Rule
	unit  parsing.CodeUnit
}

// evaluationOutcome is the result of one job.
type evaluationOutcome struct {
	evaluations int
	rules       map[string]RuleReport
	findings    []pendingFinding
	belowFloor  []BelowFloorFinding
}

// pendingFinding is a finding that may still gain a location.
type pendingFinding struct {
	finding Finding
	rule    config.Rule
	unit    parsing.CodeUnit
}

// localizationJob is one request locating a failed unit.
type localizationJob struct {
	findingIndex int
	rule         config.Rule
	parent       parsing.CodeUnit
	regions      []parsing.Region
}

// localizationOutcome is the places found for one finding.
type localizationOutcome struct {
	findingIndex int
	locations    []Location
}

// checkSetup holds the resolved settings for a run.
type checkSetup struct {
	root          string
	paths         []string
	concurrency   int
	sourceOverlay map[string][]byte
}

// plannedFile is a source file and the work it produced.
type plannedFile struct {
	relative    string
	units       []parsing.CodeUnit
	applicable  []config.Rule
	types       []parsing.TypeDeclaration
	testModules []parsing.RustModule
}

// selectedRegion is a region and the size of the unit that holds it.
type selectedRegion struct {
	unit       parsing.CodeUnit
	parentSpan uint
}

// scheduledWork pairs a job with its outcome.
type scheduledWork[Job any, Outcome any] struct {
	job     Job
	outcome Outcome
}

// Evaluate reads the files and returns the findings.
func (runner Runner) Evaluate(ctx context.Context, cfg config.Config, options Options) (Report, error) {
	cacheBefore, hasCacheStats := evaluation.CacheStats{}, false
	if provider, ok := runner.Evaluator.(evaluation.CacheStatsProvider); ok {
		cacheBefore, hasCacheStats = provider.CacheStats(), true
	}
	metaBefore := evaluation.RunMeta{}
	if provider, ok := runner.Evaluator.(evaluation.RunMetaProvider); ok {
		metaBefore = provider.RunMeta()
	}
	setup, err := runner.prepareCheck(options)
	if err != nil {
		return Report{}, err
	}
	root, err := os.OpenRoot(setup.root)
	if err != nil {
		return Report{}, fmt.Errorf("open project root: %w", err)
	}
	defer root.Close()
	files, err := discover(ctx, root, setup.paths, runner.Extractor)
	if err != nil {
		return Report{}, err
	}
	report, jobs, err := runner.planEvaluations(
		ctx,
		cfg,
		root,
		files,
		setup.sourceOverlay,
	)
	if err != nil {
		return Report{}, err
	}
	outcomes, err := runJobs(
		ctx,
		jobs,
		setup.concurrency,
		func(ctx context.Context, job evaluationJob) (evaluationOutcome, error) {
			return evaluateJob(ctx, runner.Evaluator, job, cfg)
		},
		options.Progress,
	)
	if err != nil {
		return Report{}, err
	}
	pending := collectOutcomes(&report, outcomes, options.ShowBelowFloor)
	localizationEvaluations, err := localizeFindings(
		ctx,
		runner.Evaluator,
		pending,
		setup.concurrency,
		cfg,
	)
	if err != nil {
		return Report{}, err
	}
	report.Evaluations += localizationEvaluations
	for _, item := range pending {
		report.Findings = append(report.Findings, item.finding)
	}
	if hasCacheStats {
		cacheAfter := evaluation.CacheStats{}
		if provider, ok := runner.Evaluator.(evaluation.CacheStatsProvider); ok {
			cacheAfter = provider.CacheStats()
		}
		cacheDelta := subtractCacheStats(cacheAfter, cacheBefore)
		if cacheDelta.Hits+cacheDelta.Misses+cacheDelta.Writes > 0 {
			report.Cache = &cacheDelta
		}
	}
	if provider, ok := runner.Evaluator.(evaluation.RunMetaProvider); ok {
		applyRunMeta(&report, metaBefore, provider.RunMeta())
	} else {
		report.Models = []string{}
	}
	return report, nil
}

// subtractCacheStats returns the cache counts added during a run.
func subtractCacheStats(
	after evaluation.CacheStats,
	before evaluation.CacheStats,
) evaluation.CacheStats {
	return evaluation.CacheStats{
		Hits:   after.Hits - before.Hits,
		Misses: after.Misses - before.Misses,
		Writes: after.Writes - before.Writes,
	}
}
func applyRunMeta(report *Report, before, after evaluation.RunMeta) {
	report.Models = modelsSince(before.Models, after.Models)
	report.Usage = evaluation.Usage{
		Requests:     after.Usage.Requests - before.Usage.Requests,
		InputTokens:  after.Usage.InputTokens - before.Usage.InputTokens,
		OutputTokens: after.Usage.OutputTokens - before.Usage.OutputTokens,
	}
	if len(after.Oversized) > len(before.Oversized) {
		report.Oversized = append([]evaluation.Oversized(nil), after.Oversized[len(before.Oversized):]...)
	}
	if len(after.ContextDropped) > len(before.ContextDropped) {
		report.ContextDropped = append(
			[]evaluation.ContextDrop(nil),
			after.ContextDropped[len(before.ContextDropped):]...,
		)
	}
}

func modelsSince(before, after []string) []string {
	seen := make(map[string]struct{}, len(before))
	for _, name := range before {
		seen[name] = struct{}{}
	}
	added := make([]string, 0)
	for _, name := range after {
		if _, ok := seen[name]; ok {
			continue
		}
		added = append(added, name)
	}
	return added
}

// prepareCheck checks the settings and resolves the project root.
func (runner Runner) prepareCheck(options Options) (checkSetup, error) {
	if runner.Extractor == nil {
		return checkSetup{}, fmt.Errorf("extractor is required")
	}
	if runner.Evaluator == nil {
		return checkSetup{}, fmt.Errorf("evaluator is required")
	}
	if options.Concurrency < 0 {
		return checkSetup{}, fmt.Errorf("concurrency cannot be negative")
	}
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = defaultConcurrency
	}

	root, err := filepath.Abs(options.Root)
	if err != nil {
		return checkSetup{}, fmt.Errorf("resolve project root: %w", err)
	}
	paths := options.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	return checkSetup{
		root:          root,
		paths:         paths,
		concurrency:   concurrency,
		sourceOverlay: options.SourceOverlay,
	}, nil
}

// planEvaluations reads the files and builds the work for each one.
func (runner Runner) planEvaluations(
	ctx context.Context,
	cfg config.Config,
	root *os.Root,
	files []string,
	sourceOverlay map[string][]byte,
) (Report, []evaluationJob, error) {
	needCallees := rulesWantCallees(cfg.Rules)
	needTypes := rulesWantTypes(cfg.Rules)
	sourceMatches, err := compileSourceMatches(cfg.Rules)
	if err != nil {
		return Report{}, nil, err
	}
	extracted := make([]plannedFile, 0, len(files))
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return Report{}, nil, err
		}
		planned, err := runner.extractFile(
			cfg,
			root,
			file,
			sourceOverlay,
			needCallees || needTypes || filepath.Ext(file) == ".rs",
		)
		if err != nil {
			return Report{}, nil, err
		}
		if planned.relative == "" {
			continue
		}
		extracted = append(extracted, planned)
	}
	markTestModuleFiles(extracted)
	if needTypes {
		resolveTypeContext(extracted)
	}
	if needCallees {
		parsing.ResolveCallees(functionUnits(extracted))
	}

	report := Report{Findings: make([]Finding, 0), Models: []string{}}
	jobs := make([]evaluationJob, 0)
	for _, planned := range extracted {
		if len(planned.applicable) == 0 {
			continue
		}
		report.ScannedFiles++
		report.SourcePaths = append(report.SourcePaths, planned.relative)
		report.CodeUnits += len(planned.units)
		fileJobs, err := jobsForUnits(planned.units, planned.applicable, sourceMatches)
		if err != nil {
			return Report{}, nil, err
		}
		jobs = append(jobs, fileJobs...)
	}
	return report, jobs, nil
}

// extractFile reads one file and records the rules that apply to it.
func (runner Runner) extractFile(
	cfg config.Config,
	root *os.Root,
	file string,
	sourceOverlay map[string][]byte,
	needContext bool,
) (plannedFile, error) {
	relative, err := relativeProjectPath(root.Name(), file)
	if err != nil {
		return plannedFile{}, err
	}
	applicable := make([]config.Rule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		applies, err := scoping.Applies(rule, relative)
		if err != nil {
			return plannedFile{}, err
		}
		if applies {
			applicable = append(applicable, rule)
		}
	}
	if len(applicable) == 0 && !needContext {
		return plannedFile{}, nil
	}
	source, err := readOverlayOrFile(root, relative, sourceOverlay)
	if err != nil {
		if len(applicable) == 0 {
			return plannedFile{}, nil
		}
		return plannedFile{}, err
	}
	extracted, err := runner.Extractor.ExtractFile(relative, source)
	if err != nil {
		if len(applicable) == 0 {
			return plannedFile{}, nil
		}
		return plannedFile{}, err
	}
	units := extracted.Units
	if len(applicable) > 0 {
		requested := requestedRegionKinds(applicable)
		if len(requested) > 0 {
			units = expandUnitsWithRegions(units, selectClosestRegions(units, requested))
		}
	}
	return plannedFile{
		relative:    relative,
		units:       units,
		applicable:  applicable,
		types:       extracted.Types,
		testModules: extracted.TestModules,
	}, nil
}

// functionUnits collects the function units from the planned files.
func functionUnits(files []plannedFile) []*parsing.CodeUnit {
	functions := make([]*parsing.CodeUnit, 0)
	for fileIndex := range files {
		for unitIndex := range files[fileIndex].units {
			unit := &files[fileIndex].units[unitIndex]
			if unit.Kind == parsing.CodeKindFunction {
				functions = append(functions, unit)
			}
		}
	}
	return functions
}

// rulesWantCallees reports whether any rule asks for the called functions.
func rulesWantCallees(rules []config.Rule) bool {
	for _, rule := range rules {
		if rule.Context.Callees {
			return true
		}
	}
	return false
}

// relativeProjectPath returns a path relative to the project root.
func relativeProjectPath(root string, file string) (string, error) {
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return "", fmt.Errorf(
			"make %q relative to project root: %w",
			file,
			err,
		)
	}
	if !filepath.IsLocal(relative) {
		return "", fmt.Errorf("source path %q is outside project root", file)
	}
	return filepath.ToSlash(relative), nil
}

// readOverlayOrFile reads a file from the given source or from disk.
func readOverlayOrFile(
	root *os.Root,
	relative string,
	sourceOverlay map[string][]byte,
) ([]byte, error) {
	if source, ok := sourceOverlay[relative]; ok {
		return source, nil
	}
	file, err := root.Open(filepath.FromSlash(relative))
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", relative, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect %q: %w", relative, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source %q is not a regular file", relative)
	}
	source, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", relative, err)
	}
	return source, nil
}

// jobsForUnits builds the work for each code unit and its rules.
func jobsForUnits(
	units []parsing.CodeUnit,
	rules []config.Rule,
	sourceMatches sourceMatchers,
) ([]evaluationJob, error) {
	jobs := make([]evaluationJob, 0, len(units))
	for _, unit := range units {
		ordinary := make([]config.Rule, 0, len(rules))
		enriched := make([]evaluationJob, 0)
		for _, rule := range rules {
			if !appliesToKind(rule, unit.Kind) ||
				(unit.Test && !rule.IncludeTests) ||
				!sourceMatches.matches(rule.ID, unit.Source) {
				continue
			}
			if !rule.Context.Callees && !rule.Context.Types {
				ordinary = append(ordinary, rule)
				continue
			}
			enrichedUnit := unit
			enrichedUnit.Callees = nil
			if rule.Context.Callees {
				callees, err := allowedCallees(unit.Resolved, rule)
				if err != nil {
					return nil, err
				}
				enrichedUnit.Callees = callees
			}
			if rule.Context.Types {
				types, err := allowedTypes(unit, rule)
				if err != nil {
					return nil, err
				}
				if len(types) > 0 {
					enrichedUnit.RelatedTypes = slices.Concat(unit.RelatedTypes, types)
				}
			}
			group := -1
			// ponytail: quadratic in rule groups; index contexts if large rule sets warrant it.
			for i := range enriched {
				if slices.Equal(enriched[i].unit.Callees, enrichedUnit.Callees) &&
					slices.Equal(enriched[i].unit.RelatedTypes, enrichedUnit.RelatedTypes) {
					group = i
					break
				}
			}
			if group >= 0 {
				enriched[group].rules = append(enriched[group].rules, rule)
			} else {
				enriched = append(enriched, evaluationJob{rules: []config.Rule{rule}, unit: enrichedUnit})
			}
		}
		if len(ordinary) > 0 {
			jobs = append(jobs, evaluationJob{rules: ordinary, unit: unit})
		}
		jobs = append(jobs, enriched...)
	}
	return jobs, nil
}

// allowedCallees filters forbidden source before applying the context quotas.
func allowedCallees(resolved []parsing.CalleeContext, rule config.Rule) ([]parsing.CalleeContext, error) {
	if len(rule.Exclude) == 0 {
		return parsing.ExpandCallees(resolved), nil
	}
	allowed := make([]parsing.CalleeContext, 0, len(resolved))
	for _, callee := range resolved {
		excluded, err := scoping.Excluded(rule, callee.Path)
		if err != nil {
			return nil, err
		}
		if !excluded {
			allowed = append(allowed, callee)
		}
	}
	return parsing.ExpandCallees(allowed), nil
}

// selectClosestRegions picks each wanted region once, from its closest parent.
func selectClosestRegions(
	units []parsing.CodeUnit,
	requested map[parsing.CodeKind]parsing.CodeKind,
) []parsing.CodeUnit {
	selected := make(map[parsing.Region]selectedRegion)
	for _, parent := range units {
		parentSpan := parent.EndByte - parent.StartByte
		for _, region := range parent.Regions {
			rememberClosestRegion(selected, parent, region, parentSpan, requested)
		}
	}
	regions := make([]parsing.CodeUnit, 0, len(selected))
	for _, item := range selected {
		regions = append(regions, item.unit)
	}
	return regions
}

// rememberClosestRegion keeps a region when it has a closer parent.
func rememberClosestRegion(
	selected map[parsing.Region]selectedRegion,
	parent parsing.CodeUnit,
	region parsing.Region,
	parentSpan uint,
	requested map[parsing.CodeKind]parsing.CodeKind,
) {
	kind, ok := requested[region.Category]
	if !ok {
		return
	}
	existing, exists := selected[region]
	if exists && existing.parentSpan <= parentSpan {
		return
	}
	selected[region] = selectedRegion{
		unit:       regionCodeUnit(parent, region, kind),
		parentSpan: parentSpan,
	}
}

// expandUnitsWithRegions adds the wanted regions to the list of units.
func expandUnitsWithRegions(
	units []parsing.CodeUnit,
	regions []parsing.CodeUnit,
) []parsing.CodeUnit {
	expanded := append([]parsing.CodeUnit(nil), units...)
	expanded = append(expanded, regions...)
	sort.SliceStable(expanded, func(i, j int) bool {
		return codeUnitLess(expanded[i], expanded[j])
	})
	return expanded
}

// codeUnitLess orders code units that start at the same place.
func codeUnitLess(left parsing.CodeUnit, right parsing.CodeUnit) bool {
	if left.StartByte != right.StartByte {
		return left.StartByte < right.StartByte
	}
	rank := map[parsing.CodeKind]int{
		parsing.CodeKindType:       codeUnitRankType,
		parsing.CodeKindFunction:   codeUnitRankFunction,
		parsing.CodeKindComment:    codeUnitRankComment,
		parsing.CodeKindDocComment: codeUnitRankComment,
		parsing.CodeKindField:      codeUnitRankField,
	}
	leftRank, rightRank := codeUnitRankOther, codeUnitRankOther
	if value, ok := rank[left.Kind]; ok {
		leftRank = value
	}
	if value, ok := rank[right.Kind]; ok {
		rightRank = value
	}
	return leftRank < rightRank
}

// requestedRegionKinds returns the region kinds the rules ask for.
func requestedRegionKinds(rules []config.Rule) map[parsing.CodeKind]parsing.CodeKind {
	requested := make(map[parsing.CodeKind]parsing.CodeKind)
	for _, rule := range rules {
		for _, kind := range rule.Kinds {
			parsed, err := parsing.ParseCodeKind(kind.String())
			if err != nil {
				continue
			}
			switch parsed {
			case parsing.CodeKindComment, parsing.CodeKindDocComment,
				parsing.CodeKindField, parsing.CodeKindStatement:
				requested[parsed] = parsed
			}
		}
	}
	return requested
}

// regionCodeUnit builds a code unit for a region inside a parent.
func regionCodeUnit(
	parent parsing.CodeUnit,
	region parsing.Region,
	kind parsing.CodeKind,
) parsing.CodeUnit {
	return parsing.CodeUnit{
		Kind:           kind,
		Name:           parent.Name + ":" + string(region.Kind),
		Language:       parent.Language,
		Path:           parent.Path,
		Test:           parent.Test,
		Source:         region.Source,
		ParentSource:   parent.Source,
		RegionKind:     region.Kind,
		StartLine:      region.StartLine,
		EndLine:        region.EndLine,
		StartColumn:    region.StartColumn,
		EndColumn:      region.EndColumn,
		StartByte:      region.StartByte,
		EndByte:        region.EndByte,
		RelatedTypes:   parent.RelatedTypes,
		TypeCandidates: parent.TypeCandidates,
	}
}

// collectOutcomes gathers the findings and counts from the jobs.
func collectOutcomes(
	report *Report,
	outcomes []evaluationOutcome,
	showBelowFloor bool,
) []pendingFinding {
	pending := make([]pendingFinding, 0)
	if report.Rules == nil {
		report.Rules = make(map[string]RuleReport)
	}
	for _, outcome := range outcomes {
		report.Evaluations += outcome.evaluations
		pending = append(pending, outcome.findings...)
		if showBelowFloor {
			report.BelowFloor = append(report.BelowFloor, outcome.belowFloor...)
		}
		for id, rule := range outcome.rules {
			combined := report.Rules[id]
			combined.Description = rule.Description
			combined.Severity = rule.Severity
			combined.Decisions.Pass += rule.Decisions.Pass
			combined.Decisions.Fail += rule.Decisions.Fail
			combined.Decisions.Skip += rule.Decisions.Skip
			combined.Decisions.Abstain += rule.Decisions.Abstain
			combined.Decisions.Reported += rule.Decisions.Reported
			combined.Decisions.BelowFloor += rule.Decisions.BelowFloor
			report.Rules[id] = combined
		}
	}
	return pending
}

// appliesToKind reports whether a rule checks a given kind.
func appliesToKind(rule config.Rule, kind parsing.CodeKind) bool {
	if len(rule.Kinds) == 0 {
		return kind == parsing.CodeKindFunction || kind == parsing.CodeKindType
	}
	for _, allowed := range rule.Kinds {
		parsed, err := parsing.ParseCodeKind(allowed.String())
		if err == nil && parsed == kind {
			return true
		}
	}
	return false
}

// runJobs runs the jobs with the given number of workers and keeps their order.
func runJobs[Job any, Outcome any](
	ctx context.Context,
	jobs []Job,
	concurrency int,
	evaluate func(context.Context, Job) (Outcome, error),
	progress func(done, total int),
) ([]Outcome, error) {
	if len(jobs) == 0 {
		return nil, nil
	}
	if concurrency > len(jobs) {
		concurrency = len(jobs)
	}

	jobContext, cancel := context.WithCancel(ctx)
	defer cancel()

	scheduled := make([]scheduledWork[Job, Outcome], len(jobs))
	indices := make(chan int, len(jobs))
	for index, job := range jobs {
		scheduled[index].job = job
		indices <- index
	}
	close(indices)
	firstError := runWorkers(jobContext, cancel, scheduled, indices, concurrency, evaluate, progress)
	if firstError != nil {
		return nil, firstError
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	outcomes := make([]Outcome, len(scheduled))
	for index, item := range scheduled {
		outcomes[index] = item.outcome
	}
	return outcomes, nil
}

// runWorkers runs the jobs and stops the others when one fails.
func runWorkers[Job any, Outcome any](
	ctx context.Context,
	cancel context.CancelFunc,
	scheduled []scheduledWork[Job, Outcome],
	indices <-chan int,
	concurrency int,
	evaluate func(context.Context, Job) (Outcome, error),
	progress func(done, total int),
) error {
	var workers sync.WaitGroup
	var errorOnce sync.Once
	var firstError error
	var progressMu sync.Mutex
	completed := 0

	workers.Add(concurrency)
	for range concurrency {
		go func() {
			defer workers.Done()
			for index := range indices {
				if ctx.Err() != nil {
					return
				}
				outcome, err := evaluate(ctx, scheduled[index].job)
				if err != nil {
					errorOnce.Do(func() {
						firstError = err
						cancel()
					})
					return
				}
				scheduled[index].outcome = outcome
				if progress != nil {
					progressMu.Lock()
					completed++
					progress(completed, len(scheduled))
					progressMu.Unlock()
				}
			}
		}()
	}
	workers.Wait()
	return firstError
}

// evaluateJob asks the evaluator about one job and builds its findings.
func evaluateJob(
	ctx context.Context,
	evaluator evaluation.Evaluator,
	job evaluationJob,
	cfg config.Config,
) (evaluationOutcome, error) {
	results, err := evaluator.Evaluate(ctx, evaluation.Batch{
		Rules:    job.rules,
		CodeUnit: job.unit,
	})
	if errors.Is(err, evaluation.ErrOversized) {
		return evaluationOutcome{}, nil
	}
	if err != nil {
		return evaluationOutcome{}, fmt.Errorf(
			"evaluate %s:%d: %w",
			job.unit.Path,
			job.unit.StartLine,
			err,
		)
	}

	outcome := evaluationOutcome{
		findings: make([]pendingFinding, 0),
		rules:    make(map[string]RuleReport, len(job.rules)),
	}
	for _, rule := range job.rules {
		result, ok := results[rule.ID]
		if !ok {
			return evaluationOutcome{}, fmt.Errorf(
				"invalid result for rule %q at %s:%d: result is missing",
				rule.ID,
				job.unit.Path,
				job.unit.StartLine,
			)
		}
		if err := result.Validate(); err != nil {
			return evaluationOutcome{}, fmt.Errorf(
				"invalid result for rule %q at %s:%d: %w",
				rule.ID,
				job.unit.Path,
				job.unit.StartLine,
				err,
			)
		}
		outcome.evaluations++

		floor := cfg.FailProbabilityFloor(rule)
		reported := result.FailProbability >= floor
		notable := result.FailProbability >= floor/2 && result.FailProbability < floor
		if job.unit.Kind != parsing.CodeKindRegion {
			decisions := Decisions{}
			switch result.Status {
			case evaluation.StatusPass:
				decisions.Pass = 1
			case evaluation.StatusFail:
				decisions.Fail = 1
			case evaluation.StatusSkip:
				decisions.Skip = 1
			case evaluation.StatusAbstain:
				decisions.Abstain = 1
			}
			if reported {
				decisions.Reported = 1
			} else if notable {
				decisions.BelowFloor = 1
			}
			outcome.rules[rule.ID] = RuleReport{
				Description: rule.Description,
				Severity:    rule.Severity,
				Decisions:   decisions,
			}
		}
		if !reported && !notable {
			continue
		}
		finding := Finding{
			RuleID:          rule.ID,
			Severity:        rule.Severity,
			Status:          result.Status,
			Path:            job.unit.Path,
			Language:        job.unit.Language.String(),
			Kind:            job.unit.Kind,
			Name:            job.unit.Name,
			StartLine:       job.unit.StartLine,
			EndLine:         job.unit.EndLine,
			StartColumn:     job.unit.StartColumn,
			EndColumn:       job.unit.EndColumn,
			Snippet:         job.unit.Source,
			Locations:       directUnitLocations(job.unit),
			FailProbability: result.FailProbability,
		}
		if !reported {
			outcome.belowFloor = append(outcome.belowFloor, BelowFloorFinding{Finding: finding})
			continue
		}
		outcome.findings = append(outcome.findings, pendingFinding{
			finding: finding,
			rule:    rule,
			unit:    job.unit,
		})
	}
	return outcome, nil
}

// directUnitLocations returns the place of a region that is checked on its own.
func directUnitLocations(unit parsing.CodeUnit) []Location {
	if unit.RegionKind == "" {
		return nil
	}
	return []Location{{
		Category:    unit.Kind.String(),
		Kind:        string(unit.RegionKind),
		Source:      unit.Source,
		StartLine:   unit.StartLine,
		EndLine:     unit.EndLine,
		StartColumn: unit.StartColumn,
		EndColumn:   unit.EndColumn,
	}}
}

// localizeFindings points at the exact places inside failed units.
// One request covers every matching region of a finding.
func localizeFindings(
	ctx context.Context,
	evaluator evaluation.Evaluator,
	findings []pendingFinding,
	concurrency int,
	cfg config.Config,
) (int, error) {
	jobs := localizationJobs(findings)
	outcomes, err := runJobs(
		ctx,
		jobs,
		concurrency,
		func(ctx context.Context, job localizationJob) (localizationOutcome, error) {
			return evaluateLocalizationJob(ctx, evaluator, job, cfg)
		},
		nil,
	)
	if err != nil {
		return 0, err
	}
	for _, outcome := range outcomes {
		item := &findings[outcome.findingIndex]
		item.finding.Locations = append(item.finding.Locations, outcome.locations...)
	}
	return len(jobs), nil
}

// localizationJobs builds one job per finding that has matching regions.
func localizationJobs(findings []pendingFinding) []localizationJob {
	jobs := make([]localizationJob, 0)
	for findingIndex, item := range findings {
		regions := make([]parsing.Region, 0)
		for _, region := range item.unit.Regions {
			if !localizesTo(item.rule, region.Category) {
				continue
			}
			regions = append(regions, region)
			if len(regions) == 24 {
				break
			}
		}
		if len(regions) == 0 {
			continue
		}
		jobs = append(jobs, localizationJob{
			findingIndex: findingIndex,
			rule:         item.rule,
			parent:       item.unit,
			regions:      regions,
		})
	}
	return jobs
}

// evaluateLocalizationJob asks one noul per region and keeps those at the threshold.
func evaluateLocalizationJob(
	ctx context.Context,
	evaluator evaluation.Evaluator,
	job localizationJob,
	cfg config.Config,
) (localizationOutcome, error) {
	results, err := evaluator.Evaluate(ctx, evaluation.Batch{
		Rules:    []config.Rule{job.rule},
		CodeUnit: job.parent,
		Regions:  job.regions,
	})
	if errors.Is(err, evaluation.ErrOversized) {
		return localizationOutcome{findingIndex: job.findingIndex}, nil
	}
	if err != nil {
		return localizationOutcome{}, fmt.Errorf(
			"localize rule %q at %s:%d: %w",
			job.rule.ID,
			job.parent.Path,
			job.parent.StartLine,
			err,
		)
	}
	floor := cfg.FailProbabilityFloor(job.rule)
	outcome := localizationOutcome{findingIndex: job.findingIndex}
	for index, region := range job.regions {
		id := fmt.Sprintf("r%d", index)
		result, ok := results[id]
		if !ok {
			return localizationOutcome{}, fmt.Errorf(
				"invalid localization result for rule %q region %s at %s:%d: result is missing",
				job.rule.ID,
				id,
				job.parent.Path,
				region.StartLine,
			)
		}
		if err := result.Validate(); err != nil {
			return localizationOutcome{}, fmt.Errorf(
				"invalid localization result for rule %q region %s at %s:%d: %w",
				job.rule.ID,
				id,
				job.parent.Path,
				region.StartLine,
				err,
			)
		}
		if result.FailProbability < floor {
			continue
		}
		outcome.locations = append(outcome.locations, Location{
			Category:    region.Category.String(),
			Kind:        string(region.Kind),
			Source:      region.Source,
			StartLine:   region.StartLine,
			EndLine:     region.EndLine,
			StartColumn: region.StartColumn,
			EndColumn:   region.EndColumn,
		})
	}
	return outcome, nil
}

// localizesTo reports whether a rule can point at a given kind of region.
func localizesTo(rule config.Rule, category parsing.CodeKind) bool {
	for _, allowed := range rule.Localize {
		parsed, err := parsing.ParseCodeKind(allowed.String())
		if err == nil && parsed == category {
			return true
		}
	}
	return false
}

// HasFailures reports whether a reported failure meets the severity threshold.
func (report Report) HasFailures(threshold config.Severity) bool {
	for _, finding := range report.Findings {
		if finding.Severity >= threshold {
			return true
		}
	}
	return false
}

// discover lists the supported files under the requested paths.
func discover(ctx context.Context, root *os.Root, requested []string, extractor *parsing.Extractor) ([]string, error) {
	seen := make(map[string]struct{})
	explicit := make(map[string]struct{})
	broad := false
	for _, requestedPath := range requested {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		walked, err := discoverRequestedPath(root, requestedPath, extractor, seen, explicit)
		if err != nil {
			return nil, err
		}
		broad = broad || walked
	}
	if broad {
		if err := removeGitIgnored(ctx, root.Name(), seen, explicit); err != nil {
			return nil, err
		}
	}
	for file := range explicit {
		seen[file] = struct{}{}
	}
	return sortedDiscoveredFiles(seen), nil
}

// discoverRequestedPath lists the supported files under one path.
func discoverRequestedPath(
	root *os.Root,
	requestedPath string,
	extractor *parsing.Extractor,
	seen map[string]struct{},
	explicit map[string]struct{},
) (bool, error) {
	path := resolveRequestedPath(root.Name(), requestedPath)
	relative, err := relativeProjectPath(root.Name(), path)
	if err != nil {
		return false, err
	}
	info, err := root.Stat(filepath.FromSlash(relative))
	if err != nil {
		return false, fmt.Errorf("inspect %q: %w", requestedPath, err)
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() && extractor.Supports(path) {
			explicit[path] = struct{}{}
		}
		return false, nil
	}
	if err := fs.WalkDir(root.FS(), relative, func(
		candidate string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		return collectWalkEntry(root, relative, candidate, entry, walkErr, extractor, seen)
	}); err != nil {
		return true, fmt.Errorf("walk %q: %w", requestedPath, err)
	}
	return true, nil
}

// resolveRequestedPath turns a requested path into a full path.
func resolveRequestedPath(root string, requestedPath string) string {
	if filepath.IsAbs(requestedPath) {
		return filepath.Clean(requestedPath)
	}
	return filepath.Clean(filepath.Join(root, requestedPath))
}

// collectWalkEntry adds a file from a directory walk and skips ignored folders.
func collectWalkEntry(
	root *os.Root,
	walkRoot string,
	candidate string,
	entry fs.DirEntry,
	walkErr error,
	extractor *parsing.Extractor,
	seen map[string]struct{},
) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() {
		if candidate != walkRoot {
			if _, ignored := ignoredDirectoryNames[entry.Name()]; ignored {
				return filepath.SkipDir
			}
		}
		return nil
	}
	if entry.Type()&fs.ModeSymlink != 0 {
		info, err := root.Stat(filepath.FromSlash(candidate))
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
	} else if !entry.Type().IsRegular() {
		return nil
	}
	if extractor.Supports(candidate) {
		seen[filepath.Join(root.Name(), filepath.FromSlash(candidate))] = struct{}{}
	}
	return nil
}

// sortedDiscoveredFiles returns the found files in a stable order.
func sortedDiscoveredFiles(seen map[string]struct{}) []string {
	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

var ignoredDirectoryNames = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	".venv":        {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
}
