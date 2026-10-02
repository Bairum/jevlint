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
)

func TestDoctorOfflineReportsHealthy(t *testing.T) {
	root := t.TempDir()
	writeProjectConfig(t, root)
	t.Setenv("TYPESAFE_API_KEY", "apikey_test")
	t.Setenv("TYPESAFE_BASE_URL", "")
	t.Setenv("TYPESAFE_ENDPOINT", "")
	t.Setenv("JEVLINT_PROVIDER", "")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		[]string{"doctor", "--offline", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("doctor exit = %d; stdout = %q stderr = %q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"config", "credentials", "skipped (--offline)"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing %q:\n%s", want, out)
		}
	}
}

func TestDoctorReportsMissingKey(t *testing.T) {
	root := t.TempDir()
	writeProjectConfig(t, root)
	t.Setenv("TYPESAFE_API_KEY", "")
	t.Setenv("JEVLINT_PROVIDER", "")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		[]string{"doctor", "--offline", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf("doctor exit = %d, want 2; stdout = %q", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Fatalf("doctor output = %q, want a failure", stdout.String())
	}
}

func TestDoctorJSONOutput(t *testing.T) {
	root := t.TempDir()
	writeProjectConfig(t, root)
	t.Setenv("TYPESAFE_API_KEY", "apikey_test")
	t.Setenv("TYPESAFE_BASE_URL", "")
	t.Setenv("TYPESAFE_ENDPOINT", "")
	t.Setenv("JEVLINT_PROVIDER", "")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		[]string{"doctor", "--offline", "--json", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("doctor --json exit = %d; stdout = %q", code, stdout.String())
	}
	var report struct {
		OK     bool `json:"ok"`
		Checks []struct {
			Name string `json:"name"`
			OK   bool   `json:"ok"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode doctor JSON: %v\n%s", err, stdout.String())
	}
	if !report.OK || len(report.Checks) != 3 {
		t.Fatalf("report = %#v", report)
	}
	for _, check := range report.Checks {
		if !check.OK {
			t.Errorf("check %q failed", check.Name)
		}
	}
}

func TestDoctorReportsServiceFailure(t *testing.T) {
	root := t.TempDir()
	writeProjectConfig(t, root)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(writer, `{"error":"invalid key"}`)
	}))
	defer server.Close()
	t.Setenv("TYPESAFE_API_KEY", "apikey_test")
	t.Setenv("TYPESAFE_BASE_URL", server.URL)
	t.Setenv("TYPESAFE_ENDPOINT", "")
	t.Setenv("JEVLINT_PROVIDER", "")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		[]string{"doctor", "--config", filepath.Join(root, "jevlint.json")},
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf("doctor exit = %d, want 2; stdout = %q", code, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "service") || !strings.Contains(out, "FAIL") {
		t.Fatalf("doctor output = %q, want a service failure", out)
	}
}
