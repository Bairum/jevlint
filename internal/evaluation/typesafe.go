package evaluation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	defaultBaseURL = "https://api.typesafe.ai"
	defaultModel   = "jev-latest"
)

// TypeSafeProvider talks to TypeSafe's Jev over the SystemOne API.
type TypeSafeProvider struct{}

// Name identifies the provider.
func (TypeSafeProvider) Name() string { return "typesafe" }

// BaseURL is the default TypeSafe service base.
func (TypeSafeProvider) BaseURL() string { return defaultBaseURL }

// DefaultModel is the default Jev model.
func (TypeSafeProvider) DefaultModel() string { return defaultModel }

// Configure fills options from the TYPESAFE_* environment variables.
func (TypeSafeProvider) Configure(options *Options, getenv func(string) string) error {
	if options.Endpoint == "" {
		options.Endpoint = strings.TrimSpace(getenv("TYPESAFE_ENDPOINT"))
	}
	if options.BaseURL == "" {
		options.BaseURL = ServiceURL(getenv("TYPESAFE_BASE_URL"))
	}
	if options.APIKey == "" {
		options.APIKey = APIKey(strings.TrimSpace(getenv("TYPESAFE_API_KEY")))
	}
	if options.Model == "" {
		options.Model = strings.TrimSpace(getenv("TYPESAFE_DEFAULT_MODEL"))
	}
	return nil
}

// ValidateQuestions applies no extra limits.
func (TypeSafeProvider) ValidateQuestions(map[string]question) error { return nil }

// MaxBodyBytes applies no explicit body cap.
func (TypeSafeProvider) MaxBodyBytes() int { return 0 }

// Answers reads the top-level answers from a Jev response.
func (TypeSafeProvider) Answers(body []byte) (map[string]choiceAnswer, error) {
	var response systemOneResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return response.Answers, nil
}

// ResponseError reads Jev's {"error": "..."} shape.
func (TypeSafeProvider) ResponseError(
	status int,
	headers http.Header,
	body []byte,
) error {
	message, _ := parseServiceError(body)
	return serviceError(status, headers.Get("x-typesafe-request-id"), message)
}

// DescribeCredential names the credential without revealing it.
func (TypeSafeProvider) DescribeCredential(key string) string {
	if strings.HasPrefix(key, "apikey_") {
		return fmt.Sprintf("typesafe-key (len %d)", len(key))
	}
	return fmt.Sprintf("unknown (len %d)", len(key))
}

// systemOneResponse is the body returned by a Jev/SystemOne service.
type systemOneResponse struct {
	Answers map[string]choiceAnswer `json:"answers"`
}
