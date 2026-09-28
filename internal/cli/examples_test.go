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
	"testing"

	"jevlint/internal/evals"
)

func writeExamplesProject(t *testing.T) string {
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

func TestExamplesCommandScoresFixtures(t *testing.T) {
	root := writeExamplesProject(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload struct {
			State struct {
				Source string `json:"source"`
			} `json:"state"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		choice := "pass"
		if strings.Contains(payload.State.Source, "Bad") {
			choice = "fail"
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(
			writer,
			`{"answers":{"database-joins":{"type":"choice","choice":%q,"confidence":1}}}`,
			choice,
		)
	}))
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"examples",
			"--config", filepath.Join(root, "jevlint.json"),
			"--format", "json",
		},
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("Run() exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	var report evals.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output: %v\noutput: %s", err, stdout.String())
	}
	if report.Total != 2 || report.Matched != 2 || report.Mismatched != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestExamplesCommandReportsMismatch(t *testing.T) {
	root := writeExamplesProject(t)
	server := passingJevServer(t)
	defer server.Close()
	setEvalEnv(t, server.URL)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{
			"examples",
			"--config", filepath.Join(root, "jevlint.json"),
			"--format", "json",
		},
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
	if report.Mismatched != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestExamplesCommandRejectsMissingDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeProjectConfig(t, root)

	var stdout, stderr bytes.Buffer
	exitCode := runCLI(
		context.Background(),
		[]string{"examples", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf("Run() exit code = %d, want 2", exitCode)
	}
}

func TestCheckSkipsExamplesDirectory(t *testing.T) {
	root := writeExamplesProject(t)
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
