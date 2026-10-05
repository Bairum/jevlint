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
	"net/netip"
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
	defaultTimeout      = 10 * time.Second
	defaultMaxRetries   = 2
	maxResponseBytes    = 1 << 20
	cacheKeyVersion     = "typesafe-evaluation-v1"
	answerTypeChoice    = "choice"
	httpSuccessMin      = 200
	httpSuccessLimit    = 300
	retryBackoffBase    = 500 * time.Millisecond
	retryBackoffCap     = 5 * time.Second
	retryAfterHeaderMax = time.Minute
	criterionPass       = "The code complies with the rule, or an explicit exception applies."
	criterionFail       = "The code violates the rule, and no explicit exception applies."
	criterionSkip       = "The rule's subject is not present in this unit; the rule does not apply."
	criterionAbstain    = "The rule is relevant, but the supplied code does not contain enough context to decide pass or fail."
	minimumBatchRules   = 1
)

// APIKey is the credential sent to the service.
type APIKey string

// ServiceURL is the base address of the service.
type ServiceURL string

// Options holds the settings for a client.
type Options struct {
	APIKey APIKey
	// Endpoint is the full request URL. When empty, BaseURL plus
	// "/v1/systemone" is used.
	Endpoint   string
	BaseURL    ServiceURL
	Model      string
	HTTPClient *http.Client
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
	Cache      ResultCache
	Refresh    bool
	// Logf, when set, receives request and response metadata only.
	// URLs, headers, and body contents are never logged.
	Logf func(format string, args ...any)
	// Warnf, when set, receives warnings such as a provider left at its
	// default while another provider's variables are present.
	Warnf func(format string, args ...any)
}

// Client talks to a SystemOne-compatible decision service.
type Client struct {
	apiKey      string
	endpoint    string
	model       string
	provider    Provider
	httpClient  *http.Client
	maxRetries  int
	sleep       func(context.Context, time.Duration) error
	cache       ResultCache
	refresh     bool
	logf        func(string, ...any)
	warnf       func(string, ...any)
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
	Guidance     []string               `json:"guidance,omitempty"`
}

// requestType is a related type sent as context.
type requestType struct {
	Name   string `json:"name"`
	Path   string `json:"path,omitempty"`
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

// NewClient builds a client with the default provider.
func NewClient(options Options) (*Client, error) {
	return NewClientWithProvider(TypeSafeProvider{}, options)
}

// NewClientWithProvider builds a client from the given settings and provider.
func NewClientWithProvider(provider Provider, options Options) (*Client, error) {
	apiKey := strings.TrimSpace(string(options.APIKey))
	if apiKey == "" {
		return nil, errors.New("an API key is required: set TYPESAFE_API_KEY")
	}

	endpoint := strings.TrimSpace(options.Endpoint)
	if endpoint == "" {
		baseURL := strings.TrimRight(strings.TrimSpace(string(options.BaseURL)), "/")
		if baseURL == "" {
			baseURL = provider.BaseURL()
		}
		if _, err := validateServiceURL(baseURL); err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
		endpoint = baseURL + "/v1/systemone"
	}
	parsedEndpoint, err := validateServiceURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}
	endpoint = parsedEndpoint.String()

	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = provider.DefaultModel()
	}

	httpClient := http.Client{Timeout: defaultTimeout}
	if options.HTTPClient != nil {
		httpClient = *options.HTTPClient
	}
	callerRedirect := httpClient.CheckRedirect
	httpClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if callerRedirect != nil {
			if err := callerRedirect(request, via); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		// The caller's policy may rewrite URL or via; check the final URL
		// against the separately parsed original endpoint, not request history.
		if request.URL == nil {
			return errors.New("redirect rejected: invalid destination")
		}
		if _, err := validateServiceURL(request.URL.String()); err != nil {
			return errors.New("redirect rejected: invalid destination")
		}
		if !sameOrigin(parsedEndpoint, request.URL) {
			return errors.New("redirect rejected: destination must have the same origin")
		}
		return nil
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

	return &Client{
		apiKey:     apiKey,
		endpoint:   endpoint,
		model:      model,
		provider:   provider,
		httpClient: &httpClient,
		maxRetries: maxRetries,
		sleep:      sleep,
		cache:      options.Cache,
		refresh:    options.Refresh,
		logf:       options.Logf,
		warnf:      options.Warnf,
		inflight:   make(map[string]*evaluationCall),
	}, nil
}

