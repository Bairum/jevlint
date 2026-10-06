package evaluation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const hardeningAnswers = `{"answers":{"database-joins":{"type":"choice","choice":"pass","probabilities":{"pass":1,"fail":0}},"semicolons":{"type":"choice","choice":"pass","probabilities":{"pass":1,"fail":0}}},"extra":"RESPONSE_SECRET"}`

func TestClientDebugMetadataOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, hardeningAnswers)
	}))
	defer server.Close()
	var logs strings.Builder
	client, err := NewClient(Options{
		APIKey: "CREDENTIAL_SECRET", Endpoint: server.URL + "/PATH_SECRET?token=QUERY_SECRET",
		HTTPClient: server.Client(),
		Logf:       func(format string, args ...any) { fmt.Fprintf(&logs, format+"\n", args...) },
	})
	if err != nil {
		t.Fatal(err)
	}
	batch := testBatch()
	batch.CodeUnit.Source = "SOURCE_SECRET"
	if _, err := client.Evaluate(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"CREDENTIAL_SECRET", "SOURCE_SECRET", "RESPONSE_SECRET", "PATH_SECRET", "QUERY_SECRET", server.URL} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("debug log exposed %q", secret)
		}
	}
	if !strings.Contains(logs.String(), "200") {
		t.Error("debug log omitted response status")
	}
}

func TestClientRejectsUnsafeEndpoints(t *testing.T) {
	for _, endpoint := range []string{
		"http://example.com/PATH_SECRET?token=QUERY_SECRET", "http://127.0.0.1.example.com", "http://dev.localhost", "http://localhost.",
		"https://USER_SECRET:PASS_SECRET@example.com", "https://example.com/#FRAGMENT_SECRET", "ftp://example.com",
		"https://example.com:70000", "https://example.com:", "https://[not-an-ip]", "https://example..com", "https:///PATH_SECRET", "https://example.com/%zz",
	} {
		t.Run(endpoint, func(t *testing.T) {
			for _, base := range []bool{false, true} {
				options := Options{APIKey: "test"}
				if base {
					options.BaseURL = ServiceURL(endpoint)
				} else {
					options.Endpoint = endpoint
				}
				_, err := NewClient(options)
				if err == nil {
					t.Fatalf("accepted unsafe URL %q", endpoint)
				}
				for _, secret := range []string{endpoint, "PATH_SECRET", "QUERY_SECRET", "USER_SECRET", "PASS_SECRET", "FRAGMENT_SECRET"} {
					if strings.Contains(err.Error(), secret) {
						t.Errorf("endpoint error exposed %q", secret)
					}
				}
			}
		})
	}
	for _, endpoint := range []string{"https://example.com", "HTTP://LOCALHOST:1234", "http://127.1.2.3:1234", "http://[::1]:1234"} {
		if _, err := NewClient(Options{APIKey: "test", Endpoint: endpoint}); err != nil {
			t.Errorf("rejected allowed endpoint %q: %v", endpoint, err)
		}
	}
}

func TestClientBlocksRedirectDestinations(t *testing.T) {
	for _, downgrade := range []bool{false, true} {
		t.Run(fmt.Sprintf("downgrade=%t", downgrade), func(t *testing.T) {
			var contacted atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { contacted.Add(1); fmt.Fprint(w, hardeningAnswers) })
			var target *httptest.Server
			if downgrade {
				target = httptest.NewServer(handler)
			} else {
				target = httptest.NewTLSServer(handler)
			}
			defer target.Close()
			source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target.URL+"/PATH_SECRET?token=QUERY_SECRET", http.StatusTemporaryRedirect)
			}))
			defer source.Close()
			client, err := NewClient(Options{APIKey: "test", Endpoint: source.URL, HTTPClient: source.Client(), MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Evaluate(context.Background(), testBatch())
			if err == nil {
				t.Error("redirect succeeded")
			}
			if contacted.Load() != 0 {
				t.Errorf("redirect destination received %d requests", contacted.Load())
			}
			if err != nil && (strings.Contains(err.Error(), "PATH_SECRET") || strings.Contains(err.Error(), "QUERY_SECRET") || strings.Contains(err.Error(), source.URL)) {
				t.Errorf("redirect error exposed URL: %v", err)
			}
		})
	}
}

