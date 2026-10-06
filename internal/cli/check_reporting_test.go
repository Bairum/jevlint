package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/evaluation"
	"github.com/codegirl-007/jevlint/internal/parsing"
	"github.com/codegirl-007/jevlint/internal/runner"
)

func TestRunFailOnSeverity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"model":"jev-test","answers":{"database-joins":{"type":"choice","choice":"fail","probabilities":{"fail":1,"pass":0}}}}`)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		severity string
		failOn   string
		wantExit int
	}{
		{name: "default info fails info", severity: "info", wantExit: 1},
		{name: "default info fails warning", severity: "warning", wantExit: 1},
		{name: "info fails warning", severity: "warning", failOn: "info", wantExit: 1},
		{name: "warning ignores info", severity: "info", failOn: "warning", wantExit: 0},
		{name: "warning fails warning", severity: "warning", failOn: "warning", wantExit: 1},
		{name: "error ignores warning", severity: "warning", failOn: "error", wantExit: 0},
		{name: "error fails error", severity: "error", failOn: "error", wantExit: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeProjectFile(t, root, "sample.go", "package sample\n\nfunc JoinInCode() {}\n")
			writeProjectFile(t, root, "jevlint.json", fmt.Sprintf(`{
				"languages":{"go":{}},
				"rules":[{"id":"database-joins","description":"Join related records in the database.","severity":%q}]
			}`, test.severity))
			setEvalEnv(t, server.URL)
			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			args := []string{"check", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"}
			if test.failOn != "" {
				args = append(args, "--fail-on", test.failOn)
			}

			var stdout, stderr bytes.Buffer
			if code := runCLI(context.Background(), args, &stdout, &stderr); code != test.wantExit {
				t.Fatalf("Run() exit = %d, want %d; stderr = %q", code, test.wantExit, stderr.String())
			}
			report := decodeReport(t, stdout.Bytes())
			if len(report.Findings) != 1 || report.Findings[0].Severity.String() != test.severity {
				t.Fatalf("findings = %#v, want one %s finding regardless of exit threshold", report.Findings, test.severity)
			}
		})
	}
}

func TestRunRejectsInvalidFailOn(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := runCLI(context.Background(), []string{"check", "--fail-on", "fatal"}, &stdout, &stderr)
	if code != exitUsageError {
		t.Fatalf("Run() exit = %d, want %d; stderr = %q", code, exitUsageError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--fail-on") ||
		!strings.Contains(stderr.String(), "info, warning, or error") {
		t.Fatalf("stderr = %q, want valid --fail-on choices", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("invalid --fail-on wrote a report: %s", stdout.String())
	}
}

func TestRunBelowFloorJSONVisibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"model":"jev-test","answers":{"database-joins":{"type":"choice","choice":"fail","probabilities":{"fail":0.6,"pass":0.4}}}}`)
	}))
	defer server.Close()

	for _, show := range []bool{false, true} {
		t.Run(fmt.Sprintf("show=%t", show), func(t *testing.T) {
			root := t.TempDir()
			writeProjectFile(t, root, "sample.go", "package sample\n\nfunc JoinInCode() {}\n")
			writeProjectFile(t, root, "jevlint.json", `{
				"languages":{"go":{}},
				"rules":[{"id":"database-joins","description":"Join related records in the database.","severity":"error","minFailProbability":0.8}]
			}`)
			setEvalEnv(t, server.URL)
			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			args := []string{"check", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"}
			if show {
				args = append(args, "--show-below-floor")
			}

			var stdout, stderr bytes.Buffer
			if code := runCLI(context.Background(), args, &stdout, &stderr); code != exitSuccess {
				t.Fatalf("below-floor failure exit = %d, want 0; stderr = %q", code, stderr.String())
			}
			report := decodeReport(t, stdout.Bytes())
			if len(report.Findings) != 0 {
				t.Fatalf("below-floor failure entered findings: %#v", report.Findings)
			}
			if got := report.Rules["database-joins"].Decisions; got != (runner.Decisions{Fail: 1, BelowFloor: 1}) {
				t.Fatalf("decisions = %#v, want one raw below-floor failure", got)
			}
			var document map[string]json.RawMessage
			if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
				t.Fatal(err)
			}
			if _, present := document["belowFloor"]; present != show {
				t.Fatalf("belowFloor present = %t, want %t; output = %s", present, show, stdout.String())
			}
			if show {
				if len(report.BelowFloor) != 1 || report.BelowFloor[0].FailProbability != 0.6 ||
					report.BelowFloor[0].RuleID != "database-joins" {
					t.Fatalf("belowFloor = %#v, want database-joins at fail probability 0.6", report.BelowFloor)
				}
				if bytes.Count(stdout.Bytes(), []byte(`"description"`)) != 1 {
					t.Fatalf("below-floor finding repeats the rule description: %s", stdout.String())
				}
			} else if bytes.Contains(stdout.Bytes(), []byte(`"confidence"`)) {
				t.Fatalf("hidden below-floor confidence leaked: %s", stdout.String())
			}
		})
	}
}