// validateServiceURL accepts TLS endpoints and explicitly local HTTP development.
// Errors deliberately omit all URL components, which can carry credentials.
func validateServiceURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || parsed.Host == "" {
		return nil, errors.New("expected an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.Fragment != "" || strings.Contains(value, "#") {
		return nil, errors.New("credentials and fragments are not allowed")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return nil, errors.New("scheme must be HTTPS")
	}
	host := parsed.Hostname()
	address, addressErr := netip.ParseAddr(host)
	if addressErr != nil {
		if strings.Contains(host, ":") || strings.HasPrefix(parsed.Host, "[") {
			return nil, errors.New("invalid host")
		}
		domain := strings.TrimSuffix(host, ".")
		if len(domain) == 0 || len(domain) > 253 {
			return nil, errors.New("invalid host")
		}
		for _, label := range strings.Split(domain, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return nil, errors.New("invalid host")
			}
			for _, character := range label {
				if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
					character >= '0' && character <= '9' || character == '-') {
					return nil, errors.New("invalid host")
				}
			}
		}
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return nil, errors.New("invalid port")
		}
	} else if strings.HasSuffix(parsed.Host, ":") {
		return nil, errors.New("invalid port")
	}
	if scheme == "http" && !strings.EqualFold(host, "localhost") && (addressErr != nil || !address.IsLoopback()) {
		return nil, errors.New("HTTP is allowed only for localhost or a literal loopback IP")
	}
	parsed.Scheme = scheme
	return parsed, nil
}

func sameOrigin(first, second *url.URL) bool {
	return strings.EqualFold(first.Scheme, second.Scheme) &&
		strings.EqualFold(first.Hostname(), second.Hostname()) &&
		effectivePort(first) == effectivePort(second)
}

func effectivePort(value *url.URL) int {
	if port := value.Port(); port != "" {
		number, _ := strconv.Atoi(port)
		return number
	}
	if strings.EqualFold(value.Scheme, "https") {
		return 443
	}
	return 80
}

// transportFailure preserves error causes and diagnostic reasons without
// rendering URL fields from transport wrappers.
type transportFailure struct {
	cause error
}

func (failure transportFailure) Error() string {
	err := failure.cause
	for {
		urlError, ok := err.(*url.Error)
		if !ok {
			break
		}
		err = urlError.Err
	}
	message := err.Error()
	if strings.HasPrefix(message, "failed to parse Location header") {
		return "invalid redirect destination"
	}
	for cause := failure.cause; cause != nil; cause = errors.Unwrap(cause) {
		if urlError, ok := cause.(*url.Error); ok && urlError.URL != "" {
			// %q escaping can differ from the raw URL for queries with spaces
			// or quotes. Redact both forms without removing the failure reason.
			message = strings.ReplaceAll(message, strconv.Quote(urlError.URL), `"<redacted>"`)
			message = strings.ReplaceAll(message, urlError.URL, "<redacted>")
		}
	}
	return message
}

func (failure transportFailure) Unwrap() error {
	return failure.cause
}

func safeTransportError(err error) error {
	return transportFailure{cause: err}
}

// Evaluate answers the rules in a batch for one piece of code.
func (client *Client) Evaluate(ctx context.Context, batch Batch) (map[string]Result, error) {
	body, err := client.requestBody(batch)
	if err != nil {
		return nil, err
	}
	if client.cache == nil {
		responseBody, err := client.perform(ctx, body)
		if err != nil {
			return nil, err
		}
		return client.decode(responseBody, batch.Rules)
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
		results, err := client.decode(responseBody, batch.Rules)
		if err != nil {
			return nil, err
		}
		if client.cache.Put(key, results) {
			client.cacheWrites.Add(1)
		}
		return results, nil
	})
}

// decode extracts the answers from a response and checks them against the rules.
func (client *Client) decode(
	responseBody []byte,
	rules []config.Rule,
) (map[string]Result, error) {
	answers, err := client.provider.Answers(responseBody)
	if err != nil {
		return nil, err
	}
	return decodeResults(answers, rules)
}

