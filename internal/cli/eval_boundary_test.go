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
	"sync/atomic"
	"testing"

	"github.com/codegirl-007/jevlint/internal/evals"
	"github.com/codegirl-007/jevlint/internal/packs"
)

func TestEvalPacksConfinesFixturesToDocumentDirectory(t *testing.T) {
	for _, fixture := range []string{"../outside.go", "fixtures/sample.go"} {
		t.Run(fixture, func(t *testing.T) {
			cache := t.TempDir()
			t.Setenv("XDG_CACHE_HOME", cache)
			sha := strings.Repeat("a", 40)
			packDir, err := packs.CacheDir(cache, sha, "example/rules")
			if err != nil {
				t.Fatal(err)
			}
			writeProjectFile(t, packDir, "pack.json", `{"version":1,"id":"example/rules","languages":["go"],"evals":"evals/cases.json"}`)
			writeProjectFile(t, packDir, "rules.json", `{"rules":[{"id":"pack-rule","description":"Pack rule.","severity":"error"}]}`)
			writeProjectFile(t, packDir, "outside.go", "package sample\n\nfunc Outside() {}\n")
			writeProjectFile(t, packDir, "evals/fixtures/sample.go", "package sample\n\nfunc Ready() {}\n")
			writeProjectFile(t, packDir, "evals/cases.json", fmt.Sprintf(`{"version":1,"cases":[{"rule":"pack-rule","file":%q,"expect":"pass"}]}`, fixture))
			root := writeEvalProject(t, evalProject{
				config: fmt.Sprintf(`{
					"languages":{"go":{}},
					"packs":[{"id":"example/rules","source":"example/rules","sha":%q}],
					"rules":[{"id":"local-rule","description":"Local rule.","severity":"info"}]
				}`, sha),
				source: "package sample\n\nfunc Local() {}\n",
				evals:  `{"version":1,"cases":[{"rule":"local-rule","file":"sample.go","expect":"pass"}]}`,
			})
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				writer.Header().Set("Content-Type", "application/json")
				fmt.Fprint(writer, `{"model":"jev-test","answers":{"local-rule":{"type":"choice","choice":"pass","confidence":1},"pack-rule":{"type":"choice","choice":"pass","confidence":1}}}`)
			}))
			defer server.Close()
			setEvalEnv(t, server.URL)
			var stdout, stderr bytes.Buffer
			code := runCLI(context.Background(), []string{"eval", "--packs", "--config", filepath.Join(root, "jevlint.json"), "--format", "json"}, &stdout, &stderr)
			if fixture == "../outside.go" {
				if code != exitUsageError {
					t.Errorf("eval --packs exit = %d, want %d; stderr = %q", code, exitUsageError, stderr.String())
				}
				if calls.Load() != 0 {
					t.Fatalf("evaluator invoked %d times for an escaping pack fixture", calls.Load())
				}
				return
			}
			if code != exitSuccess {
				t.Fatalf("eval --packs exit = %d; stderr = %q", code, stderr.String())
			}
			var report evals.Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Total != 2 || report.Matched != 2 || calls.Load() != 2 {
				t.Fatalf("report = %#v; evaluator calls = %d", report, calls.Load())
			}
		})
	}
}
