package evaluation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

const (
	defaultBaseURL          = "https://api.typesafe.ai"
	defaultModel            = "jev-latest"
	defaultCloudflareAPIURL = "https://api.cloudflare.com/client/v4"
	defaultClefModel        = "clef"
	defaultTimeout          = 10 * time.Second
	defaultMaxRetries       = 2
	maxResponseBytes        = 1 << 20
	maxDebugBodyBytes       = 16 << 10
	cacheKeyVersion         = "typesafe-evaluation-v1"
	answerTypeChoice        = "choice"
	httpSuccessMin          = 200
	httpSuccessLimit        = 300
	retryBackoffBase        = 500 * time.Millisecond
	retryBackoffCap         = 5 * time.Second
	retryAfterHeaderMax     = time.Minute
	criterionPass           = "The code complies with the rule, or an explicit exception applies."
	criterionFail           = "The code violates the rule, and no explicit exception applies."
	criterionSkip           = "The rule's subject is not present in this unit; the rule does not apply."
	criterionAbstain        = "The rule is relevant, but the supplied code does not contain enough context to decide pass or fail."
	minimumBatchRules       = 1
)

// APIKey is the credential sent to the service.
type APIKey string

// ServiceURL is the base address of the service.
type ServiceURL string

// TypeSafeOptions holds the settings for a client.
type TypeSafeOptions struct {
	APIKey APIKey
	// Endpoint is the full request URL. When empty, BaseURL plus
	// "/v1/systemone" is used.
	Endpoint   string
	BaseURL    ServiceURL
	HTTPClient *http.Client
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
	Cache      ResultCache
	Refresh    bool
	// Logf, when set, receives request and response debug lines. The
	// Authorization header is never logged.
	Logf func(format string, args ...any)
}

// TypeSafe is a client for a Jev/SystemOne compatible decision service.
type TypeSafe struct {
	apiKey      string
	endpoint    string
	model       string
	httpClient  *http.Client
	maxRetries  int
	sleep       func(context.Context, time.Duration) error
	cache       ResultCache
	refresh     bool
	logf        func(string, ...any)
	inflightMu  sync.Mutex
	inflight    map[string]*evaluationCall
	cacheHits   atomic.Uint64
	cacheMisses atomic.Uint64
	cacheWrites atomic.Uint64
}

// systemOneRequest is the body sent to the service.
type systemOneRequest struct {
	Model     string              `json:"model"`
	State     any                 `json:"state"`
	Questions map[string]question `json:"questions"`
}

// questionType names the kind of question sent to the service.
type questionType int

const (
	questionTypeUnknown questionType = iota
	questionTypeChoice
)

// questionCriteria holds the wording for each possible answer.
type questionCriteria struct {
	Pass    string `json:"pass"`
	Fail    string `json:"fail"`
	Skip    string `json:"skip,omitempty"`
	Abstain string `json:"abstain,omitempty"`
}

// question is one rule sent to the service.
type question struct {
	Type         questionType     `json:"type"`
	Instructions string           `json:"instructions"`
	Criteria     questionCriteria `json:"criteria"`
}

// systemOneResponse is the body returned by the service.
type systemOneResponse struct {
	Answers map[string]choiceAnswer `json:"answers"`
}

// envelopeResponse handles hosts such as Cloudflare Workers AI that wrap the
// SystemOne body in a "result" field.
type envelopeResponse struct {
	Answers map[string]choiceAnswer `json:"answers"`
	Result  *systemOneResponse      `json:"result"`
}

// choiceAnswer is the service answer for one rule.
type choiceAnswer struct {
	Type       questionType `json:"type"`
	Choice     string       `json:"choice"`
	Confidence *float64     `json:"confidence"`
}

// evaluationCall tracks one running request that callers share.
type evaluationCall struct {
	done    chan struct{}
	results map[string]Result
	err     error
}

// requestState is the code and context sent about one piece of code.
type requestState struct {
	Kind         parsing.CodeKind       `json:"kind"`
	Name         string                 `json:"name"`
	Language     parsing.SourceLanguage `json:"language"`
	Path         string                 `json:"path"`
	Source       string                 `json:"source"`
	ParentSource string                 `json:"parentSource,omitempty"`
	RegionKind   parsing.NodeKind       `json:"regionKind,omitempty"`
	RelatedTypes []requestType          `json:"types,omitempty"`
	Callees      []requestCallee        `json:"callees,omitempty"`
}

