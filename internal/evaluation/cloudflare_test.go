package evaluation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

func TestNewTypeSafeFromEnvUsesCloudflareClef(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_AUTH_TOKEN": "cf-token",
	}
	client, err := NewClientFromEnv(
		Options{HTTPClient: http.DefaultClient},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	want := "https://api.cloudflare.com/client/v4/accounts/e81a7093dc03d8bc6da9f0332d444b8f" +
		"/ai/run/@cf/cloudflare/clef"
	if client.endpoint != want {
		t.Errorf("endpoint = %q, want %q", client.endpoint, want)
	}
	if client.model != "clef" {
		t.Errorf("model = %q, want clef", client.model)
	}
	if client.apiKey != "cf-token" {
		t.Errorf("apiKey = %q, want cf-token", client.apiKey)
	}
}

func TestNewTypeSafeFromEnvCloudflareClefFlash(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "clef",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_API_TOKEN":  "cf-token",
		"CLEF_MODEL":            "clef-flash",
	}
	client, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if !strings.HasSuffix(client.endpoint, "/@cf/cloudflare/clef-flash") {
		t.Errorf("endpoint = %q", client.endpoint)
	}
	if client.model != "clef-flash" {
		t.Errorf("model = %q, want clef-flash", client.model)
	}
}

func TestNewTypeSafeFromEnvCloudflareRejectsConflictingTokens(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_AUTH_TOKEN": "one-token",
		"CLOUDFLARE_API_TOKEN":  "another-token",
	}
	_, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err == nil || !strings.Contains(err.Error(), "set only one") {
		t.Fatalf("error = %v, want a conflicting token error", err)
	}
}

func TestNewTypeSafeFromEnvCloudflareAcceptsMatchingTokens(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_AUTH_TOKEN": "cf-token",
		"CLOUDFLARE_API_TOKEN":  "cf-token",
	}
	client, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.apiKey != "cf-token" {
		t.Fatalf("apiKey = %q, want cf-token", client.apiKey)
	}
}

func TestNewTypeSafeFromEnvCloudflarePrefersCloudflareToken(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_AUTH_TOKEN": "cf-token",
		"TYPESAFE_API_KEY":      "stale-typesafe-key",
	}
	client, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.apiKey != "cf-token" {
		t.Fatalf("apiKey = %q, want the Cloudflare token", client.apiKey)
	}
}

func TestNewTypeSafeFromEnvCloudflareRejectsTypesafeKeyAlone(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"TYPESAFE_API_KEY":      "stale-typesafe-key",
	}
	_, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err == nil || !strings.Contains(err.Error(), "CLOUDFLARE_AUTH_TOKEN") {
		t.Fatalf("error = %v, want a missing token error", err)
	}
}

func TestNewTypeSafeFromEnvCloudflareCustomEndpointUsesEnvKey(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":  "cloudflare",
		"TYPESAFE_ENDPOINT": "https://clef.internal/v1/systemone",
		"TYPESAFE_API_KEY":  "custom-key",
		"CLEF_MODEL":        "clef-flash",
	}
	client, err := NewClientFromEnv(
		Options{},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.endpoint != "https://clef.internal/v1/systemone" {
		t.Fatalf("endpoint = %q", client.endpoint)
	}
	if client.apiKey != "custom-key" {
		t.Fatalf("apiKey = %q, want the custom key", client.apiKey)
	}
	if client.model != "clef-flash" {
		t.Fatalf("model = %q, want clef-flash", client.model)
	}
}

func TestNewTypeSafeFromEnvCloudflareRequiresCredentials(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]string{
		"missing account": {
			"JEVLINT_PROVIDER":      "cloudflare",
			"CLOUDFLARE_AUTH_TOKEN": "cf-token",
		},
		"missing token": {
			"JEVLINT_PROVIDER":      "cloudflare",
			"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		},
	}
	for name, env := range tests {
		name, env := name, env
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := NewClientFromEnv(
				Options{},
				func(key string) string { return env[key] },
			)
			if err == nil || !strings.Contains(err.Error(), "CLOUDFLARE_") {
				t.Fatalf("error = %v, want a Cloudflare credential error", err)
			}
		})
	}
}

func TestDecodeResultsUnwrapsCloudflareEnvelope(t *testing.T) {
	t.Parallel()

	rules := []config.Rule{{ID: "database-joins"}, {ID: "semicolons"}}
	body := []byte(`{
		"result": {
			"model": "clef",
			"answers": {
				"database-joins.s0": {"type": "score", "score": 2},
				"semicolons.s0": {"type": "score", "score": 0}
			},
			"usage": {"input_tokens": 10, "output_tokens": 0}
		},
		"success": true,
		"errors": [],
		"messages": []
	}`)
	answers, err := (CloudflareProvider{}).Answers(body)
	if err != nil {
		t.Fatalf("Answers() error = %v", err)
	}
	results, err := decodeAnswers(answers.Answers, Batch{Rules: rules}, answers.Model)
	if err != nil {
		t.Fatalf("decodeAnswers() error = %v", err)
	}
	if results["database-joins"].Status != StatusFail || results["database-joins"].FailProbability != 1 ||
		results["semicolons"].Status != StatusPass || results["semicolons"].FailProbability != 0 {
		t.Fatalf("decodeAnswers() = %#v", results)
	}
}

