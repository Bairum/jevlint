package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeValidConfig(t *testing.T) {
	t.Parallel()

	cfg, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "database-joins",
			"description": "Join related database records in the database.",
			"severity": "error",
			"include": ["**/*.go"],
			"kinds": ["comment", "docComment", "field", "function", "statement", "type"],
			"localize": ["docComment", "statement"]
		}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].ID != "database-joins" {
		t.Fatalf("Decode() rules = %#v", cfg.Rules)
	}
	if cfg.MinFailProbability != nil {
		t.Fatalf("Decode() minFailProbability = %#v, want omitted", cfg.MinFailProbability)
	}
	if cfg.Rules[0].Score != nil || cfg.Rules[0].Checks != nil {
		t.Fatalf("signals = score:%#v checks:%#v, want omitted", cfg.Rules[0].Score, cfg.Rules[0].Checks)
	}
	if cfg.Rules[0].Context.Callees {
		t.Fatalf("context = %#v, want omitted", cfg.Rules[0].Context)
	}
}

func TestValidateSourceMatch(t *testing.T) {
	t.Parallel()

	for _, partial := range []bool{false, true} {
		for _, pattern := range []string{"", " ", "[", `\bunsafe\b`} {
			cfg := Config{
				Languages: map[string]Language{"rust": {}},
				Rules:     []Rule{{ID: "subject", SourceMatch: []string{pattern}}},
			}
			if partial {
				cfg.Packs = []PackRef{{
					ID: "org/rules", Source: "local", SHA: strings.Repeat("a", 40),
				}}
			} else {
				cfg.Rules[0].Description = "A rule."
				cfg.Rules[0].Severity = SeverityWarning
			}
			err := cfg.Validate()
			if pattern == `\bunsafe\b` {
				if err != nil {
					t.Fatalf("partial=%v valid pattern: %v", partial, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "sourceMatch[0]") {
				t.Fatalf("partial=%v pattern=%q error=%v", partial, pattern, err)
			}
		}
	}
}

func TestParseTargetKindAndSeverity(t *testing.T) {
	t.Parallel()

	if kind, err := ParseTargetKind(KindFunction); err != nil || kind != TargetKindFunction {
		t.Fatalf("ParseTargetKind(%q) = %v, %v", KindFunction, kind, err)
	}
	if _, err := ParseTargetKind("banana"); err == nil || err.Error() != `invalid kind "banana"` {
		t.Fatalf("ParseTargetKind(banana) error = %v", err)
	}

	if severity, err := ParseSeverity("error"); err != nil || severity != SeverityError {
		t.Fatalf("ParseSeverity(error) = %v, %v", severity, err)
	}
	if _, err := ParseSeverity("erorr"); err == nil || err.Error() != `invalid severity "erorr"` {
		t.Fatalf("ParseSeverity(erorr) error = %v", err)
	}
}

func TestDecodeAllowsNoRules(t *testing.T) {
	t.Parallel()

	cfg, err := Decode(strings.NewReader(`{"languages": {"go": {}}}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(cfg.Rules) != 0 || len(cfg.Packs) != 0 {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestDecodeRuleContext(t *testing.T) {
	t.Parallel()

	disabled, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "database-joins",
			"description": "Join related database records in the database.",
			"severity": "error",
			"context": { "callees": false }
		}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if disabled.Rules[0].Context.Callees {
		t.Fatalf("context.callees = %#v, want false", disabled.Rules[0].Context)
	}

	enabled, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "database-joins",
			"description": "Join related database records in the database.",
			"severity": "error",
			"kinds": ["function"],
			"context": { "callees": true }
		}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !enabled.Rules[0].Context.Callees {
		t.Fatalf("context.callees = %#v, want true", enabled.Rules[0].Context)
	}
}

func TestDecodeRejectsRemovedRuleFields(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"allowSkip", "allowAbstain", "failWhen"} {
		field := field
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			_, err := Decode(strings.NewReader(withGoLanguage(`{
				"rules": [{
					"id": "database-joins",
					"description": "Join related database records in the database.",
					"severity": "error",
					"` + field + `": true
				}]
			}`)))
			if err == nil || !strings.Contains(err.Error(), field+" "+RuleFormatRemoved) {
				t.Fatalf("Decode() error = %v", err)
			}
		})
	}
	_, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "database-joins",
			"description": "Join related database records in the database.",
			"severity": "error",
			"checks": [{"id": "present", "question": "Is it present?", "failWhen": true}]
		}]
	}`)))
	if err == nil || !strings.Contains(err.Error(), "checks "+RuleFormatRemoved) {
		t.Fatalf("Decode() checks array error = %v", err)
	}
}

func TestDecodeMinFailProbability(t *testing.T) {
	t.Parallel()

	zero, err := Decode(strings.NewReader(withGoLanguage(`{
		"minFailProbability": 0,
		"rules": [{"id": "one", "description": "A rule.", "severity": "info"}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if zero.MinFailProbability == nil || *zero.MinFailProbability != 0 {
		t.Fatalf("Decode() minFailProbability = %#v, want 0", zero.MinFailProbability)
	}

	floor, err := Decode(strings.NewReader(withGoLanguage(`{
		"minFailProbability": 0.8,
		"rules": [{"id": "one", "description": "A rule.", "severity": "info"}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if floor.MinFailProbability == nil || *floor.MinFailProbability != 0.8 {
		t.Fatalf("Decode() minFailProbability = %#v, want 0.8", floor.MinFailProbability)
	}

	ruleZero, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "one",
			"description": "A rule.",
			"severity": "info",
			"minFailProbability": 0
		}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() rule minFailProbability error = %v", err)
	}
	if ruleZero.Rules[0].MinFailProbability == nil || *ruleZero.Rules[0].MinFailProbability != 0 {
		t.Fatalf("Decode() rule minFailProbability = %#v, want 0", ruleZero.Rules[0].MinFailProbability)
	}
}

func TestDecodeRejectsMinConfidence(t *testing.T) {
	t.Parallel()

	_, err := Decode(strings.NewReader(withGoLanguage(`{
		"minConfidence": 0.8,
		"rules": [{"id": "one", "description": "A rule.", "severity": "info"}]
	}`)))
	if err == nil || !strings.Contains(err.Error(), MinConfidenceReplaced) {
		t.Fatalf("Decode() error = %v", err)
	}
	_, err = Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{"id": "one", "description": "A rule.", "severity": "info", "minConfidence": 0.5}]
	}`)))
	if err == nil || !strings.Contains(err.Error(), MinConfidenceReplaced) {
		t.Fatalf("Decode() rule error = %v", err)
	}
}

func TestDecodeRejectsInvalidOverlaySeverity(t *testing.T) {
	t.Parallel()

	_, err := Decode(strings.NewReader(`{
		"languages": {"go": {}},
		"packs": [{"id": "database-joins", "source": "local", "sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],
		"rules": [{"id": "database-joins", "severity": "erorr"}]
	}`))
	if err == nil || !strings.Contains(err.Error(), `invalid severity "erorr"`) {
		t.Fatalf("Decode() error = %v, want invalid severity", err)
	}
}

func TestDecodeAllowsOverlayWithoutSeverity(t *testing.T) {
	t.Parallel()

	cfg, err := Decode(strings.NewReader(`{
		"languages": {"go": {}},
		"packs": [{"id": "database-joins", "source": "local", "sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],
		"rules": [{"id": "database-joins", "include": ["src/**/*.go"]}]
	}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].Severity != SeverityUnknown {
		t.Fatalf("rules = %#v", cfg.Rules)
	}
}

func TestWritePartialOverlayRoundTrips(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "jevlint.json")
	cfg, err := Decode(strings.NewReader(`{
		"languages": {"go": {}},
		"packs": [{"id": "database-joins", "source": "local", "sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],
		"rules": [{"id": "database-joins", "include": ["src/**/*.go"]}]
	}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"severity"`) {
		t.Fatalf("written config included an unknown severity: %s", data)
	}
	reloaded, err := Decode(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("reload error = %v; data = %s", err, data)
	}
	if len(reloaded.Rules) != 1 || len(reloaded.Rules[0].Include) != 1 {
		t.Fatalf("reloaded = %#v", reloaded)
	}
}

func TestWritePreservesModeAndFormatting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "jevlint.json")
	if err := os.WriteFile(path, []byte("original\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Languages: map[string]Language{"go": {}},
		Rules: []Rule{{
			ID:          "one",
			Description: "A rule.",
			Severity:    SeverityError,
		}},
	}
	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n  \"languages\"") {
		t.Fatalf("config is not indented: %s", data)
	}
	if _, err := Decode(strings.NewReader(string(data))); err != nil {
		t.Fatalf("written config does not decode: %v", err)
	}
}

func TestWriteFailureLeavesExistingConfigIntact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "jevlint.json")
	original := "{\n  \"keep\": true\n}\n"
	if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Languages: map[string]Language{"go": {}},
		Rules: []Rule{{
			ID:          "one",
			Description: "A rule.",
			Severity:    SeverityError,
			Kinds:       []TargetKind{TargetKindUnknown},
		}},
	}
	if err := Write(path, cfg); err == nil {
		t.Fatal("Write() error = nil, want a failure")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("config was modified: %q, want %q", data, original)
	}
}

func TestWriteFailureRemovesTemporaryFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "jevlint.json")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Languages: map[string]Language{"go": {}},
		Rules: []Rule{{
			ID:          "one",
			Description: "A rule.",
			Severity:    SeverityError,
		}},
	}
	if err := Write(target, cfg); err == nil {
		t.Fatal("Write() error = nil, want a failure")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".jevlint-config-") {
			t.Fatalf("left temporary file %q", entry.Name())
		}
	}
}
func TestFailProbabilityFloorPrefersRuleWhenSet(t *testing.T) {
	t.Parallel()

	global := 0.8
	ruleZero := 0.0
	cfg := Config{MinFailProbability: &global}

	if got := cfg.FailProbabilityFloor(Rule{}); got != 0.8 {
		t.Fatalf("omitted rule floor = %v, want global 0.8", got)
	}
	if got := cfg.FailProbabilityFloor(Rule{MinFailProbability: &ruleZero}); got != 0 {
		t.Fatalf("rule floor = %v, want 0", got)
	}
	if got := (Config{}).FailProbabilityFloor(Rule{}); got != DefaultMinFailProbability {
		t.Fatalf("default floor = %v, want %v", got, DefaultMinFailProbability)
	}
}

func TestDecodeRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"unknown field": `{
			"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "error",
				"unexpected": true
			}]
		}`,
		"duplicate id": `{
			"rules": [
				{"id": "same", "description": "One.", "severity": "error"},
				{"id": "same", "description": "Two.", "severity": "warning"}
			]
		}`,
		"trailing value": `{
			"rules": [{"id": "one", "description": "A rule.", "severity": "info"}]
		} {}`,
		"invalid localization category": `{
			"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"localize": ["banana"]
			}]
		}`,
		"invalid code unit kind": `{
			"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"kinds": ["banana"]
			}]
		}`,
		"unknown context field": `{
			"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"context": { "graph": true }
			}]
		}`,
	}

	for name, input := range tests {
		name, input := name, input
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Decode(strings.NewReader(withGoLanguage(input))); err == nil {
				t.Fatal("Decode() error = nil, want an error")
			}
		})
	}
}

func TestDecodeValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"minFailProbability below zero": {
			input: `{
				"minFailProbability": -0.1,
				"rules": [{"id": "one", "description": "A rule.", "severity": "info"}]
			}`,
			want: "minFailProbability must be between 0 and 1",
		},
		"minFailProbability above one": {
			input: `{
				"minFailProbability": 1.1,
				"rules": [{"id": "one", "description": "A rule.", "severity": "info"}]
			}`,
			want: "minFailProbability must be between 0 and 1",
		},
		"rule minFailProbability below zero": {
			input: `{
				"rules": [{
					"id": "one",
					"description": "A rule.",
					"severity": "info",
					"minFailProbability": -0.1
				}]
			}`,
			want: "rules[0].minFailProbability must be between 0 and 1",
		},
		"rule minFailProbability above one": {
			input: `{
				"rules": [{
					"id": "one",
					"description": "A rule.",
					"severity": "info",
					"minFailProbability": 1.1
				}]
			}`,
			want: "rules[0].minFailProbability must be between 0 and 1",
		},
		"missing id takes precedence": {
			input: `{"rules": [{
				"id": " ",
				"description": "",
				"severity": "error"
			}]}`,
			want: "rules[0].id is required",
		},
		"duplicate id takes precedence": {
			input: `{"rules": [
				{"id": "same", "description": "Valid.", "severity": "info"},
				{"id": "same", "description": "", "severity": "error"}
			]}`,
			want: `duplicate rule id "same"`,
		},
		"missing description": {
			input: `{"rules": [{
				"id": "one",
				"description": " ",
				"severity": "error"
			}]}`,
			want: "rules[0].description is required",
		},
		"invalid severity": {
			input: `{"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "unknown"
			}]}`,
			want: `decode config: invalid severity "unknown"`,
		},
		"misspelled severity": {
			input: `{"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "erorr"
			}]}`,
			want: `decode config: invalid severity "erorr"`,
		},
		"empty include pattern": {
			input: `{"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"include": [" "]
			}]}`,
			want: "rules[0] contains an empty file pattern",
		},
		"empty exclude pattern": {
			input: `{"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"exclude": [""]
			}]}`,
			want: "rules[0] contains an empty file pattern",
		},
		"invalid localization category": {
			input: `{"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"localize": ["banana"]
			}]}`,
			want: `decode config: invalid kind "banana"`,
		},
		"invalid code unit kind": {
			input: `{"rules": [{
				"id": "one",
				"description": "A rule.",
				"severity": "info",
				"kinds": ["banana"]
			}]}`,
			want: `decode config: invalid kind "banana"`,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := Decode(strings.NewReader(withGoLanguage(test.input)))
			if err == nil || err.Error() != test.want {
				t.Fatalf("Decode() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestDecodeLanguageOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := Decode(strings.NewReader(`{
		"languages": {
			"java": {},
			"cpp": {
				"extensions": [".cpp", ".hpp"],
				"functionQueries": ["(function_definition) @function"],
				"typeQueries": ["(class_specifier) @type"],
				"regions": {
					"comment": ["comment"],
					"field": ["field_declaration"],
					"statement": ["return_statement"]
				}
			}
		},
		"rules": [{
			"id": "one",
			"description": "A rule.",
			"severity": "info"
		}]
	}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(cfg.Languages) != 2 ||
		len(cfg.Languages["cpp"].Extensions) != 2 ||
		cfg.Languages["java"].Extensions != nil {
		t.Fatalf("Decode() languages = %#v", cfg.Languages)
	}
}

func TestValidateRejectsInvalidLanguages(t *testing.T) {
	t.Parallel()

	validRule := []Rule{{
		ID:          "one",
		Description: "A rule.",
		Severity:    SeverityInfo,
	}}
	tests := map[string]struct {
		languages map[string]Language
		want      string
	}{
		"missing languages": {
			want: "config must enable at least one language",
		},
		"unknown preset": {
			languages: map[string]Language{"brainfuck": {}},
			want:      `languages contains unknown preset "brainfuck"`,
		},
		"empty extensions": {
			languages: map[string]Language{"go": {Extensions: []string{}}},
			want:      "languages.go.extensions cannot be empty",
		},
		"malformed extension": {
			languages: map[string]Language{"go": {Extensions: []string{"GO"}}},
			want:      `languages.go.extensions contains invalid extension "GO"`,
		},
		"duplicate extension": {
			languages: map[string]Language{
				"go":   {Extensions: []string{".source"}},
				"rust": {Extensions: []string{".source"}},
			},
			want: `language extension ".source" is assigned to both "go" and "rust"`,
		},
		"empty function queries": {
			languages: map[string]Language{
				"go": {FunctionQueries: []string{}},
			},
			want: "languages.go.functionQueries cannot be empty",
		},
		"blank type query": {
			languages: map[string]Language{
				"go": {TypeQueries: []string{" "}},
			},
			want: "languages.go.typeQueries contains an empty query",
		},
		"invalid region category": {
			languages: map[string]Language{
				"go": {Regions: map[string][]string{"banana": {"node"}}},
			},
			want: `languages.go.regions contains invalid category "banana"`,
		},
		"empty region kinds": {
			languages: map[string]Language{
				"go": {Regions: map[string][]string{"field": {}}},
			},
			want: "languages.go.regions.field cannot be empty",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := (Config{
				Languages: test.languages,
				Rules:     validRule,
			}).Validate()
			if err == nil || err.Error() != test.want {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestDecodePacksWithoutRules(t *testing.T) {
	t.Parallel()

	cfg, err := Decode(strings.NewReader(withGoLanguage(`{
		"packs": [{
			"id": "database-joins",
			"source": "https://github.com/org/repo",
			"sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}]
	}`)))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(cfg.Packs) != 1 || len(cfg.Rules) != 0 {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestFailProbabilityFloorPrefersRule(t *testing.T) {
	t.Parallel()

	global := 0.8
	ruleFloor := 0.5
	cfg := Config{MinFailProbability: &global}
	if cfg.FailProbabilityFloor(Rule{MinFailProbability: &ruleFloor}) != 0.5 {
		t.Fatalf("rule override = %v", cfg.FailProbabilityFloor(Rule{MinFailProbability: &ruleFloor}))
	}
	if cfg.FailProbabilityFloor(Rule{}) != 0.8 {
		t.Fatalf("global floor = %v", cfg.FailProbabilityFloor(Rule{}))
	}
}

func TestDecodeChecksValidation(t *testing.T) {
	t.Parallel()

	_, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "split",
			"description": "A decomposed rule.",
			"severity": "error",
			"checks": {"subject": [{"question": "In scope?"}]}
		}]
	}`)))
	if err == nil || !strings.Contains(err.Error(), "violation must include") {
		t.Fatalf("Decode() error = %v", err)
	}
	cfg, err := Decode(strings.NewReader(withGoLanguage(`{
		"rules": [{
			"id": "split",
			"description": "A decomposed rule.",
			"severity": "error",
			"exceptions": ["A documented exception."],
			"score": {
				"question": "How clearly does source break the rule?",
				"levels": ["compliant", "unclear", "broken"],
				"paraphrases": [{"question": "How obvious is the break?", "levels": ["no", "maybe", "yes"]}]
			},
			"checks": {
				"subject": [{"question": "Is source in scope?", "yes": "In scope.", "no": "Out of scope."}],
				"violation": [{"question": "Does source break it?", "yes": "Broken.", "no": "Sound.", "paraphrases": [{"question": "Is the break visible?", "yes": "Visible.", "no": "Not visible."}]}]
			}
		}]
	}`)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Rules[0].Score == nil || len(cfg.Rules[0].Score.Paraphrases) != 1 {
		t.Fatalf("score = %#v", cfg.Rules[0].Score)
	}
	if cfg.Rules[0].Checks == nil || len(cfg.Rules[0].Checks.Violation) != 1 || len(cfg.Rules[0].Checks.Subject) != 1 {
		t.Fatalf("checks = %#v", cfg.Rules[0].Checks)
	}
}
func withGoLanguage(input string) string {
	return strings.Replace(
		input,
		"{",
		`{"languages":{"go":{}},`,
		1,
	)
}
