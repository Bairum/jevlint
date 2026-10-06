package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/evals"
	"github.com/codegirl-007/jevlint/internal/evaluation"
	"github.com/codegirl-007/jevlint/internal/packs"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

// evalOptions holds the settings for an eval run.
type evalOptions struct {
	output outputContext
	eval   evalContext
}

// evalContext holds the settings that shape an eval.
type evalContext struct {
	configPath   string
	evalsPath    string
	ruleID       string
	includePacks bool
	concurrency  int
	cache        cacheMode
	verbose      bool
}

// parseEvalOptions reads the eval flags.
func parseEvalOptions(args []string, stderr io.Writer) (evalOptions, int, bool) {
	flags := flag.NewFlagSet("eval", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, evalUsage)
	}
	clearCache := flags.Bool("clear-cache", false, "clear cached evaluations")
	color := flags.String("color", "auto", "color output")
	configPath := flags.String("config", defaultConfigFile, "rule configuration")
	concurrency := flags.Int("concurrency", defaultCheckConcurrency, "maximum concurrent Jev requests")
	evalsPath := flags.String("evals", "", "eval cases")
	format := flags.String("format", "text", "output format")
	includePacks := flags.Bool("packs", false, "also run pack evals")
	refreshCache := flags.Bool("refresh-cache", false, "refresh cached evaluations")
	ruleID := flags.String("rule", "", "evaluate only this rule")
	verbose := flags.Bool("verbose", false, "show per-unit Jev decisions")
	flagArgs, leftover, err := splitFlagsAndPaths(flags, args)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return evalOptions{}, exitUsageError, false
	}
	if err := flags.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return evalOptions{}, exitSuccess, false
		}
		return evalOptions{}, exitUsageError, false
	}
	if len(leftover) > 0 {
		fmt.Fprintf(stderr, "jevlint: eval does not take path arguments\n")
		return evalOptions{}, exitUsageError, false
	}
	parsedFormat, parsedColor, exitCode, valid := parseOutputOptions(
		*format,
		*color,
		*concurrency,
		stderr,
	)
	if !valid {
		return evalOptions{}, exitCode, false
	}
	var mode cacheMode
	switch {
	case *clearCache && *refreshCache:
		mode = cacheClearAndRefresh
	case *clearCache:
		mode = cacheClear
	case *refreshCache:
		mode = cacheRefresh
	default:
		mode = cacheReadWrite
	}
	return evalOptions{
		output: outputContext{
			format: parsedFormat,
			color:  parsedColor,
		},
		eval: evalContext{
			configPath:   *configPath,
			evalsPath:    *evalsPath,
			ruleID:       *ruleID,
			includePacks: *includePacks,
			concurrency:  *concurrency,
			cache:        mode,
			verbose:      *verbose,
		},
	}, exitSuccess, true
}

// executeEval loads the cases, runs them, and writes the report.
func executeEval(
	ctx context.Context,
	options evalOptions,
	stdout io.Writer,
	stderr io.Writer,
	userCacheDir func() (string, error),
) int {
	absoluteConfig, cfg, extractor, loadedPacks, exitCode := loadProject(
		options.eval.configPath,
		stderr,
		userCacheDir,
	)
	if exitCode != 0 {
		return exitCode
	}
	document, exitCode := loadEvalDocument(
		absoluteConfig,
		options.eval.evalsPath,
		cfg,
		extractor,
		stderr,
	)
	if exitCode != 0 {
		return exitCode
	}
	if options.eval.includePacks {
		packDocument, packExit := loadPackEvalDocuments(loadedPacks, cfg, extractor, stderr)
		if packExit != 0 {
			return packExit
		}
		document = document.Concat(packDocument)
	}
	document, err := document.FilterRule(options.eval.ruleID)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	resultCache, exitCode := openResultCache(
		filepath.Dir(absoluteConfig),
		options.eval.cache,
		stderr,
		userCacheDir,
	)
	if exitCode != 0 {
		return exitCode
	}
	evaluator, err := evaluation.NewClientFromEnv(
		evaluation.Options{
			Cache:   resultCache,
			Refresh: options.eval.cache.shouldRefresh(),
			Budget:  jevBudget,
			Logf:    debugLogger(stderr),
			Warnf:   warnLogger(stderr),
		},
		os.Getenv,
	)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	text := options.output.format != formatJSON
	style := outputStyle{color: shouldUseColor(options.output.color, stdout, options.output.hints)}
	runOptions := evals.Options{
		Root:        filepath.Dir(absoluteConfig),
		Concurrency: options.eval.concurrency,
	}
	if text {
		writeRunLegend(stdout, style)
		printer := &streamPrinter{writer: stdout, style: style}
		runOptions.OnUnit = printer.unit
		runOptions.OnResult = printer.caseResult
	}
	report, err := evals.Run(ctx, document, cfg, extractor, evaluator, runOptions)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	if text {
		writeRunSummary(stdout, style, report, "eval cases")
	} else if err := writeEvalJSON(stdout, report, options.eval.verbose); err != nil {
		fmt.Fprintf(stderr, "jevlint: write output: %v\n", err)
		return exitUsageError
	}
	if report.Mismatched > 0 || report.Inconclusive > 0 {
		return exitHasFindings
	}
	return exitSuccess
}