func TestClientAllowsSameOriginBodyRedirect(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/start" {
					http.Redirect(w, r, "/finish", status)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if r.Method != http.MethodPost || !strings.Contains(string(body), "SOURCE_SECRET") || r.Header.Get("Authorization") != "Bearer test" {
					t.Error("same-origin redirect lost request body or authorization")
				}
				fmt.Fprint(w, hardeningAnswers)
			}))
			defer server.Close()
			client, err := NewClient(Options{APIKey: "test", Endpoint: server.URL + "/start", HTTPClient: server.Client()})
			if err != nil {
				t.Fatal(err)
			}
			batch := testBatch()
			batch.CodeUnit.Source = "SOURCE_SECRET"
			if _, err := client.Evaluate(context.Background(), batch); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientPreservesCallerRedirectPolicy(t *testing.T) {
	denied := errors.New("caller denied redirect")
	var called atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/finish", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/finish", http.StatusTemporaryRedirect)
	}))
	defer other.Close()
	supplied := server.Client()
	supplied.CheckRedirect = func(r *http.Request, via []*http.Request) error { called.Add(1); return denied }
	client, err := NewClient(Options{APIKey: "test", Endpoint: server.URL, HTTPClient: supplied, MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Evaluate(context.Background(), testBatch())
	if !errors.Is(err, denied) || called.Load() == 0 {
		t.Fatalf("caller restriction lost: %v, calls=%d", err, called.Load())
	}
	before := called.Load()
	_, err = supplied.Get(other.URL)
	if !errors.Is(err, denied) || called.Load() != before+1 {
		t.Fatalf("supplied client's policy was mutated: %v", err)
	}
}

type hardeningTransport func(*http.Request) (*http.Response, error)

func (transport hardeningTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestClientTransportErrorsHideURLAndPreserveCause(t *testing.T) {
	cause := errors.New("underlying transport failure")
	supplied := &http.Client{Transport: hardeningTransport(func(r *http.Request) (*http.Response, error) {
		return nil, &url.Error{Op: "Post", URL: r.URL.String(), Err: cause}
	})}
	client, err := NewClient(Options{APIKey: "test", Endpoint: "https://example.com/PATH_SECRET?token=QUERY_SECRET", HTTPClient: supplied, MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Evaluate(context.Background(), testBatch())
	if !errors.Is(err, cause) || !strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("transport cause lost: %v", err)
	}
	for _, secret := range []string{"PATH_SECRET", "QUERY_SECRET", "https://example.com"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("transport error exposed %q", secret)
		}
	}
}

func TestClientRejectsUserinfoAndMalformedRedirects(t *testing.T) {
	for _, location := range []string{"/PATH_SECRET%zz?token=QUERY_SECRET", "userinfo"} {
		t.Run(location, func(t *testing.T) {
			var contacted atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/start" {
					contacted.Add(1)
					fmt.Fprint(w, hardeningAnswers)
					return
				}
				destination := location
				if location == "userinfo" {
					destination = "http://USER_SECRET:PASS_SECRET@" + r.Host + "/finish"
				}
				w.Header().Set("Location", destination)
				w.WriteHeader(http.StatusTemporaryRedirect)
			}))
			defer server.Close()
			client, err := NewClient(Options{APIKey: "test", Endpoint: server.URL + "/start", HTTPClient: server.Client(), MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Evaluate(context.Background(), testBatch())
			if err == nil || contacted.Load() != 0 {
				t.Fatalf("unsafe redirect: err=%v, destination requests=%d", err, contacted.Load())
			}
			for _, secret := range []string{"PATH_SECRET", "QUERY_SECRET", "USER_SECRET", "PASS_SECRET", server.URL} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("redirect error exposed %q", secret)
				}
			}
		})
	}
}

func TestClientDefaultRedirectLimit(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Redirect(w, r, "/again", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, err := NewClient(Options{APIKey: "test", Endpoint: server.URL, HTTPClient: server.Client(), MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Evaluate(context.Background(), testBatch()); err == nil {
		t.Fatal("redirect loop succeeded")
	}
	if requests.Load() != 20 {
		t.Fatalf("requests=%d, want ten per attempt", requests.Load())
	}
}

func TestClientDebugIgnoresServerStatusText(t *testing.T) {
	var logs strings.Builder
	client, err := NewClient(Options{
		APIKey: "test", Endpoint: "https://example.com",
		HTTPClient: &http.Client{Transport: hardeningTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Status: "200 STATUS_SECRET", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(hardeningAnswers))}, nil
		})},
		Logf: func(format string, args ...any) { fmt.Fprintf(&logs, format+"\n", args...) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Evaluate(context.Background(), testBatch()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "STATUS_SECRET") {
		t.Fatal("debug log included untrusted HTTP status text")
	}
}

func TestClientBlocksRedirectPolicyDestinationRewrite(t *testing.T) {
	var contacted atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacted.Add(1)
		fmt.Fprint(w, hardeningAnswers)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/finish", http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	destination, err := url.Parse(target.URL + "/finish")
	if err != nil {
		t.Fatal(err)
	}
	supplied := source.Client()
	supplied.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		*request.URL = *destination
		// Mutating the request history must not change the trusted origin either.
		*via[0].URL = *destination
		return nil
	}
	client, err := NewClient(Options{APIKey: "test", Endpoint: source.URL, HTTPClient: supplied, MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Evaluate(context.Background(), testBatch())
	if err == nil || contacted.Load() != 0 {
		t.Fatalf("rewritten redirect: error=%v, destination requests=%d", err, contacted.Load())
	}
}

func TestClientTransportErrorRedactionKeepsWrappedReason(t *testing.T) {
	cause := errors.New("connection refused")
	supplied := &http.Client{Transport: hardeningTransport(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("dial failure for %s: %w", r.URL.String(), cause)
	})}
	client, err := NewClient(Options{APIKey: "test", Endpoint: "https://example.com/PATH_SECRET?token=QUERY_SECRET", HTTPClient: supplied, MaxRetries: 1, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Evaluate(context.Background(), testBatch())
	if !errors.Is(err, cause) || !strings.Contains(err.Error(), "dial failure") || !strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("wrapped failure reason lost: %v", err)
	}
	for _, secret := range []string{"PATH_SECRET", "QUERY_SECRET", "https://example.com"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("wrapped transport error exposed %q", secret)
		}
	}
}

func TestClientSameOriginNormalizesHostAndEffectivePort(t *testing.T) {
	var requests atomic.Int32
	supplied := &http.Client{Transport: hardeningTransport(func(r *http.Request) (*http.Response, error) {
		if requests.Add(1) == 1 {
			return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {"https://EXAMPLE.COM:00443/finish"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(hardeningAnswers))}, nil
	})}
	client, err := NewClient(Options{APIKey: "test", Endpoint: "https://example.com/start", HTTPClient: supplied})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Evaluate(context.Background(), testBatch()); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("same-origin redirect made %d requests, want two", requests.Load())
	}
}