func TestWriteReportGroupsCodeUnitsAndRuleDescriptions(t *testing.T) {
	t.Parallel()

	first := runner.Finding{
		RuleID:    "database-joins",
		Severity:  config.SeverityError,
		Status:    evaluation.StatusFail,
		Path:      "store.go",
		Kind:      parsing.CodeKindFunction,
		Name:      "First",
		StartLine: 3,
		EndLine:   5,
		Snippet:   "func First() {\n\tprintln(1)\n}",
	}
	next := first
	next.Name, next.StartLine, next.EndLine = "Next", 8, 10
	next.Snippet = "func Next() {\n\tprintln(2)\n}"
	secondRule := first
	secondRule.RuleID, secondRule.Severity = "transactions", config.SeverityWarning
	other := secondRule
	other.Path, other.Name = "other.go", "Other"
	other.Snippet = "func Other() {\n\tprintln(3)\n}"
	belowFirst, belowOther := first, other
	belowFirst.RuleID, belowOther.RuleID = "uncertain", "uncertain"
	belowOther.Severity = config.SeverityError
	belowFirst.FailProbability, belowOther.FailProbability = 0.42, 0.42
	report := runner.Report{
		ScannedFiles: 2,
		CodeUnits:    3,
		Evaluations:  25,
		Rules: map[string]runner.RuleReport{
			"database-joins": {
				Description: "Join records in the database.",
				Severity:    config.SeverityError,
				Decisions:   runner.Decisions{Pass: 2, Fail: 7, Skip: 1, Abstain: 3, Reported: 2, BelowFloor: 5},
			},
			"transactions": {
				Description: "Keep changes in a transaction.",
				Severity:    config.SeverityWarning,
				Decisions:   runner.Decisions{Pass: 1, Fail: 4, Abstain: 4, Reported: 2, BelowFloor: 2},
			},
			"uncertain": {
				Description: "Resolve uncertain database writes.",
				Severity:    config.SeverityError,
				Decisions:   runner.Decisions{Fail: 2, Abstain: 1, BelowFloor: 2},
			},
		},
		Findings: []runner.Finding{first, next, secondRule, other},
		BelowFloor: []runner.BelowFloorFinding{
			{Finding: belowFirst},
			{Finding: belowOther},
		},
	}

	var output bytes.Buffer
	if err := writeReport(&output, report, outputContext{format: formatText, color: colorNever}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, once := range []string{
		"store.go:3-5", "store.go:8-10", "other.go:3-5",
		"  3 │ func First() {", "  8 │ func Next() {", "  3 │ func Other() {",
		"Join records in the database.", "Keep changes in a transaction.", "Resolve uncertain database writes.",
	} {
		if count := strings.Count(text, once); count != 1 {
			t.Fatalf("%q count = %d, want 1; output = %s", once, count, text)
		}
	}
	for _, twice := range []string{"✗ ERROR  database-joins", "✗ WARNING  transactions", "✗ ERROR  uncertain", "fail probability 0.42"} {
		if count := strings.Count(text, twice); count != 2 {
			t.Fatalf("%q count = %d, want 2; output = %s", twice, count, text)
		}
	}
	location := strings.Index(text, "store.go:3-5")
	frame := strings.Index(text, "  3 │ func First() {")
	header := strings.Index(text, "✗ ERROR  database-joins")
	if !(location < frame && frame < header) {
		t.Fatalf("location and frame must precede rule headers: %s", text)
	}
	if strings.Count(text, "below-floor") != 3 {
		t.Fatalf("want two below-floor labels plus totals: %s", text)
	}
	for _, expected := range []string{
		"4 findings  2 errors  2 warnings",
		"2 files · 3 code units · 25 evaluations",
		"8 abstained · 9 below-floor",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("output = %s, want %q", text, expected)
		}
	}
}
