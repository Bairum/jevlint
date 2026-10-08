package evaluation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"runtime"
	"sort"
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
	cacheKeyVersion     = "typesafe-evaluation-v3"
	httpSuccessMin      = 200
	httpSuccessLimit    = 300
	retryBackoffBase    = 500 * time.Millisecond
	retryBackoffCap     = 5 * time.Second
	retryAfterHeaderMax = time.Minute
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
	// Budget, when set, estimates a serialized request before it is sent.
	// Nil disables the pre-flight check. Unknown models are not checked.
	Budget func([]byte) (Estimate, error)
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
	budget      func([]byte) (Estimate, error)
	logf        func(string, ...any)
	warnf       func(string, ...any)
	inflightMu  sync.Mutex
	inflight    map[string]*evaluationCall
	cacheHits   atomic.Uint64
	cacheMisses atomic.Uint64
	cacheWrites atomic.Uint64
	metaMu      sync.Mutex
	models      map[string]struct{}
	usage       Usage
	oversized   []Oversized
	dropped     []ContextDrop
}

// evaluationCall tracks one running request that callers share.
type evaluationCall struct {
	done    chan struct{}
	waiters atomic.Int32
	answers map[string]QuestionAnswer
	model   string
	err     error
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
		budget:     options.Budget,
		logf:       options.Logf,
		warnf:      options.Warnf,
		inflight:   make(map[string]*evaluationCall),
		models:     make(map[string]struct{}),
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
	if answers, model, ok, err := client.cachedUnsplit(batch); err != nil {
		return nil, err
	} else if ok {
		if model != "" {
			client.noteModel(model)
		}
		return decodeAnswers(answers, batch, model)
	}
	bodies, dropped, oversized, err := client.planRequests(batch)
	if err != nil {
		return nil, err
	}
	if oversized != nil {
		client.noteOversized(*oversized)
		return nil, ErrOversized
	}
	client.noteDrops(dropped)
	answers := make(map[string]QuestionAnswer)
	model := ""
	for _, body := range bodies {
		chunk, chunkModel, err := client.answersFor(ctx, body)
		if err != nil {
			return nil, err
		}
		if chunkModel != "" {
			model = chunkModel
		}
		for id, answer := range chunk {
			answers[id] = answer
		}
	}
	if model != "" {
		client.noteModel(model)
	}
	return decodeAnswers(answers, batch, model)
}

// cachedUnsplit returns a cache hit for the exact unsplit request body.
// A hit skips budget planning. A miss, including a request that will be
// split, falls through so each sent body keeps its own cache key.
func (client *Client) cachedUnsplit(batch Batch) (map[string]QuestionAnswer, string, bool, error) {
	if client.cache == nil || client.refresh {
		return nil, "", false, nil
	}
	state := stateFor(batch)
	questions, err := questionsFor(batch, state)
	if err != nil {
		return nil, "", false, err
	}
	body, err := marshalRequest(client.model, state, questions)
	if err != nil {
		return nil, "", false, err
	}
	hit, ok := client.hitCache(client.cacheKey(body))
	if !ok {
		return nil, "", false, nil
	}
	return cloneAnswers(hit.Answers), hit.Model, true, nil
}

// answersFor returns cached or live per-question answers for one request body.
func (client *Client) answersFor(
	ctx context.Context,
	body []byte,
) (map[string]QuestionAnswer, string, error) {
	if client.cache == nil {
		return client.fetchAnswers(ctx, body)
	}
	key := client.cacheKey(body)
	if hit, ok := client.hitCache(key); ok {
		return cloneAnswers(hit.Answers), hit.Model, nil
	}
	return client.evaluateOnce(ctx, key, func() (map[string]QuestionAnswer, string, error) {
		if hit, ok := client.hitCache(key); ok {
			return cloneAnswers(hit.Answers), hit.Model, nil
		}
		client.cacheMisses.Add(1)
		fetched, fetchedModel, err := client.fetchAnswers(ctx, body)
		if err != nil {
			return nil, "", err
		}
		if client.cache.Put(key, CacheHit{Model: fetchedModel, Answers: fetched}) {
			client.cacheWrites.Add(1)
		}
		return fetched, fetchedModel, nil
	})
}

