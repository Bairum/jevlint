package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"jevlint/internal/evals"
	"jevlint/internal/evaluation"
	"jevlint/internal/parsing"
)

type evalProject struct {
	config     string
	source     string
	evals      string
	extraFiles map[string]string
}

func TestEvalReportsJSONWithConfidence(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc JoinInCode() {\n\tprintln(\"join\")\n}\n",
		evals: `{
			"version": 1,
			"cases": [{
				"name": "join-in-code",
				"rule": "database-joins",
				"file": "sample.go",
				"expect": "fail"
			}]
		}`,
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{
			"model": "jev-test",
			"answers": {
				"database-joins": {
					"type": "choice",
					"choice": "fail",
					"confidence": 0.92
				}
			}
		}`)
	}))
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"eval",
			"--config", filepath.Join(root, "jevlint.json"),
			"--format", "json",
			"--color", "never",
		},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}

	var report evals.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output: %v\noutput: %s", err, stdout.String())
	}
	if report.Matched != 1 || report.Total != 1 {
		t.Fatalf("report = %#v", report)
	}
	if report.Cases[0].Confidence == nil || *report.Cases[0].Confidence != 0.92 {
		t.Fatalf("confidence = %v", report.Cases[0].Confidence)
	}
}

func TestEvalMismatchExitsOne(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc JoinInCode() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "fail"}]
		}`,
	})
	server := passingJevServer(t)
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json"), "--color", "never"},
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("Run() exit code = %d, want 1; stderr = %q stdout = %q", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "0/1 eval cases matched expectations") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestEvalFiltersRuleFlag(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		config: `{
			"languages": {"go": {}},
			"rules": [
				{"id": "database-joins", "description": "Join in the database.", "severity": "error"},
				{"id": "naming", "description": "Use clear names.", "severity": "warning"}
			]
		}`,
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [
				{"rule": "database-joins", "file": "sample.go", "expect": "pass"},
				{"rule": "naming", "file": "other.go", "expect": "pass"}
			]
		}`,
		extraFiles: map[string]string{
			"other.go": "package sample\n\nfunc Other() {}\n",
		},
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{
			"model": "jev-test",
			"answers": {
				"database-joins": {"type": "choice", "choice": "pass", "confidence": 1},
				"naming": {"type": "choice", "choice": "fail", "confidence": 1}
			}
		}`)
	}))
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"eval",
			"--config", filepath.Join(root, "jevlint.json"),
			"--rule", "database-joins",
			"--format", "json",
		},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	var report evals.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Total != 1 || report.Cases[0].Rule != "database-joins" {
		t.Fatalf("report = %#v", report)
	}
}

func TestEvalEvalsFlagUsesAlternateFile(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals:  `{"version":1,"cases":[]}`,
	})
	alternate := filepath.Join(root, "alt-evals.json")
	if err := os.WriteFile(alternate, []byte(`{
		"version": 1,
		"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "pass"}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	server := passingJevServer(t)
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"eval",
			"--config", filepath.Join(root, "jevlint.json"),
			"--evals", alternate,
			"--format", "json",
		},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
}

func TestEvalRejectsUnknownRule(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "missing-rule", "file": "sample.go", "expect": "pass"}]
		}`,
	})
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown eval rule") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEvalRejectsMissingFixture(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "gone.go", "expect": "pass"}]
		}`,
	})
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "eval fixture") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEvalRejectsUnsupportedExtension(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "notes.txt", "expect": "pass"}]
		}`,
		extraFiles: map[string]string{"notes.txt": "notes"},
	})
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "unsupported language") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEvalRejectsInvalidExpect(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "warn"}]
		}`,
	})
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2", exitCode)
	}
}

func TestEvalRejectsDuplicateCaseIdentity(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [
				{"rule": "database-joins", "file": "sample.go", "expect": "pass"},
				{"rule": "database-joins", "file": "./sample.go", "expect": "fail"}
			]
		}`,
	})
	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "duplicate eval case") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEvalRequiresApplicableUnits(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "pass"}]
		}`,
	})
	server := passingJevServer(t)
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no applicable code units") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEvalReusesCacheOnSecondRun(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "pass"}]
		}`,
	})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{
			"model": "jev-test",
			"answers": {
				"database-joins": {"type": "choice", "choice": "pass", "confidence": 1}
			}
		}`)
	}))
	defer server.Close()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	setEvalEnv(t, server.URL)

	args := []string{"eval", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"}
	for range 2 {
		var stdout, stderr bytes.Buffer
		exitCode := runCLI(context.Background(), args, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("Run() exit code = %d; stderr = %q", exitCode, stderr.String())
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("Jev calls = %d, want 1 after cache hit", calls.Load())
	}
}

func TestCheckDoesNotLoadEvals(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc Ready() {}\n",
		evals:  `{not valid json`,
	})
	server := passingJevServer(t)
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"check", "--config", filepath.Join(root, "jevlint.json"), "."},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("check exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
}

func TestEvalRejectsChangedFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--changed"},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2", exitCode)
	}
}

func writeEvalProject(t *testing.T, project evalProject) string {
	t.Helper()
	root := t.TempDir()
	config := project.config
	if config == "" {
		config = `{
			"languages": {"go": {}},
			"rules": [{
				"id": "database-joins",
				"description": "Join related records in the database.",
				"severity": "error",
				"include": ["src/**/*.go"],
				"exclude": ["fixtures/**"]
			}]
		}`
	}
	if err := os.WriteFile(filepath.Join(root, "jevlint.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(project.source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, evals.DefaultFile), []byte(project.evals), 0o600); err != nil {
		t.Fatal(err)
	}
	for relative, contents := range project.extraFiles {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func setEvalEnv(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("TYPESAFE_API_KEY", "sk-test")
	t.Setenv("TYPESAFE_BASE_URL", baseURL)
	t.Setenv("TYPESAFE_DEFAULT_MODEL", "jev-test")
}

func TestEvalInconclusiveExitsOne(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		config: `{
			"languages": {"go": {}},
			"rules": [{
				"id": "database-joins",
				"description": "Join related records in the database.",
				"severity": "error",
				"allowSkip": true
			}]
		}`,
		source: "package sample\n\nfunc Ready() {}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "pass"}]
		}`,
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"answers":{"database-joins":{"type":"choice","choice":"skip","confidence":0.9}}}`)
	}))
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"eval", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"},
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("Run() exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	var report evals.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Inconclusive != 1 || report.Matched != 0 || report.Mismatched != 0 {
		t.Fatalf("report = %#v", report)
	}
	if report.Cases[0].Actual != evals.OutcomeInconclusive {
		t.Fatalf("actual = %v", report.Cases[0].Actual)
	}
}

func TestEvalVerboseJSONIncludesUnits(t *testing.T) {
	root := writeEvalProject(t, evalProject{
		source: "package sample\n\nfunc JoinInCode() {\n\tprintln(\"join\")\n}\n",
		evals: `{
			"version": 1,
			"cases": [{"rule": "database-joins", "file": "sample.go", "expect": "fail"}]
		}`,
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{
			"model": "jev-test",
			"answers": {"database-joins": {"type": "choice", "choice": "fail", "confidence": 0.92}}
		}`)
	}))
	defer server.Close()
	setEvalEnv(t, server.URL)

	run := func(args []string) []byte {
		t.Helper()
		var stdout, stderr bytes.Buffer
		exitCode := runCLI(context.Background(), args, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("Run() exit code = %d; stderr = %q", exitCode, stderr.String())
		}
		return stdout.Bytes()
	}
	base := []string{"eval", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"}

	plain := run(base)
	if bytes.Contains(plain, []byte(`"units"`)) {
		t.Fatalf("default JSON includes units: %s", plain)
	}
	verbose := run(append(append([]string(nil), base...), "--verbose"))
	if !bytes.Contains(verbose, []byte(`"units"`)) {
		t.Fatalf("verbose JSON omits units: %s", verbose)
	}
	if !bytes.Contains(verbose, []byte(`"confidence": 0.92`)) {
		t.Fatalf("verbose JSON omits confidence: %s", verbose)
	}
}