func TestResponseErrorParsesCloudflareEnvelope(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("cf-ray", "8abc123-FRA")
	body := []byte(`{"result":null,"success":false,"errors":[{"code":10000,"message":"Authentication error"}],"messages":[]}`)
	err := (CloudflareProvider{}).ResponseError(http.StatusUnauthorized, headers, body)
	message := err.Error()
	for _, want := range []string{"10000", "Authentication error", "Workers AI permission", "8abc123-FRA"} {
		if !strings.Contains(message, want) {
			t.Errorf("responseError() = %q, want it to contain %q", message, want)
		}
	}
}

func TestResponseErrorHintsRoutingCode(t *testing.T) {
	t.Parallel()

	body := []byte(`{"success":false,"errors":[{"code":7003,"message":"Could not route"}],"messages":[]}`)
	err := (CloudflareProvider{}).ResponseError(http.StatusNotFound, http.Header{}, body)
	if !strings.Contains(err.Error(), "CLOUDFLARE_ACCOUNT_ID") {
		t.Fatalf("responseError() = %v, want an account-id hint", err)
	}
}

func TestNewTypeSafeFromEnvCloudflareRejectsBadAccountID(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "not-hex",
		"CLOUDFLARE_AUTH_TOKEN": "cfut_test",
	}
	_, err := NewClientFromEnv(Options{}, func(key string) string { return env[key] })
	if err == nil || !strings.Contains(err.Error(), "32-character") {
		t.Fatalf("error = %v, want an account-id error", err)
	}
}

func TestNewTypeSafeFromEnvCloudflareRejectsBadModel(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_AUTH_TOKEN": "cfut_test",
		"CLEF_MODEL":            "llama-3",
	}
	_, err := NewClientFromEnv(Options{}, func(key string) string { return env[key] })
	if err == nil || !strings.Contains(err.Error(), "CLEF_MODEL") {
		t.Fatalf("error = %v, want a model error", err)
	}
}

func TestNewTypeSafeFromEnvCloudflareRejectsGlobalKey(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": "e81a7093dc03d8bc6da9f0332d444b8f",
		"CLOUDFLARE_AUTH_TOKEN": "cfk_" + strings.Repeat("a", 48),
	}
	_, err := NewClientFromEnv(Options{}, func(key string) string { return env[key] })
	if err == nil || !strings.Contains(err.Error(), "Global API Key") {
		t.Fatalf("error = %v, want a global-key error", err)
	}
}

func TestNewTypeSafeFromEnvCloudflareStripsQuotes(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"JEVLINT_PROVIDER":      "cloudflare",
		"CLOUDFLARE_ACCOUNT_ID": `"e81a7093dc03d8bc6da9f0332d444b8f"`,
		"CLOUDFLARE_AUTH_TOKEN": `"cfut_secret"`,
	}
	client, err := NewClientFromEnv(Options{}, func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.apiKey != "cfut_secret" {
		t.Fatalf("apiKey = %q, want the unquoted token", client.apiKey)
	}
	if !strings.Contains(client.endpoint, "e81a7093dc03d8bc6da9f0332d444b8f") {
		t.Fatalf("endpoint = %q", client.endpoint)
	}
}

func TestPlanRequestsEnforcesCloudflareLimits(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Options{
		APIKey:   "cfut_test",
		Endpoint: "https://example.test/clef",
	})
	if err != nil {
		t.Fatal(err)
	}
	client.provider = CloudflareProvider{}
	unit := parsing.CodeUnit{
		Kind:     parsing.CodeKindFunction,
		Name:     "f",
		Language: parsing.SourceLanguageGo,
		Path:     "f.go",
		Source:   "func f() {}",
	}

	tooMany := make([]config.Rule, maxCloudflareQuestions+1)
	for index := range tooMany {
		tooMany[index] = config.Rule{ID: fmt.Sprintf("r%d", index)}
	}
	bodies, _, oversized, err := client.planRequests(Batch{Rules: tooMany, CodeUnit: unit})
	if err != nil || oversized != nil || len(bodies) != 2 {
		t.Fatalf("planRequests() bodies=%d err=%v oversized=%#v", len(bodies), err, oversized)
	}
	for _, body := range bodies {
		var payload struct {
			Questions map[string]json.RawMessage `json:"questions"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Questions) == 0 || len(payload.Questions) > maxCloudflareQuestions {
			t.Fatalf("questions = %d, want 1..%d", len(payload.Questions), maxCloudflareQuestions)
		}
	}

	badID := []config.Rule{{ID: "not valid!"}}
	if _, _, _, err := client.planRequests(Batch{Rules: badID, CodeUnit: unit}); err == nil ||
		!strings.Contains(err.Error(), "cannot be sent to Cloudflare") {
		t.Fatalf("planRequests() error = %v, want a rule-id error", err)
	}

	bodies, _, oversized, err = client.planRequests(Batch{Rules: []config.Rule{{ID: "ok"}}, CodeUnit: unit})
	if err != nil || oversized != nil || len(bodies) != 1 {
		t.Fatalf("planRequests() bodies=%d err=%v oversized=%#v", len(bodies), err, oversized)
	}
}

func TestNewTypeSafeFromEnvWarnsOnCloudflareVarMismatch(t *testing.T) {
	t.Parallel()

	var warnings []string
	env := map[string]string{
		"TYPESAFE_API_KEY":      "apikey_test",
		"CLOUDFLARE_AUTH_TOKEN": "cfut_test",
	}
	client, err := NewClientFromEnv(
		Options{Warnf: func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}},
		func(key string) string { return env[key] },
	)
	if err != nil {
		t.Fatalf("NewClientFromEnv() error = %v", err)
	}
	if client.provider.Name() != "typesafe" {
		t.Fatalf("provider = %q, want the default", client.provider.Name())
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "JEVLINT_PROVIDER") {
		t.Fatalf("warnings = %#v", warnings)
	}
}