// requestType is a related type sent as context.
type requestType struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// requestCallee is a called function sent as context.
type requestCallee struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

// MarshalJSON writes the question kind as its name.
func (kind questionType) MarshalJSON() ([]byte, error) {
	switch kind {
	case questionTypeChoice:
		return json.Marshal(answerTypeChoice)
	default:
		return nil, fmt.Errorf("unsupported question type")
	}
}

// UnmarshalJSON reads a question kind from its name.
func (kind *questionType) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != answerTypeChoice {
		return fmt.Errorf("unsupported question type %q", value)
	}
	*kind = questionTypeChoice
	return nil
}

// NewTypeSafeFromEnvWithOptions reads the environment and builds a client.
//
// Two providers are supported. The default talks to a Jev/SystemOne
// compatible service (TypeSafe by default) using TYPESAFE_API_KEY,
// TYPESAFE_BASE_URL, and TYPESAFE_DEFAULT_MODEL. Set
// JEVLINT_PROVIDER=cloudflare to talk to the Clef models hosted on Cloudflare
// Workers AI using CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_AUTH_TOKEN (or
// CLOUDFLARE_API_TOKEN), with CLEF_MODEL selecting "clef" or "clef-flash".
// TYPESAFE_ENDPOINT overrides the request URL for any provider.
func NewTypeSafeFromEnvWithOptions(
	options TypeSafeOptions,
	getenv func(string) string,
) (*TypeSafe, error) {
	model := ""
	if getenv != nil {
		if options.Endpoint == "" {
			options.Endpoint = strings.TrimSpace(getenv("TYPESAFE_ENDPOINT"))
		}
		if options.BaseURL == "" {
			options.BaseURL = ServiceURL(getenv("TYPESAFE_BASE_URL"))
		}
		model = strings.TrimSpace(getenv("TYPESAFE_DEFAULT_MODEL"))
		typesafeKey := APIKey(strings.TrimSpace(getenv("TYPESAFE_API_KEY")))

		provider := strings.ToLower(strings.TrimSpace(getenv("JEVLINT_PROVIDER")))
		switch provider {
		case "", "typesafe", "jev":
			if options.APIKey == "" {
				options.APIKey = typesafeKey
			}
		case "cloudflare", "clef":
			clefModel := firstNonEmpty(
				strings.TrimSpace(getenv("CLEF_MODEL")),
				model,
				defaultClefModel,
			)
			// A caller-supplied endpoint may point at a self-hosted
			// SystemOne service, so it is the only case where a non
			// Cloudflare key is accepted.
			customEndpoint := options.Endpoint != ""
			if !customEndpoint {
				accountID := strings.TrimSpace(getenv("CLOUDFLARE_ACCOUNT_ID"))
				if accountID == "" {
					return nil, errors.New(
						"CLOUDFLARE_ACCOUNT_ID is required when " +
							"JEVLINT_PROVIDER=cloudflare",
					)
				}
				options.Endpoint = cloudflareAIEndpoint(accountID, clefModel)
			}
			if options.APIKey == "" {
				authToken := strings.TrimSpace(getenv("CLOUDFLARE_AUTH_TOKEN"))
				apiToken := strings.TrimSpace(getenv("CLOUDFLARE_API_TOKEN"))
				if authToken != "" && apiToken != "" && authToken != apiToken {
					return nil, errors.New(
						"CLOUDFLARE_AUTH_TOKEN and CLOUDFLARE_API_TOKEN " +
							"are set to different values; set only one",
					)
				}
				token := firstNonEmpty(authToken, apiToken)
				switch {
				case token != "":
					options.APIKey = APIKey(token)
				case customEndpoint:
					options.APIKey = typesafeKey
				default:
					return nil, errors.New(
						"CLOUDFLARE_AUTH_TOKEN is required when " +
							"JEVLINT_PROVIDER=cloudflare",
					)
				}
			}
			model = clefModel
		default:
			return nil, fmt.Errorf("unknown JEVLINT_PROVIDER %q", provider)
		}
	}
	client, err := NewTypeSafe(options)
	if err != nil {
		return nil, err
	}
	if model != "" {
		client.model = model
	}
	if client.logf != nil {
		client.debugf(
			"jevlint: using endpoint=%s model=%s credential=%s",
			client.endpoint,
			client.model,
			describeCredential(client.apiKey),
		)
	}
	return client, nil
}

