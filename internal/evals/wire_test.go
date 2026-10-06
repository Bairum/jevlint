package evals

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/evaluation"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

// The fixture name must not reach the provider anywhere in the request,
// including same-file related types added by context.types.
func TestEvalWirePayloadHidesFixtureNameWithTypeContext(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data, _ := io.ReadAll(request.Body)
		var payload struct {
			Questions map[string]json.RawMessage `json:"questions"`
		}
		_ = json.Unmarshal(data, &payload)
		mu.Lock()
		bodies = append(bodies, string(data))
		mu.Unlock()
		answers := make(map[string]any, len(payload.Questions))
		for id := range payload.Questions {
			answers[id] = map[string]any{"type": "choice", "choice": "pass", "probabilities": map[string]float64{"pass": 1, "fail": 0}}
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"model": "jev-test", "answers": answers})
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := "pub struct Percent { pub value: u8 }\n" +
		"impl Percent { pub fn new(value: u8) -> Option<Percent> { (value <= 100).then_some(Percent { value }) } }\n" +
		"pub fn bypass(percent: &mut Percent) { percent.value = 200; }\n"
	if err := os.WriteFile(filepath.Join(root, "bad", "verdict-leak-bug.rs"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	evalsJSON := `{"version": 1, "cases": [{"rule": "typed", "file": "bad/verdict-leak-bug.rs", "expect": "fail"}]}`
	if err := os.WriteFile(filepath.Join(root, DefaultFile), []byte(evalsJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Languages: map[string]config.Language{"rust": {}},
		Rules: []config.Rule{{
			ID:          "typed",
			Description: "A rule that needs type context.",
			Severity:    config.SeverityWarning,
			Kinds:       []config.TargetKind{config.TargetKindFunction, config.TargetKindType},
			Context:     config.RuleContext{Types: true},
		}},
	}
	extractor, err := parsing.NewExtractor(cfg.Languages)
	if err != nil {
		t.Fatal(err)
	}
	document, err := Load(filepath.Join(root, DefaultFile), cfg, extractor)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	client, err := evaluation.NewClient(evaluation.Options{
		APIKey:     "sk-test",
		BaseURL:    evaluation.ServiceURL(server.URL),
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), document, cfg, extractor, client, Options{Root: root, Concurrency: 1}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	sawTypes := false
	for _, body := range bodies {
		if strings.Contains(body, "verdict-leak-bug") || strings.Contains(body, "bad/") {
			t.Fatalf("request leaks the fixture name: %s", body)
		}
		sawTypes = sawTypes || strings.Contains(body, `"types"`)
	}
	if !sawTypes {
		t.Fatalf("no request carried type context; the regression is not exercised: %v", bodies)
	}
}
