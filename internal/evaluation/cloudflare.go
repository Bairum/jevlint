package evaluation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	defaultCloudflareAPIURL = "https://api.cloudflare.com/client/v4"
	defaultClefModel        = "clef"
	clefFlashModel          = "clef-flash"
	maxCloudflareQuestions  = 64
	maxCloudflareBodyBytes  = 13 << 20
)

// CloudflareProvider talks to the Clef models on Cloudflare Workers AI.
type CloudflareProvider struct{}

// Name identifies the provider.
func (CloudflareProvider) Name() string { return "cloudflare" }

// BaseURL is the default Cloudflare API base.
func (CloudflareProvider) BaseURL() string { return defaultCloudflareAPIURL }

// DefaultModel is the default Clef model.
func (CloudflareProvider) DefaultModel() string { return defaultClefModel }

// Configure fills options from the CLOUDFLARE_* environment variables.
func (CloudflareProvider) Configure(options *Options, getenv func(string) string) error {
	if options.Endpoint == "" {
		options.Endpoint = cleanCredential(getenv("TYPESAFE_ENDPOINT"))
	}
	options.Model = firstNonEmpty(
		options.Model,
		cleanCredential(getenv("CLEF_MODEL")),
		cleanCredential(getenv("TYPESAFE_DEFAULT_MODEL")),
		defaultClefModel,
	)
	if !isClefModel(options.Model) {
		return fmt.Errorf(
			"unsupported CLEF_MODEL %q; want %q or %q",
			options.Model,
			defaultClefModel,
			clefFlashModel,
		)
	}
	// A caller-supplied endpoint may point at a self-hosted SystemOne
	// service, so it is the only case where a non-Cloudflare key is accepted.
	customEndpoint := options.Endpoint != ""
	if !customEndpoint {
		accountID := cleanCredential(getenv("CLOUDFLARE_ACCOUNT_ID"))
		if accountID == "" {
			return errors.New(
				"CLOUDFLARE_ACCOUNT_ID is required when " +
					"JEVLINT_PROVIDER=cloudflare",
			)
		}
		if !isCloudflareAccountID(accountID) {
			return fmt.Errorf(
				"CLOUDFLARE_ACCOUNT_ID %q is not a 32-character hex account id",
				accountID,
			)
		}
		options.Endpoint = cloudflareAIEndpoint(accountID, options.Model)
	}
	if options.APIKey == "" {
		authToken := cleanCredential(getenv("CLOUDFLARE_AUTH_TOKEN"))
		apiToken := cleanCredential(getenv("CLOUDFLARE_API_TOKEN"))
		if authToken != "" && apiToken != "" && authToken != apiToken {
			return errors.New(
				"CLOUDFLARE_AUTH_TOKEN and CLOUDFLARE_API_TOKEN " +
					"are set to different values; set only one",
			)
		}
		token := firstNonEmpty(authToken, apiToken)
		switch {
		case strings.HasPrefix(token, "cfk_"):
			return errors.New(
				"CLOUDFLARE token looks like a Global API Key (cfk_); " +
					"create a Workers AI API token instead",
			)
		case token != "":
			options.APIKey = APIKey(token)
		case customEndpoint:
			options.APIKey = APIKey(cleanCredential(getenv("TYPESAFE_API_KEY")))
		default:
			return errors.New(
				"CLOUDFLARE_AUTH_TOKEN is required when " +
					"JEVLINT_PROVIDER=cloudflare",
			)
		}
	}
	return nil
}

// ValidateQuestions checks a batch against Cloudflare's limits.
func (CloudflareProvider) ValidateQuestions(questions map[string]question) error {
	if len(questions) > maxCloudflareQuestions {
		return fmt.Errorf(
			"Cloudflare accepts at most %d questions per request, got %d",
			maxCloudflareQuestions,
			len(questions),
		)
	}
	ids := make([]string, 0, len(questions))
	for id := range questions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !cloudflareQuestionID.MatchString(id) {
			return fmt.Errorf(
				"rule id %q cannot be sent to Cloudflare; ids allow "+
					"letters, digits, '_', '.', and '-', up to 100 characters",
				id,
			)
		}
	}
	return nil
}

// MaxBodyBytes caps the request body at Cloudflare's limit.
func (CloudflareProvider) MaxBodyBytes() int { return maxCloudflareBodyBytes }

// Answers reads the answers from Cloudflare's {"result": {...}} envelope.
func (CloudflareProvider) Answers(body []byte) (map[string]choiceAnswer, error) {
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

// ResponseError reads Cloudflare's errors[] shape and adds an actionable hint.
func (CloudflareProvider) ResponseError(
	status int,
	headers http.Header,
	body []byte,
) error {
	message, code := parseServiceError(body)
	switch {
	case code != 0 && errorHint(code) != "":
		message = fmt.Sprintf("%s (code %d: %s)", message, code, errorHint(code))
	case code != 0:
		message = fmt.Sprintf("%s (code %d)", message, code)
	}
	return serviceError(status, headers.Get("cf-ray"), message)
}

// DescribeCredential names the credential without revealing it.
func (CloudflareProvider) DescribeCredential(key string) string {
	switch {
	case strings.HasPrefix(key, "cfut_"):
		return fmt.Sprintf("cloudflare user token (len %d)", len(key))
	case strings.HasPrefix(key, "cfat_"):
		return fmt.Sprintf("cloudflare account token (len %d)", len(key))
	case strings.HasPrefix(key, "cfk_"):
		return fmt.Sprintf("cloudflare global key (len %d)", len(key))
	default:
		return fmt.Sprintf("unknown (len %d)", len(key))
	}
}

// Headers returns no extra request headers.
func (CloudflareProvider) Headers() map[string]string { return nil }

// cloudflareQuestionID matches the question ids Cloudflare accepts.
var cloudflareQuestionID = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)

// cloudflareAIEndpoint builds the Workers AI URL for one Clef model.
func cloudflareAIEndpoint(accountID string, model string) string {
	return defaultCloudflareAPIURL + "/accounts/" + url.PathEscape(accountID) +
		"/ai/run/@cf/cloudflare/" + url.PathEscape(model)
}

// cloudflareVariablesSet reports whether any Cloudflare credential variable is
// present, which usually means the provider was meant to be cloudflare.
func cloudflareVariablesSet(getenv func(string) string) bool {
	for _, name := range []string{
		"CLOUDFLARE_ACCOUNT_ID",
		"CLOUDFLARE_AUTH_TOKEN",
		"CLOUDFLARE_API_TOKEN",
	} {
		if strings.TrimSpace(getenv(name)) != "" {
			return true
		}
	}
	return false
}

// isClefModel reports whether a model is one Cloudflare Workers AI serves.
func isClefModel(model string) bool {
	return model == defaultClefModel || model == clefFlashModel
}

// isCloudflareAccountID reports whether value is a 32-character hex id.
func isCloudflareAccountID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		case char >= 'A' && char <= 'F':
		default:
			return false
		}
	}
	return true
}

// errorHint turns a Cloudflare error code into an actionable hint.
func errorHint(code int) string {
	switch code {
	case 10000:
		return "check the Cloudflare API token and its Workers AI permission"
	case 7003:
		return "check CLOUDFLARE_ACCOUNT_ID"
	case 6003, 6111:
		return "the Authorization header was malformed"
	default:
		return ""
	}
}

// envelopeResponse handles hosts such as Cloudflare Workers AI that wrap the
// SystemOne body in a "result" field.
type envelopeResponse struct {
	Answers map[string]choiceAnswer `json:"answers"`
	Result  *systemOneResponse      `json:"result"`
}
