package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorEscapesProviderTerminalControls(t *testing.T) {
	root := t.TempDir()
	writeProjectConfig(t, root)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("x-typesafe-request-id", "request-visible\u202e\u009brequest-end")
		writer.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(writer).Encode(map[string]string{
			"error": "échec 日本語\x1b]0;forged-title\a\r\b\u2066message-end",
		})
	}))
	defer server.Close()
	t.Setenv("TYPESAFE_API_KEY", "apikey_test")
	t.Setenv("TYPESAFE_BASE_URL", server.URL)
	t.Setenv("TYPESAFE_ENDPOINT", "")
	t.Setenv("JEVLINT_PROVIDER", "typesafe")
	t.Setenv("JEVLINT_DEBUG", "")

	var stdout, stderr bytes.Buffer
	code := runCLI(context.Background(), []string{
		"doctor", "--config", filepath.Join(root, "jevlint.json"),
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("doctor exit = %d, want 2; stdout = %q stderr = %q", code, stdout.String(), stderr.String())
	}
	out := stdout.String() + stderr.String()
	for _, control := range []string{"\x1b", "\a", "\r", "\b", "\u2066", "\u202e", "\u009b"} {
		if strings.Contains(out, control) {
			t.Errorf("doctor output contains raw terminal control %q: %q", control, out)
		}
	}
	for _, want := range []string{"échec 日本語", "message-end", "request-visible", "request-end"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output lost readable diagnostic %q: %q", want, out)
		}
	}
}
