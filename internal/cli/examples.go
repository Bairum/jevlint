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

// examplesOptions holds the settings for an examples run.
type examplesOptions struct {
	output   outputContext
	examples examplesContext
}

// examplesContext holds the settings that shape an examples run.
type examplesContext struct {
	configPath   string
	examplesPath string
	ruleID       string
	concurrency  int
	cache        cacheMode
}

// parseExamplesOptions reads the examples flags.
func parseExamplesOptions(args []string, stderr io.Writer) (examplesOptions, int, bool) {
	flags := flag.NewFlagSet("examples", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, examplesUsage)
	}
	clearCache := flags.Bool("clear-cache", false, "clear cached evaluations")
	color := flags.String("color", "auto", "color output")
	configPath := flags.String("config", defaultConfigFile, "rule configuration")
	concurrency := flags.Int("concurrency", defaultCheckConcurrency, "maximum concurrent Jev requests")
	examplesPath := flags.String("examples", "", "examples directory")
	format := flags.String("format", "text", "output format")
	refreshCache := flags.Bool("refresh-cache", false, "refresh cached evaluations")
	ruleID := flags.String("rule", "", "evaluate only this rule")
	flagArgs, leftover, err := splitFlagsAndPaths(flags, args)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return examplesOptions{}, exitUsageError, false
	}
	if err := flags.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return examplesOptions{}, exitSuccess, false
		}
		return examplesOptions{}, exitUsageError, false
	}
	if len(leftover) > 0 {
		fmt.Fprintf(stderr, "jevlint: examples does not take path arguments\n")
		return examplesOptions{}, exitUsageError, false
	}
	parsedFormat, parsedColor, exitCode, valid := parseOutputOptions(
		*format,
		*color,
		*concurrency,
		stderr,
	)
	if !valid {
		return examplesOptions{}, exitCode, false
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
	return examplesOptions{
		output: outputContext{
			format: parsedFormat,
			color:  parsedColor,
		},
		examples: examplesContext{
			configPath:   *configPath,
			examplesPath: *examplesPath,
			ruleID:       *ruleID,
			concurrency:  *concurrency,
			cache:        mode,
		},
	}, exitSuccess, true
}

// executeExamples loads the examples, runs them, and writes the report.
func executeExamples(
	ctx context.Context,
	options examplesOptions,
	stdout io.Writer,
	stderr io.Writer,
	userCacheDir func() (string, error),
) int {
	absoluteConfig, cfg, extractor, exitCode := loadProject(options.examples.configPath, stderr)
	if exitCode != 0 {
		return exitCode
	}
	document, exitCode := loadExamplesDocument(
		absoluteConfig,
		options.examples.examplesPath,
		cfg,
		extractor,
		stderr,
	)
	if exitCode != 0 {
		return exitCode
	}
	document, err := document.FilterRule(options.examples.ruleID)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	resultCache, exitCode := openResultCache(
		filepath.Dir(absoluteConfig),
		options.examples.cache,
		stderr,
		userCacheDir,
	)
	if exitCode != 0 {
		return exitCode
	}
	evaluator, err := evaluation.NewTypeSafeFromEnvWithOptions(
		evaluation.TypeSafeOptions{
			Cache:   resultCache,
			Refresh: options.examples.cache.shouldRefresh(),
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
			Concurrency: options.examples.concurrency,
		},
	)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	if err := writeExamplesReport(stdout, report, options.output); err != nil {
		fmt.Fprintf(stderr, "jevlint: write output: %v\n", err)
		return exitUsageError
	}
	if report.Mismatched > 0 {
		return exitHasFindings
	}
	return exitSuccess
}

// loadExamplesDocument loads the examples directory for the project.
func loadExamplesDocument(
	absoluteConfig string,
	examplesPath string,
	cfg config.Config,
	extractor *parsing.Extractor,
	stderr io.Writer,
) (evals.Document, int) {
	path := examplesPath
	if path == "" {
		path = filepath.Join(filepath.Dir(absoluteConfig), evals.DefaultExamplesDir)
	}
	document, err := evals.LoadExamples(path, cfg, extractor)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return evals.Document{}, exitUsageError
	}
	return document, exitSuccess
}

// writeExamplesReport prints the example results in the chosen format.
func writeExamplesReport(
	writer io.Writer,
	report evals.Report,
	output outputContext,
) error {
	if output.format == formatJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	style := outputStyle{color: shouldUseColor(output.color, writer, output.hints)}
	writeRunText(writer, style, report, "examples")
	return nil
}