// describeCredential names the kind of credential without revealing it.
func describeCredential(key string) string {
	kind := "unknown"
	switch {
	case strings.HasPrefix(key, "apikey_"):
		kind = "typesafe-key"
	case strings.HasPrefix(key, "cfut_"):
		kind = "cloudflare user token"
	case strings.HasPrefix(key, "cfat_"):
		kind = "cloudflare account token"
	case strings.HasPrefix(key, "cfk_"):
		kind = "cloudflare global key"
	}
	return fmt.Sprintf("%s (len %d)", kind, len(key))
}

// Endpoint returns the resolved request URL.
func (client *TypeSafe) Endpoint() string {
	return client.endpoint
}

// Model returns the resolved model name.
func (client *TypeSafe) Model() string {
	return client.model
}

// CredentialKind describes the credential without revealing it.
func (client *TypeSafe) CredentialKind() string {
	return describeCredential(client.apiKey)
}

// Ping verifies that the service answers a trivial request. It always makes a
// live call, ignoring any cache.
func (client *TypeSafe) Ping(ctx context.Context) error {
	batch := Batch{
		Rules: []config.Rule{{
			ID:          "jevlint-doctor",
			Description: "A connectivity check. This rule always passes.",
			Severity:    config.SeverityInfo,
		}},
		CodeUnit: parsing.CodeUnit{
			Kind:     parsing.CodeKindFunction,
			Name:     "jevlintDoctor",
			Language: parsing.SourceLanguageGo,
			Path:     "jevlint-doctor.go",
			Source:   "func jevlintDoctor() {}",
		},
	}
	saved := client.cache
	client.cache = nil
	_, err := client.Evaluate(ctx, batch)
	client.cache = saved
	return err
}

// cloudflareAIEndpoint builds the Workers AI URL for one Clef model.
func cloudflareAIEndpoint(accountID string, model string) string {
	return defaultCloudflareAPIURL + "/accounts/" + url.PathEscape(accountID) +
		"/ai/run/@cf/cloudflare/" + url.PathEscape(model)
}

// firstNonEmpty returns the first non-empty value, or "".
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// debugf writes one debug line when request logging is enabled.
func (client *TypeSafe) debugf(format string, args ...any) {
	if client.logf != nil {
		client.logf(format, args...)
	}
}

// debugBody renders a body for logging, capped so a large code state stays
// readable while still showing where it was cut off.
func debugBody(body []byte) string {
	if len(body) <= maxDebugBodyBytes {
		return string(body)
	}
	return string(body[:maxDebugBodyBytes]) +
		fmt.Sprintf("\n... (truncated %d bytes)", len(body)-maxDebugBodyBytes)
}

// NewTypeSafe builds a client from the given settings.
func NewTypeSafe(options TypeSafeOptions) (*TypeSafe, error) {
	apiKey := strings.TrimSpace(string(options.APIKey))
	if apiKey == "" {
		return nil, errors.New("an API key is required: set TYPESAFE_API_KEY")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(string(options.BaseURL)), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid TypeSafe base URL %q", baseURL)
	}

	endpoint := strings.TrimSpace(options.Endpoint)
	if endpoint != "" {
		parsedEndpoint, endpointErr := url.Parse(endpoint)
		if endpointErr != nil || parsedEndpoint.Scheme == "" ||
			parsedEndpoint.Host == "" {
			return nil, fmt.Errorf("invalid endpoint %q", endpoint)
		}
	} else {
		endpoint = baseURL + "/v1/systemone"
	}

	model := defaultModel

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	maxRetries := options.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}
	if maxRetries < 0 {
		return nil, errors.New("maximum retries cannot be negative")
	}

	sleep := options.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	return &TypeSafe{
		apiKey:     apiKey,
		endpoint:   endpoint,
		model:      model,
		httpClient: httpClient,
		maxRetries: maxRetries,
		sleep:      sleep,
		cache:      options.Cache,
		refresh:    options.Refresh,
		logf:       options.Logf,
		inflight:   make(map[string]*evaluationCall),
	}, nil
}

