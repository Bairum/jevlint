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

	"jevlint/internal/config"
	"jevlint/internal/evals"
	"jevlint/internal/evaluation"
	"jevlint/internal/parsing"
)

// evalOptions holds the settings for an eval run.
type evalOptions struct {
	output outputContext
	eval   evalContext
}

// evalContext holds the settings that shape an eval.
type evalContext struct {
	configPath  string
	evalsPath   string
	ruleID      string
	concurrency int
	cache       cacheMode
	verbose     bool
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
			configPath:  *configPath,
			evalsPath:   *evalsPath,
			ruleID:      *ruleID,
			concurrency: *concurrency,
			cache:       mode,
			verbose:     *verbose,
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
	absoluteConfig, cfg, extractor, exitCode := loadProject(options.eval.configPath, stderr)
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
	evaluator, err := evaluation.NewTypeSafeFromEnvWithOptions(
		evaluation.TypeSafeOptions{
			Cache:   resultCache,
			Refresh: options.eval.cache.shouldRefresh(),
		},
		os.Getenv,
	)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	report, err := evals.Run(
		ctx,
		document,
		cfg,
		extractor,
		evaluator,
		evals.Options{
			Root:        filepath.Dir(absoluteConfig),
			Concurrency: options.eval.concurrency,
		},
	)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	if err := writeEvalReport(stdout, report, options.output, options.eval.verbose); err != nil {
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

// writeEvalReport prints the eval results in the chosen format.
func writeEvalReport(
	writer io.Writer,
	report evals.Report,
	output outputContext,
	verbose bool,
) error {
	if output.format == formatJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if !verbose {
			report = stripUnits(report)
		}
		return encoder.Encode(report)
	}
	style := outputStyle{color: shouldUseColor(output.color, writer, output.hints)}
	writeRunLegend(writer, style)
	writeRunText(writer, style, report, verbose, "eval cases")
	return nil
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

// writeRunLegend explains how eval outcomes are decided.
func writeRunLegend(writer io.Writer, style outputStyle) {
	fmt.Fprintln(writer, style.paint("1", "legend"))
	fmt.Fprintln(writer, "  pass          the rule passed every code unit")
	fmt.Fprintln(writer, "  fail          at least one violation at or above the confidence floor")
	fmt.Fprintln(writer, "  inconclusive  a failure below the floor, or no unit returned an explicit pass")
	fmt.Fprintln(writer, "  confidence    Jev's score for a decision; below the floor a fail is not reported")
	fmt.Fprintln(writer)
}

// writeRunText prints case results grouped by rule.
func writeRunText(
	writer io.Writer,
	style outputStyle,
	report evals.Report,
	verbose bool,
	label string,
) {
	currentRule := ""
	for _, result := range report.Cases {
		if result.Rule != currentRule {
			if currentRule != "" {
				fmt.Fprintln(writer)
			}
			fmt.Fprintln(writer, style.paint("1", result.Rule))
			currentRule = result.Rule
		}
		writeRunCase(writer, style, result, verbose)
	}
	if len(report.Cases) > 0 {
		fmt.Fprintln(writer)
	}
	fmt.Fprintf(
		writer,
		"%d/%d %s matched expectations, %d inconclusive\n",
		report.Matched,
		report.Total,
		label,
		report.Inconclusive,
	)
	fmt.Fprintf(
		writer,
		"  reportable failures: %d   hidden failures: %d\n",
		report.ReportableFailures,
		report.HiddenFailures,
	)
}

// writeRunCase prints the result of one case.
func writeRunCase(writer io.Writer, style outputStyle, result evals.Result, verbose bool) {
	label := result.File
	if result.Name != "" {
		label = result.Name + "  " + result.File
	}
	switch {
	case result.Matched:
		fmt.Fprintf(writer, "  %s\n", style.paint("32", "✓ "+label))
	case result.Actual == evals.OutcomeInconclusive:
		fmt.Fprintf(writer, "  %s\n", style.paint("33", "? "+label))
	default:
		fmt.Fprintf(writer, "  %s\n", style.paint("31", "✗ "+label))
	}
	fmt.Fprintf(writer, "      expected: %s\n", result.Expected)
	fmt.Fprintf(writer, "      actual: %s\n", result.Actual)
	fmt.Fprintf(writer, "      floor: %.2f   jev: %s\n", result.Floor, jevSummary(result.Decisions))
	if result.Decisions.Fail > 0 && result.Confidence != nil {
		fmt.Fprintf(writer, "      fail confidence: %.2f\n", *result.Confidence)
	}
	if verbose {
		writeUnitDecisions(writer, result)
	}
}

// jevSummary prints the raw rule answers for one case.
func jevSummary(decisions evals.Decisions) string {
	if decisions.Fail == 0 {
		return fmt.Sprintf(
			"%d pass, %d fail, %d skip, %d abstain",
			decisions.Pass,
			decisions.Fail,
			decisions.Skip,
			decisions.Abstain,
		)
	}
	return fmt.Sprintf(
		"%d pass, %d fail (%d reportable, %d hidden), %d skip, %d abstain",
		decisions.Pass,
		decisions.Fail,
		decisions.ReportableFail,
		decisions.HiddenFail,
		decisions.Skip,
		decisions.Abstain,
	)
}

// writeUnitDecisions prints the per-unit Jev answers for one case.
func writeUnitDecisions(writer io.Writer, result evals.Result) {
	if len(result.Units) == 0 {
		return
	}
	fmt.Fprintln(writer, "      units:")
	for _, unit := range result.Units {
		note := ""
		if unit.Status == evaluation.StatusFail {
			if unit.Reportable {
				note = "  (reportable)"
			} else {
				note = "  (below floor, not reported)"
			}
		}
		fmt.Fprintf(
			writer,
			"        %s %s  lines %d-%d  %s  %.2f%s\n",
			unit.Kind,
			unit.Name,
			unit.StartLine,
			unit.EndLine,
			unit.Status,
			unit.Confidence,
			note,
		)
	}
}
