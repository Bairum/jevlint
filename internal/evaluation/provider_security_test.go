package evaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestProviderErrorsEscapeTerminalControls(t *testing.T) {
	t.Parallel()

	var controls strings.Builder
	for r := rune(0); r < 32; r++ {
		controls.WriteRune(r)
	}
	for r := rune(127); r < 160; r++ {
		controls.WriteRune(r)
	}
	controls.WriteString("\u200b\u200d\u202e\u2066\ufeff\u2028\u2029")
	readable := "échec 日本語 Ω 🙂"
	message := readable + controls.String() + "message-end"
	requestID := "request-visible" + controls.String() + "request-end"

	for _, test := range []struct {
		provider Provider
		header   string
		payload  any
	}{
		{TypeSafeProvider{}, "x-typesafe-request-id", map[string]any{"error": message}},
		{CloudflareProvider{}, "cf-ray", map[string]any{"errors": []any{map[string]any{"code": 10000, "message": message}}}},
		{&OpenRouterProvider{}, "x-request-id", map[string]any{"error": map[string]any{"code": 401, "message": message}}},
		{&OpenRouterProvider{}, "cf-ray", map[string]any{"error": map[string]any{"message": message}}},
	} {
		t.Run(test.provider.Name()+"/"+test.header, func(t *testing.T) {
			jsonBody, err := json.Marshal(test.payload)
			if err != nil {
				t.Fatal(err)
			}
			for name, body := range map[string][]byte{
				"json": jsonBody,
				"raw":  []byte(message + "\xff\xfe"),
			} {
				t.Run(name, func(t *testing.T) {
					headers := http.Header{}
					headers.Set(test.header, requestID+"\xff")
					got := test.provider.ResponseError(http.StatusUnauthorized, headers, body).Error()
					if !utf8.ValidString(got) {
						t.Fatalf("error contains invalid UTF-8: %q", got)
					}
					for _, r := range got {
						if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
							t.Fatalf("error contains raw terminal control %U: %q", r, got)
						}
					}
					for _, want := range []string{readable, "message-end", "request-visible", "request-end"} {
						if !strings.Contains(got, want) {
							t.Errorf("error lost readable diagnostic %q: %q", want, got)
						}
					}
				})
			}
		})
	}
}

func TestProviderInitializationDebugOmitsPrivateConfiguration(t *testing.T) {
	t.Parallel()

	for _, provider := range []string{"typesafe", "cloudflare", "openrouter"} {
		t.Run(provider, func(t *testing.T) {
			model := "private-model-marker"
			if provider == "cloudflare" {
				model = "clef-flash"
			}
			env := map[string]string{
				"JEVLINT_PROVIDER": provider,
			}
			var logs bytes.Buffer
			_, err := NewClientFromEnv(Options{
				APIKey:   "private-key-marker",
				Endpoint: "https://diagnostic.invalid/private-path-marker?token=private-query-marker",
				Model:    model,
				Logf: func(format string, args ...any) {
					fmt.Fprintf(&logs, format+"\n", args...)
				},
			}, func(key string) string { return env[key] })
			if err != nil {
				t.Fatal(err)
			}
			if logs.Len() == 0 {
				t.Fatal("debug diagnostics unexpectedly empty")
			}
			for _, secret := range []string{"diagnostic.invalid", "private-path-marker", "private-query-marker", "private-key-marker", model} {
				if strings.Contains(logs.String(), secret) {
					t.Errorf("debug diagnostics expose %q: %q", secret, logs.String())
				}
			}
		})
	}
}