// Evaluate answers the rules in a batch for one piece of code.
func (client *TypeSafe) Evaluate(ctx context.Context, batch Batch) (map[string]Result, error) {
	body, err := client.requestBody(batch)
	if err != nil {
		return nil, err
	}
	if client.cache == nil {
		responseBody, err := client.perform(ctx, body)
		if err != nil {
			return nil, err
		}
		return decodeResults(responseBody, batch.Rules)
	}

	key := client.cacheKey(body)
	if results, ok := client.hitCache(key, batch.Rules); ok {
		return results, nil
	}
	return client.evaluateOnce(ctx, key, func() (map[string]Result, error) {
		if results, ok := client.hitCache(key, batch.Rules); ok {
			return results, nil
		}
		client.cacheMisses.Add(1)
		responseBody, err := client.perform(ctx, body)
		if err != nil {
			return nil, err
		}
		results, err := decodeResults(responseBody, batch.Rules)
		if err != nil {
			return nil, err
		}
		if client.cache.Put(key, results) {
			client.cacheWrites.Add(1)
		}
		return results, nil
	})
}

// hitCache returns cached results for a request when they are usable.
func (client *TypeSafe) hitCache(key string, rules []config.Rule) (map[string]Result, bool) {
	if client.refresh {
		return nil, false
	}
	results, ok := client.cachedResults(key, rules)
	if !ok {
		return nil, false
	}
	client.cacheHits.Add(1)
	return results, true
}

// cachedResults reads and checks the cached results for a request.
func (client *TypeSafe) cachedResults(
	key string,
	rules []config.Rule,
) (map[string]Result, bool) {
	results, ok := client.cache.Get(key)
	if !ok || len(results) != len(rules) {
		return nil, false
	}
	for _, rule := range rules {
		result, exists := results[rule.ID]
		if !exists || result.Validate() != nil {
			return nil, false
		}
	}
	return results, true
}

// evaluateOnce runs a request once and shares the answer with waiting callers.
func (client *TypeSafe) evaluateOnce(
	ctx context.Context,
	key string,
	evaluate func() (map[string]Result, error),
) (map[string]Result, error) {
	client.inflightMu.Lock()
	if call, ok := client.inflight[key]; ok {
		client.inflightMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
			return cloneResults(call.results), call.err
		}
	}
	call := &evaluationCall{done: make(chan struct{})}
	client.inflight[key] = call
	client.inflightMu.Unlock()

	call.results, call.err = evaluate()

	client.inflightMu.Lock()
	delete(client.inflight, key)
	close(call.done)
	client.inflightMu.Unlock()
	return cloneResults(call.results), call.err
}

// CacheStats returns the cache counts for this client.
func (client *TypeSafe) CacheStats() CacheStats {
	return CacheStats{
		Hits:   client.cacheHits.Load(),
		Misses: client.cacheMisses.Load(),
		Writes: client.cacheWrites.Load(),
	}
}

// requestBody builds the body sent to the service.
func (client *TypeSafe) requestBody(batch Batch) ([]byte, error) {
	questions, err := questionsForBatch(batch)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(systemOneRequest{
		Model:     client.model,
		State:     requestStateFrom(batch.CodeUnit),
		Questions: questions,
	})
	if err != nil {
		return nil, fmt.Errorf("encode TypeSafe request: %w", err)
	}
	return body, nil
}

// requestStateFrom builds the sent state from a piece of code.
func requestStateFrom(unit parsing.CodeUnit) requestState {
	state := requestState{
		Kind:         unit.Kind,
		Name:         unit.Name,
		Language:     unit.Language,
		Path:         unit.Path,
		Source:       unit.Source,
		ParentSource: unit.ParentSource,
		RegionKind:   unit.RegionKind,
	}
	if len(unit.RelatedTypes) > 0 {
		state.RelatedTypes = make([]requestType, 0, len(unit.RelatedTypes))
		for _, declaration := range unit.RelatedTypes {
			state.RelatedTypes = append(state.RelatedTypes, requestType{
				Name:   declaration.Name,
				Source: declaration.Source,
			})
		}
	}
	if len(unit.Callees) > 0 {
		state.Callees = make([]requestCallee, 0, len(unit.Callees))
		for _, callee := range unit.Callees {
			state.Callees = append(state.Callees, requestCallee{
				Name:   callee.Name,
				Path:   callee.Path,
				Source: callee.Source,
			})
		}
	}
	return state
}