// hitCache returns cached results for a request when they are usable.
func (client *Client) hitCache(key string, rules []config.Rule) (map[string]Result, bool) {
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
func (client *Client) cachedResults(
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
func (client *Client) evaluateOnce(
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
func (client *Client) CacheStats() CacheStats {
	return CacheStats{
		Hits:   client.cacheHits.Load(),
		Misses: client.cacheMisses.Load(),
		Writes: client.cacheWrites.Load(),
	}
}

// Endpoint returns the resolved request URL.
func (client *Client) Endpoint() string {
	return client.endpoint
}

// Model returns the resolved model name.
func (client *Client) Model() string {
	return client.model
}

// CredentialKind describes the credential without revealing it.
func (client *Client) CredentialKind() string {
	return client.provider.DescribeCredential(client.apiKey)
}

// Ping verifies that the service answers a trivial request. It always makes a
// live call, ignoring any cache.
func (client *Client) Ping(ctx context.Context) error {
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

// requestBody builds the body sent to the service.
func (client *Client) requestBody(batch Batch) ([]byte, error) {
	questions, guidance, err := questionsForBatch(batch)
	if err != nil {
		return nil, err
	}
	if err := client.provider.ValidateQuestions(questions); err != nil {
		return nil, err
	}
	state := requestStateFrom(batch.CodeUnit)
	state.Guidance = guidance
	body, err := json.Marshal(systemOneRequest{
		Model:     client.model,
		State:     state,
		Questions: questions,
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	if max := client.provider.MaxBodyBytes(); max > 0 && len(body) > max {
		return nil, fmt.Errorf(
			"request body is %d bytes, over the %d-byte limit",
			len(body),
			max,
		)
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
			requested := requestType{Name: declaration.Name, Source: declaration.Source}
			// Same-file context needs no path; sending it would repeat state.path.
			if declaration.Path != unit.Path {
				requested.Path = declaration.Path
			}
			state.RelatedTypes = append(state.RelatedTypes, requested)
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
func questionsForBatch(batch Batch) (map[string]question, []string, error) {
	if len(batch.Rules) < minimumBatchRules {
		return nil, nil, errors.New("at least one rule is required")
	}
	questions := make(map[string]question, len(batch.Rules))
	var guidance []string
	guidanceIndices := make(map[string]int)
	for _, rule := range batch.Rules {
		if _, exists := questions[rule.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate rule id %q in evaluation batch", rule.ID)
		}
		instructions := instructionsFor(rule, batch.CodeUnit)
		if rule.Guidance != "" {
			index, exists := guidanceIndices[rule.Guidance]
			if !exists {
				index = len(guidance)
				guidanceIndices[rule.Guidance] = index
				guidance = append(guidance, rule.Guidance)
			}
			instructions += fmt.Sprintf("\n\nAlso apply the shared guidance in state.guidance[%d].", index)
		}
		questions[rule.ID] = question{
			Type:         questionTypeChoice,
			Instructions: instructions,
			Criteria:     criteriaFor(rule),
		}
	}
	return questions, guidance, nil
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

// decodeResults checks the answers against the rules.
func decodeResults(
	answers map[string]choiceAnswer,
	rules []config.Rule,
) (map[string]Result, error) {
	if answers == nil {
		return nil, errors.New("decode response: answers are missing")
	}

	results := make(map[string]Result, len(rules))
	for _, rule := range rules {
		answer, ok := answers[rule.ID]
		if !ok {
			return nil, fmt.Errorf("decode response: answer for rule %q is missing", rule.ID)
		}
		if answer.Type != questionTypeChoice {
			return nil, fmt.Errorf(
				"decode response: answer for rule %q has unsupported type",
				rule.ID,
			)
		}
		if answer.Confidence == nil {
			return nil, fmt.Errorf(
				"decode response: confidence for rule %q is missing",
				rule.ID,
			)
		}

		status, err := ParseStatus(answer.Choice)
		if err != nil {
			return nil, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
		}
		if status == StatusSkip && !rule.AllowSkip {
			return nil, fmt.Errorf(
				"decode response: rule %q returned skip without allowSkip",
				rule.ID,
			)
		}
		if status == StatusAbstain && !rule.AllowAbstain {
			return nil, fmt.Errorf(
				"decode response: rule %q returned abstain without allowAbstain",
				rule.ID,
			)
		}
		result := Result{
			Status:     status,
			Confidence: *answer.Confidence,
		}
		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
		}
		results[rule.ID] = result
	}
	return results, nil
}

// cacheKey returns the key that identifies a request.
func (client *Client) cacheKey(body []byte) string {
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
func (client *Client) perform(ctx context.Context, body []byte) ([]byte, error) {
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
				return nil, fmt.Errorf("request failed: %w", safeTransportError(err))
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
		client.debugf("jevlint: response status=%d bytes=%d", response.StatusCode, len(responseBody))
		if response.StatusCode >= httpSuccessMin && response.StatusCode < httpSuccessLimit {
			return responseBody, nil
		}
		if attempt >= client.maxRetries || !retryableStatus(response.StatusCode) {
			return nil, client.provider.ResponseError(
				response.StatusCode,
				response.Header,
				responseBody,
			)
		}
		if sleepErr := client.sleep(ctx, retryDelay(attempt, response.Header)); sleepErr != nil {
			return nil, sleepErr
		}
	}
}

// newRequest builds one request to the service.
func (client *Client) newRequest(
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
		return nil, fmt.Errorf("create request: %w", safeTransportError(err))
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "github.com/codegirl-007/jevlint/0.1.0")
	request.Header.Set("X-Jevlint-SDK", "github.com/codegirl-007/jevlint/0.1.0")
	request.Header.Set("X-Jevlint-Runtime", runtime.Version())
	for key, value := range client.provider.Headers() {
		if value != "" {
			request.Header.Set(key, value)
		}
	}
	if attempt > 0 {
		request.Header.Set("X-Jevlint-Retry-Count", strconv.Itoa(attempt))
	}
	client.debugf("jevlint: request POST bytes=%d", len(body))
	return request, nil
}

// readResponse reads and closes a response body.
func readResponse(response *http.Response) ([]byte, error) {
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close response: %w", closeErr)
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
	if len(unit.Callees) > 0 || len(unit.RelatedTypes) > 0 {
		builder.WriteString("\n\nJudge only state.source. state.types and state.callees are context only; do not fail state.source for problems that exist only in them.")
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

// debugf writes one debug line when request logging is enabled.
func (client *Client) debugf(format string, args ...any) {
	if client.logf != nil {
		client.logf(format, args...)
	}
}

// warn writes one warning when a warning sink is configured.
func (client *Client) warn(format string, args ...any) {
	if client.warnf != nil {
		client.warnf(format, args...)
	}
}
