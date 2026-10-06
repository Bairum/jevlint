package evaluation

import (
	"net/http"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestNewTypeSafeRequiresAPIKey(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(Options{}); err == nil {
		t.Fatal("NewClient() error = nil")
	}
}

func TestNewTypeSafeFromEnvEndpointOverride(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"TYPESAFE_ENDPOINT": "https://clef.example/v1/systemone",
		"TYPESAFE_API_KEY":  "sk-custom",
	}
	client, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.endpoint != "https://clef.example/v1/systemone" {
		t.Errorf("endpoint = %q", client.endpoint)
	}
	if client.model != defaultModel {
		t.Errorf("model = %q, want %q", client.model, defaultModel)
	}
}

func TestNewTypeSafeFromEnvRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER": "banana",
		"TYPESAFE_API_KEY": "sk-test",
	}
	_, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err == nil || !strings.Contains(err.Error(), "JEVLINT_PROVIDER") {
		t.Fatalf("error = %v, want an unknown provider error", err)
	}
}

func TestDecodeResultsReadsTopLevelAnswers(t *testing.T) {
	t.Parallel()

	rules := []config.Rule{{ID: "database-joins"}}
	body := []byte(`{"model":"jev-test","answers":{"database-joins":{"type":"choice","choice":"pass","probabilities":{"pass":1,"fail":0}}}}`)
	answers, err := (TypeSafeProvider{}).Answers(body)
	if err != nil {
		t.Fatalf("Answers() error = %v", err)
	}
	results, err := decodeAnswers(answers.Answers, Batch{Rules: rules}, answers.Model)
	if err != nil {
		t.Fatalf("decodeAnswers() error = %v", err)
	}
	result := results["database-joins"]
	if result.Status != StatusPass || result.FailProbability != 0 || result.Model != "jev-test" {
		t.Fatalf("decodeAnswers() = %#v", results)
	}
}

func TestNewTypeSafeRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(Options{APIKey: "sk-test", Endpoint: "not-a-url"}); err == nil {
		t.Fatal("NewClient() error = nil, want an invalid endpoint error")
	}
}

func TestResponseErrorReadsJevError(t *testing.T) {
	t.Parallel()

	err := (TypeSafeProvider{}).ResponseError(http.StatusUnauthorized, http.Header{}, []byte(`{"error":"invalid key"}`))
	if !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("responseError() = %v, want the Jev error message", err)
	}
}