// fetchAnswers sends one request and records its usage.
func (client *Client) fetchAnswers(
	ctx context.Context,
	body []byte,
) (map[string]QuestionAnswer, string, error) {
	responseBody, err := client.perform(ctx, body)
	if err != nil {
		return nil, "", err
	}
	response, err := client.provider.Answers(responseBody)
	if err != nil {
		return nil, "", err
	}
	client.noteUsage(response.Usage)
	client.noteModel(response.Model)
	return response.Answers, response.Model, nil
}

// hitCache returns cached answers for a request when they are usable.
func (client *Client) hitCache(key string) (CacheHit, bool) {
	if client.refresh || client.cache == nil {
		return CacheHit{}, false
	}
	hit, ok := client.cache.Get(key)
	if !ok || len(hit.Answers) == 0 {
		return CacheHit{}, false
	}
	for _, answer := range hit.Answers {
		if err := answer.validate(); err != nil {
			return CacheHit{}, false
		}
	}
	client.cacheHits.Add(1)
	client.noteModel(hit.Model)
	return hit, true
}

// evaluateOnce runs a request once and shares the answer with waiting callers.
func (client *Client) evaluateOnce(
	ctx context.Context,
	key string,
	evaluate func() (map[string]QuestionAnswer, string, error),
) (map[string]QuestionAnswer, string, error) {
	client.inflightMu.Lock()
	if call, ok := client.inflight[key]; ok {
		call.waiters.Add(1)
		client.inflightMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		case <-call.done:
			return cloneAnswers(call.answers), call.model, call.err
		}
	}
	call := &evaluationCall{done: make(chan struct{})}
	client.inflight[key] = call
	client.inflightMu.Unlock()

	call.answers, call.model, call.err = evaluate()

	client.inflightMu.Lock()
	delete(client.inflight, key)
	close(call.done)
	client.inflightMu.Unlock()
	return cloneAnswers(call.answers), call.model, call.err
}

// inflightAttached reports how many callers are on the current single-flight calls.
func (client *Client) inflightAttached() int {
	client.inflightMu.Lock()
	defer client.inflightMu.Unlock()
	total := 0
	for _, call := range client.inflight {
		total += 1 + int(call.waiters.Load())
	}
	return total
}

// RunMeta returns models, usage, and budget fallout recorded so far.
func (client *Client) RunMeta() RunMeta {
	client.metaMu.Lock()
	defer client.metaMu.Unlock()
	return RunMeta{
		Models:         sortedModels(client.models),
		Usage:          client.usage,
		Oversized:      append([]Oversized(nil), client.oversized...),
		ContextDropped: append([]ContextDrop(nil), client.dropped...),
	}
}

func (client *Client) noteModel(model string) {
	if model == "" {
		return
	}
	client.metaMu.Lock()
	defer client.metaMu.Unlock()
	if client.models == nil {
		client.models = make(map[string]struct{})
	}
	client.models[model] = struct{}{}
}

func (client *Client) noteUsage(usage Usage) {
	client.metaMu.Lock()
	client.usage.Requests++
	client.usage.InputTokens += usage.InputTokens
	client.usage.OutputTokens += usage.OutputTokens
	client.metaMu.Unlock()
}

func (client *Client) noteOversized(item Oversized) {
	client.metaMu.Lock()
	client.oversized = append(client.oversized, item)
	client.metaMu.Unlock()
}

func (client *Client) noteDrops(dropped []ContextDrop) {
	if len(dropped) == 0 {
		return
	}
	client.metaMu.Lock()
	client.dropped = append(client.dropped, dropped...)
	client.metaMu.Unlock()
}

func sortedModels(models map[string]struct{}) []string {
	if len(models) == 0 {
		return []string{}
	}
	names := make([]string, 0, len(models))
	for name := range models {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func cloneAnswers(answers map[string]QuestionAnswer) map[string]QuestionAnswer {
	if answers == nil {
		return nil
	}
	cloned := make(map[string]QuestionAnswer, len(answers))
	for id, answer := range answers {
		if answer.Noul != nil {
			value := *answer.Noul
			answer.Noul = &value
		}
		if answer.Score != nil {
			value := *answer.Score
			answer.Score = &value
		}
		cloned[id] = answer
	}
	return cloned
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