// writeExcludedExamplesProject writes a project whose only source files live
// under examples and whose rule excludes that directory.
func writeExcludedExamplesProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	config := `{
		"languages": {"go": {}},
		"rules": [{
			"id": "database-joins",
			"description": "Join related records in the database.",
			"severity": "error",
			"include": ["**/*.go"],
			"exclude": ["**/examples/**"],
			"localize": ["statement"]
		}]
	}`
	if err := os.WriteFile(filepath.Join(root, "jevlint.json"), []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	writeProjectFile(t, root, "examples/rules/database-joins/good/sample.go",
		"package good\n\nfunc Good() {}\n")
	writeProjectFile(t, root, "examples/rules/database-joins/bad/sample.go",
		"package bad\n\nfunc Bad() {}\n")
	return root
}

func TestCheckSkipsExamplesDirectory(t *testing.T) {
	root := writeExcludedExamplesProject(t)
	server := passingJevServer(t)
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"check",
			"--config", filepath.Join(root, "jevlint.json"),
			"--format", "json",
			".",
		},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	var report struct {
		ScannedFiles int `json:"scannedFiles"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.ScannedFiles != 0 {
		t.Fatalf("scannedFiles = %d, want 0 (examples are excluded)", report.ScannedFiles)
	}
}

func TestStreamPrinterWritesUnitsThenCase(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	printer := &streamPrinter{writer: &output, style: outputStyle{}}
	printer.unit(
		evals.Case{Rule: "database-joins", File: "bad/example.go"},
		evals.UnitDecision{
			Name:       "Join",
			Kind:       parsing.CodeKindFunction,
			StartLine:  12,
			EndLine:    20,
			Status:     evaluation.StatusFail,
			Confidence: 0.62,
		},
	)
	printer.caseResult(evals.Result{
		Rule:     "database-joins",
		File:     "bad/example.go",
		Expected: evals.ExpectPass,
		Actual:   evals.OutcomeInconclusive,
	})
	got := output.String()
	for _, want := range []string{
		"database-joins\n",
		"  bad/example.go\n",
		"    function Join  lines 12-20  fail  0.62\n",
		"    ? expected: pass   actual: inconclusive\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
}

func TestWriteRunSummary(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	writeRunSummary(
		&output,
		outputStyle{},
		evals.Report{Total: 2, Matched: 1, Inconclusive: 1},
		"eval cases",
	)
	if !strings.Contains(output.String(), "1/2 eval cases matched expectations, 1 inconclusive\n") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestWriteRunLegendExplainsOutcomes(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	writeRunLegend(&output, outputStyle{})
	got := output.String()
	for _, want := range []string{
		"legend",
		"matched       ",
		"mismatched    ",
		"inconclusive  ",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("legend = %q, want %q", got, want)
		}
	}
}
