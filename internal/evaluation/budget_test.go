package evaluation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

func TestBudgetDropsContextThenSkips(t *testing.T) {
	var sawCallees, sawTypes bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"model":"jev-1.13.0","answers":{"joins":{"type":"choice","choice":"pass","probabilities":{"pass":1,"fail":0}}},"usage":{"input_tokens":10,"output_tokens":2}}`))
	}))
	defer server.Close()

	unit := parsing.CodeUnit{
		Kind: parsing.CodeKindFunction, Name: "Load", Language: parsing.SourceLanguageGo,
		Path: "db.go", Source: "func Load() {}",
		Callees:      []parsing.CalleeContext{{Name: "query", Path: "q.go", Source: "func query() {}"}},
		RelatedTypes: []parsing.TypeDeclaration{{Name: "Row", Source: "type Row struct{}"}},
	}
	batch := Batch{Rules: []config.Rule{{
		ID: "joins", Description: "Join records.", Severity: config.SeverityError,
	}}, CodeUnit: unit}

	over := func(body []byte) (Estimate, error) {
		return Estimate{Total: 70000, StateAndLongestQuestion: 100}, nil
	}
	client := budgetClient(t, server.URL, over)
	_, err := client.Evaluate(context.Background(), batch)
	if !errors.Is(err, ErrOversized) {
		t.Fatalf("error = %v, want oversized", err)
	}
	meta := client.RunMeta()
	if len(meta.Oversized) != 1 || meta.Oversized[0].Path != "db.go" || meta.Oversized[0].Tokens != 70000 {
		t.Fatalf("oversized = %#v", meta.Oversized)
	}
	if meta.Usage.Requests != 0 {
		t.Fatalf("usage = %#v, want no provider call", meta.Usage)
	}

	fitWithoutCallees := func(body []byte) (Estimate, error) {
		if contains(body, `"callees"`) {
			sawCallees = true
			return Estimate{Total: 70000, StateAndLongestQuestion: 100}, nil
		}
		return Estimate{Total: 100, StateAndLongestQuestion: 50}, nil
	}
	client = budgetClient(t, server.URL, fitWithoutCallees)
	if _, err := client.Evaluate(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	meta = client.RunMeta()
	if !sawCallees || len(meta.ContextDropped) != 1 || meta.ContextDropped[0].Dropped != "callees" {
		t.Fatalf("dropped = %#v sawCallees=%v", meta.ContextDropped, sawCallees)
	}

	fitWithoutTypes := func(body []byte) (Estimate, error) {
		if contains(body, `"callees"`) || contains(body, `"types"`) {
			if contains(body, `"types"`) {
				sawTypes = true
			}
			return Estimate{Total: 70000, StateAndLongestQuestion: 100}, nil
		}
		return Estimate{Total: 100, StateAndLongestQuestion: 50}, nil
	}
	client = budgetClient(t, server.URL, fitWithoutTypes)
	if _, err := client.Evaluate(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	meta = client.RunMeta()
	if !sawTypes || len(meta.ContextDropped) != 2 {
		t.Fatalf("dropped = %#v", meta.ContextDropped)
	}
}

func TestUnknownModelSkipsBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"model":"other","answers":{"joins":{"type":"choice","probabilities":{"pass":1,"fail":0}}},"usage":{"input_tokens":3,"output_tokens":1}}`))
	}))
	defer server.Close()
	client := budgetClient(t, server.URL, func([]byte) (Estimate, error) {
		return Estimate{Total: 999999, StateAndLongestQuestion: 999999}, nil
	})
	client.model = "other-model"
	if _, err := client.Evaluate(context.Background(), Batch{
		Rules:    []config.Rule{{ID: "joins", Description: "Join records."}},
		CodeUnit: parsing.CodeUnit{Kind: parsing.CodeKindFunction, Name: "Load", Path: "db.go", Source: "func Load() {}"},
	}); err != nil {
		t.Fatal(err)
	}
	if client.RunMeta().Usage.Requests != 1 {
		t.Fatalf("usage = %#v", client.RunMeta().Usage)
	}
}

func budgetClient(t *testing.T, endpoint string, budget func([]byte) (Estimate, error)) *Client {
	t.Helper()
	client, err := NewClient(Options{
		APIKey:   "apikey_test",
		Endpoint: endpoint,
		Model:    "jev-1.13.0",
		Budget:   budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func contains(body []byte, needle string) bool {
	return strings.Contains(string(body), needle)
}
