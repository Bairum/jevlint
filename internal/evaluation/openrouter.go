package evaluation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	defaultOpenRouterBaseURL = "https://openrouter.ai/api"
	defaultOpenRouterModel   = "typesafe/jev-1.13"
	openRouterTitle          = "jevlint"
)

// OpenRouterProvider talks to Jev through OpenRouter's System One endpoint
// (`https://openrouter.ai/api/v1/systemone`). OpenRouter routes the request to
// TypeSafe and bills it to the OpenRouter account.
type OpenRouterProvider struct {
	referer string
}

// Name identifies the provider.
func (*OpenRouterProvider) Name() string { return "openrouter" }

// BaseURL is the OpenRouter API base; the client appends "/v1/systemone".
func (*OpenRouterProvider) BaseURL() string { return defaultOpenRouterBaseURL }

// DefaultModel is the pinned Jev release served by OpenRouter.
func (*OpenRouterProvider) DefaultModel() string { return defaultOpenRouterModel }

// Configure fills options from the OPENROUTER_* environment variables.
func (provider *OpenRouterProvider) Configure(options *Options, getenv func(string) string) error {
	if options.Endpoint == "" {
		options.Endpoint = strings.TrimSpace(getenv("TYPESAFE_ENDPOINT"))
	}
	if options.BaseURL == "" {
		options.BaseURL = ServiceURL(firstNonEmpty(
			cleanCredential(getenv("OPENROUTER_BASE_URL")),
			defaultOpenRouterBaseURL,
		))
	}
	if options.APIKey == "" {
		options.APIKey = APIKey(cleanCredential(getenv("OPENROUTER_API_KEY")))
	}
	if options.Model == "" {
		options.Model = firstNonEmpty(
			cleanCredential(getenv("OPENROUTER_MODEL")),
			defaultOpenRouterModel,
		)
	}
	provider.referer = strings.TrimSpace(getenv("OPENROUTER_SITE_URL"))
	if options.APIKey == "" {
		return errors.New(
			"OPENROUTER_API_KEY is required when JEVLINT_PROVIDER=openrouter",
		)
	}
	return nil
}

// ValidateQuestions applies no extra limits.
func (*OpenRouterProvider) ValidateQuestions(map[string]question) error { return nil }

// MaxBodyBytes applies no explicit body cap.
func (*OpenRouterProvider) MaxBodyBytes() int { return 0 }

// Answers reads the answers, accepting either a bare SystemOne body or one
// wrapped in a "result" envelope.
func (*OpenRouterProvider) Answers(body []byte) (map[string]choiceAnswer, error) {
	var response envelopeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if response.Answers != nil {
		return response.Answers, nil
	}
	if response.Result != nil {
		return response.Result.Answers, nil
	}
	return nil, nil
}

// ResponseError reads OpenRouter's {"error": {"message", "code"}} shape.
func (*OpenRouterProvider) ResponseError(
	status int,
	headers http.Header,
	body []byte,
) error {
	message, code := parseServiceError(body)
	if code != 0 {
		message = fmt.Sprintf("%s (code %d)", message, code)
	}
	requestID := firstNonEmpty(
		headers.Get("x-request-id"),
		headers.Get("cf-ray"),
	)
	return serviceError(status, requestID, message)
}

// DescribeCredential names the credential without revealing it.
func (*OpenRouterProvider) DescribeCredential(key string) string {
	if strings.HasPrefix(key, "sk-or-") {
		return fmt.Sprintf("openrouter-key (len %d)", len(key))
	}
	return fmt.Sprintf("unknown (len %d)", len(key))
}

// Headers adds the optional OpenRouter attribution headers.
func (provider *OpenRouterProvider) Headers() map[string]string {
	headers := map[string]string{"X-Title": openRouterTitle}
	if provider.referer != "" {
		headers["HTTP-Referer"] = provider.referer
	}
	return headers
}

// openRouterVariablesSet reports whether any OpenRouter variable is present,
// which usually means the provider was meant to be openrouter.
func openRouterVariablesSet(getenv func(string) string) bool {
	for _, name := range []string{
		"OPENROUTER_API_KEY",
		"OPENROUTER_MODEL",
		"OPENROUTER_BASE_URL",
		"OPENROUTER_SITE_URL",
	} {
		if strings.TrimSpace(getenv(name)) != "" {
			return true
		}
	}
	return false
}