// questionsForBatch builds the questions for a batch.
func questionsForBatch(batch Batch) (map[string]question, error) {
	if len(batch.Rules) < minimumBatchRules {
		return nil, errors.New("at least one rule is required")
	}
	questions := make(map[string]question, len(batch.Rules))
	for _, rule := range batch.Rules {
		if _, exists := questions[rule.ID]; exists {
			return nil, fmt.Errorf("duplicate rule id %q in evaluation batch", rule.ID)
		}
		questions[rule.ID] = question{
			Type:         questionTypeChoice,
			Instructions: instructionsFor(rule, batch.CodeUnit),
			Criteria:     criteriaFor(rule),
		}
	}
	return questions, nil
}

// criteriaFor builds the answer wording for a rule.
func criteriaFor(rule config.Rule) questionCriteria {
	criteria := questionCriteria{
		Pass: criterionPass,
		Fail: criterionFail,
	}
	if rule.AllowSkip {
		criteria.Skip = criterionSkip
	}
	if rule.AllowAbstain {
		criteria.Abstain = criterionAbstain
	}
	return criteria
}

// decodeResults reads and checks the answers from the service.
func decodeResults(
	responseBody []byte,
	rules []config.Rule,
) (map[string]Result, error) {
	var response envelopeResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode TypeSafe response: %w", err)
	}
	answers := response.Answers
	if answers == nil && response.Result != nil {
		// Cloudflare Workers AI returns {"result":{"answers":...}}.
		answers = response.Result.Answers
	}
	if answers == nil {
		return nil, errors.New("decode TypeSafe response: answers are missing")
	}

	results := make(map[string]Result, len(rules))
	for _, rule := range rules {
		answer, ok := answers[rule.ID]
		if !ok {
			return nil, fmt.Errorf("decode TypeSafe response: answer for rule %q is missing", rule.ID)
		}
		if answer.Type != questionTypeChoice {
			return nil, fmt.Errorf(
				"decode TypeSafe response: answer for rule %q has unsupported type",
				rule.ID,
			)
		}
		if answer.Confidence == nil {
			return nil, fmt.Errorf(
				"decode TypeSafe response: confidence for rule %q is missing",
				rule.ID,
			)
		}

		status, err := ParseStatus(answer.Choice)
		if err != nil {
			return nil, fmt.Errorf("decode TypeSafe response: rule %q: %w", rule.ID, err)
		}
		if status == StatusSkip && !rule.AllowSkip {
			return nil, fmt.Errorf(
				"decode TypeSafe response: rule %q returned skip without allowSkip",
				rule.ID,
			)
		}
		if status == StatusAbstain && !rule.AllowAbstain {
			return nil, fmt.Errorf(
				"decode TypeSafe response: rule %q returned abstain without allowAbstain",
				rule.ID,
			)
		}
		result := Result{
			Status:     status,
			Confidence: *answer.Confidence,
		}
		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("decode TypeSafe response: rule %q: %w", rule.ID, err)
		}
		results[rule.ID] = result
	}
	return results, nil
}

// cacheKey returns the key that identifies a request.
func (client *TypeSafe) cacheKey(body []byte) string {
	credential := sha256.Sum256([]byte(client.apiKey))
	hash := sha256.New()
	for _, part := range [][]byte{
		[]byte(cacheKeyVersion),
		[]byte(client.endpoint),
		[]byte(client.model),
		[]byte(fmt.Sprintf("%x", credential)),
		body,
	} {
		hash.Write(part)
		hash.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// perform sends a request and retries when the service asks for it.
func (client *TypeSafe) perform(ctx context.Context, body []byte) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		request, err := client.newRequest(ctx, body, attempt)
		if err != nil {
			return nil, err
		}
		response, err := client.httpClient.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if attempt >= client.maxRetries {
				return nil, fmt.Errorf("TypeSafe request failed: %w", err)
			}
			if sleepErr := client.sleep(ctx, retryDelay(attempt, nil)); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}
		responseBody, err := readResponse(response)
		if err != nil {
			return nil, err
		}
		client.debugf("jevlint: response %s: %s", response.Status, debugBody(responseBody))
		if response.StatusCode >= httpSuccessMin && response.StatusCode < httpSuccessLimit {
			return responseBody, nil
		}
		if attempt >= client.maxRetries || !retryableStatus(response.StatusCode) {
			return nil, responseError(response.StatusCode, response.Header, responseBody)
		}
		if sleepErr := client.sleep(ctx, retryDelay(attempt, response.Header)); sleepErr != nil {
			return nil, sleepErr
		}
	}
}