// loadEvalDocument loads the eval file for the project.
func loadEvalDocument(
	absoluteConfig string,
	evalsPath string,
	cfg config.Config,
	extractor *parsing.Extractor,
	stderr io.Writer,
) (evals.Document, int) {
	path := evalsPath
	if path == "" {
		path = filepath.Join(filepath.Dir(absoluteConfig), evals.DefaultFile)
	}
	document, err := evals.Load(path, cfg, extractor)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return evals.Document{}, exitUsageError
	}
	return document, exitSuccess
}

// writeEvalJSON prints the buffered report as JSON.
func writeEvalJSON(writer io.Writer, report evals.Report, verbose bool) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if !verbose {
		report = stripUnits(report)
	}
	return encoder.Encode(report)
}

func loadPackEvalDocuments(
	loadedPacks []packs.Loaded,
	cfg config.Config,
	extractor *parsing.Extractor,
	stderr io.Writer,
) (evals.Document, int) {
	combined := evals.Document{Version: 1}
	for _, pack := range loadedPacks {
		path, err := pack.EvalPath()
		if err != nil {
			fmt.Fprintf(stderr, "jevlint: %v\n", err)
			return evals.Document{}, exitUsageError
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(stderr, "jevlint: %v\n", err)
			return evals.Document{}, exitUsageError
		}
		document, err := evals.Load(path, cfg, extractor)
		if err != nil {
			fmt.Fprintf(stderr, "jevlint: %v\n", err)
			return evals.Document{}, exitUsageError
		}
		combined = combined.Concat(document)
	}
	return combined, exitSuccess
}

// streamPrinter prints unit decisions and case results as they arrive.
type streamPrinter struct {
	writer io.Writer
	style  outputStyle
	mu     sync.Mutex
	rule   string
	file   string
}

// unit prints one Jev answer for one code unit.
func (printer *streamPrinter) unit(evalCase evals.Case, unit evals.UnitDecision) {
	printer.mu.Lock()
	defer printer.mu.Unlock()

	if evalCase.Rule != printer.rule {
		fmt.Fprintln(printer.writer, printer.style.paint("1", evalCase.Rule))
		printer.rule = evalCase.Rule
		printer.file = ""
	}
	if evalCase.File != printer.file {
		fmt.Fprintf(printer.writer, "  %s\n", printer.style.paint("36", evalCase.File))
		printer.file = evalCase.File
	}
	fmt.Fprintf(
		printer.writer,
		"    %s %s  lines %d-%d  %s  %.2f\n",
		unit.Kind,
		unit.Name,
		unit.StartLine,
		unit.EndLine,
		unit.Status,
		unit.FailProbability,
	)
}

// caseResult prints the expected and actual outcome of one case.
func (printer *streamPrinter) caseResult(result evals.Result) {
	printer.mu.Lock()
	defer printer.mu.Unlock()

	mark := printer.style.paint("32", "✓")
	if !result.Matched {
		mark = printer.style.paint("31", "✗")
	}
	if result.Actual == evals.OutcomeInconclusive {
		mark = printer.style.paint("33", "?")
	}
	fmt.Fprintf(
		printer.writer,
		"    %s expected: %s   actual: %s\n",
		mark,
		result.Expected,
		result.Actual,
	)
}

func writeRunSummary(writer io.Writer, style outputStyle, report evals.Report, label string) {
	fmt.Fprintln(writer)
	fmt.Fprintf(
		writer,
		"%d/%d %s matched expectations, %d inconclusive\n",
		report.Matched,
		report.Total,
		label,
		report.Inconclusive,
	)
	models := report.Models
	if len(models) == 0 {
		models = []string{"none"}
	}
	fmt.Fprintf(
		writer,
		"models %s · %d requests · %d input tokens · %d output tokens\n",
		strings.Join(models, ", "),
		report.Usage.Requests,
		report.Usage.InputTokens,
		report.Usage.OutputTokens,
	)
	if len(report.Oversized) > 0 {
		fmt.Fprintf(writer, "warning: skipped %d oversized units\n", len(report.Oversized))
	}
}

// stripUnits returns the report without the per-unit decisions.
func stripUnits(report evals.Report) evals.Report {
	cases := make([]evals.Result, len(report.Cases))
	copy(cases, report.Cases)
	for index := range cases {
		cases[index].Units = nil
	}
	report.Cases = cases
	return report
}

// writeRunLegend explains how eval outcomes and decisions are classified.
func writeRunLegend(writer io.Writer, style outputStyle) {
	fmt.Fprintln(writer, style.paint("1", "legend"))
	fmt.Fprintln(writer, "  Each case runs one rule on every code unit in one fixture. Jev answers")
	fmt.Fprintln(writer, "  pass, fail, skip, or abstain for each unit and scores its fail probability.")
	fmt.Fprintln(writer, "    matched       the rule decided what the case expected")
	fmt.Fprintln(writer, "    mismatched    the rule decided the opposite of what the case expected")
	fmt.Fprintln(writer, "    inconclusive  the rule did not clearly pass or fail")
	fmt.Fprintln(writer)
}
