package evaluation

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func openRouterEnv(extra map[string]string) func(string) string {
	env := map[string]string{
		"JEVLINT_PROVIDER":   "openrouter",
		"OPENROUTER_API_KEY": "sk-or-test",
	}
	for key, value := range extra {
		env[key] = value
	}
	return func(key string) string { return env[key] }
}

func TestOpenRouterProviderDefaults(t *testing.T) {
	t.Parallel()

	client, err := NewClientFromEnv(Options{}, openRouterEnv(nil))
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.endpoint != "https://openrouter.ai/api/v1/systemone" {
		t.Errorf("endpoint = %q", client.endpoint)
	}
	if client.model != "typesafe/jev-1.13" {
		t.Errorf("model = %q", client.model)
	}
	if kind := client.CredentialKind(); !strings.HasPrefix(kind, "openrouter-key") {
		t.Errorf("credential kind = %q", kind)
	}
}

func TestOpenRouterProviderOverrides(t *testing.T) {
	t.Parallel()

	client, err := NewClientFromEnv(Options{}, openRouterEnv(map[string]string{
		"OPENROUTER_MODEL":    "~typesafe/jev-latest",
		"OPENROUTER_BASE_URL": "https://example.test/api",
	}))
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.endpoint != "https://example.test/api/v1/systemone" {
		t.Errorf("endpoint = %q", client.endpoint)
	}
	if client.model != "~typesafe/jev-latest" {
		t.Errorf("model = %q", client.model)
	}
}

func TestOpenRouterProviderRequiresKey(t *testing.T) {
	t.Parallel()

	_, err := NewClientFromEnv(Options{}, func(key string) string {
		if key == "JEVLINT_PROVIDER" {
			return "openrouter"
		}
		return ""
	})
	if err == nil || !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
		t.Fatalf("error = %v, want a missing-key error", err)
	}
}

func TestOpenRouterProviderAnswers(t *testing.T) {
	t.Parallel()

	provider := &OpenRouterProvider{}
	topLevel := []byte(`{"answers":{"r":{"type":"score","score":0}}}`)
	answers, err := provider.Answers(topLevel)
	if err != nil || answers.Answers["r"].Score == nil || *answers.Answers["r"].Score != 0 {
		t.Fatalf("top-level Answers() = %#v, %v", answers, err)
	}

	wrapped := []byte(`{"result":{"answers":{"r":{"type":"score","score":2}}}}`)
	answers, err = provider.Answers(wrapped)
	if err != nil || answers.Answers["r"].Score == nil || *answers.Answers["r"].Score != 2 {
		t.Fatalf("wrapped Answers() = %#v, %v", answers, err)
	}
}

func TestOpenRouterProviderResponseError(t *testing.T) {
	t.Parallel()

	err := (&OpenRouterProvider{}).ResponseError(
		402,
		http.Header{},
		[]byte(`{"error":{"message":"Insufficient credits","code":402}}`),
	)
	message := err.Error()
	if !strings.Contains(message, "Insufficient credits") ||
		!strings.Contains(message, "code 402") {
		t.Fatalf("ResponseError() = %q", message)
	}
}

func TestParseServiceErrorAcceptsObjectError(t *testing.T) {
	t.Parallel()

	message, code := parseServiceError([]byte(`{"error":{"message":"nope","code":429}}`))
	if message != "nope" || code != 429 {
		t.Fatalf("parseServiceError() = %q, %d", message, code)
	}

	message, code = parseServiceError([]byte(`{"error":"plain"}`))
	if message != "plain" || code != 0 {
		t.Fatalf("parseServiceError() = %q, %d", message, code)
	}
}

func TestOpenRouterProviderHeaders(t *testing.T) {
	t.Parallel()

	provider := &OpenRouterProvider{}
	if err := provider.Configure(&Options{APIKey: "sk-or-test"}, func(key string) string {
		if key == "OPENROUTER_SITE_URL" {
			return "https://example.com"
		}
		return ""
	}); err != nil {
		t.Fatal(err)
	}
	headers := provider.Headers()
	if headers["X-Title"] != "jevlint" {
		t.Errorf("X-Title = %q", headers["X-Title"])
	}
	if headers["HTTP-Referer"] != "https://example.com" {
		t.Errorf("HTTP-Referer = %q", headers["HTTP-Referer"])
	}
}

func TestOpenRouterEvaluateSendsAttributionHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/systemone" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer sk-or-test" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("X-Title"); got != "jevlint" {
			t.Errorf("X-Title = %q", got)
		}
		if got := request.Header.Get("HTTP-Referer"); got != "https://example.com" {
			t.Errorf("HTTP-Referer = %q", got)
		}
		fmt.Fprint(writer, `{
			"answers": {
				"database-joins.s0": {"type": "score", "score": 0},
				"semicolons.s0": {"type": "score", "score": 0}
			}
		}`)
	}))
	defer server.Close()

	client, err := NewClientFromEnv(Options{
		Endpoint:   server.URL + "/v1/systemone",
		HTTPClient: server.Client(),
	}, openRouterEnv(map[string]string{"OPENROUTER_SITE_URL": "https://example.com"}))
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	results, err := client.Evaluate(context.Background(), testBatch())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if results["database-joins"].Status != StatusPass {
		t.Fatalf("results = %#v", results)
	}
}