// newRequest builds one request to the service.
func (client *TypeSafe) newRequest(
	ctx context.Context,
	body []byte,
	attempt int,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create TypeSafe request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "github.com/codegirl-007/jevlint/0.1.0")
	request.Header.Set("X-TypeSafe-SDK", "github.com/codegirl-007/jevlint/0.1.0")
	request.Header.Set("X-TypeSafe-Runtime", runtime.Version())
	if attempt > 0 {
		request.Header.Set("X-TypeSafe-Retry-Count", strconv.Itoa(attempt))
	}
	if client.logf != nil {
		client.debugf("jevlint: request POST %s", client.endpoint)
		client.debugf(
			"jevlint: request headers: Content-Type=%s Accept=%s Authorization=Bearer <redacted>",
			request.Header.Get("Content-Type"),
			request.Header.Get("Accept"),
		)
		client.debugf("jevlint: request body: %s", debugBody(body))
	}
	return request, nil
}

// readResponse reads and closes a response body.
func readResponse(response *http.Response) ([]byte, error) {
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read TypeSafe response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close TypeSafe response: %w", closeErr)
	}
	return body, nil
}

// instructionsFor builds the question text for a rule.
func instructionsFor(
	rule config.Rule,
	unit parsing.CodeUnit,
) string {
	var builder strings.Builder
	if unit.ParentSource != "" {
		builder.WriteString(
			"Determine whether state.source violates this rule. " +
				"Use state.parentSource only as surrounding context:\n",
		)
	} else {
		builder.WriteString("Determine whether the supplied code complies with this rule:\n")
	}
	builder.WriteString(rule.Description)
	if len(rule.Exceptions) > 0 {
		builder.WriteString("\n\nExplicit exceptions:")
		for _, exception := range rule.Exceptions {
			builder.WriteString("\n- ")
			builder.WriteString(exception)
		}
	}
	return builder.String()
}

// retryableStatus reports whether a response code should be tried again.
func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= 500
}

// retryDelay picks how long to wait before the next try.
func retryDelay(attempt int, headers http.Header) time.Duration {
	if headers != nil {
		if raw := headers.Get("retry-after-ms"); raw != "" {
			if milliseconds, err := strconv.ParseFloat(raw, 64); err == nil && milliseconds >= 0 {
				delay := time.Duration(milliseconds * float64(time.Millisecond))
				if delay <= retryAfterHeaderMax {
					return delay
				}
			}
		}
		if raw := headers.Get("Retry-After"); raw != "" {
			if seconds, err := strconv.ParseFloat(raw, 64); err == nil && seconds >= 0 {
				delay := time.Duration(seconds * float64(time.Second))
				if delay <= retryAfterHeaderMax {
					return delay
				}
			}
		}
	}

	delay := retryBackoffBase * time.Duration(1<<attempt)
	if delay > retryBackoffCap {
		return retryBackoffCap
	}
	return delay
}

// sleepContext waits for a duration or until the context ends.
func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// responseError builds an error from a failed response.
func responseError(status int, headers http.Header, body []byte) error {
	message := strings.TrimSpace(string(body))
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && payload.Error != "" {
		message = payload.Error
	}
	if message == "" {
		message = http.StatusText(status)
	}

	requestID := headers.Get("x-typesafe-request-id")
	if requestID != "" {
		return fmt.Errorf("TypeSafe API returned %d: %s (request %s)", status, message, requestID)
	}
	return fmt.Errorf("TypeSafe API returned %d: %s", status, message)
}
