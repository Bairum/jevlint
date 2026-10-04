package packs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestMergeOverlaysPackRule(t *testing.T) {
	t.Parallel()

	floor := 0.5
	project := config.Config{
		Languages:     map[string]config.Language{"go": {}},
		MinConfidence: ptr(0.8),
		Packs: []config.PackRef{{
			ID:     "database-joins",
			Source: "local",
			SHA:    strings.Repeat("a", 40),
		}},
		Rules: []config.Rule{{
			ID:            "database-joins",
			MinConfidence: &floor,
			Include:       []string{"src/**/*.go"},
		}},
	}
	loaded := []Loaded{{
		Ref:      project.Packs[0],
		Manifest: Manifest{ID: "database-joins", Languages: []string{"go"}},
		Rules: []config.Rule{{
			ID:          "database-joins",
			Description: "Join in the database.",
			Severity:    config.SeverityError,
			Kinds:       []config.TargetKind{config.TargetKindFunction},
		}},
	}}
	merged, err := Merge(project, loaded)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if len(merged.Rules) != 1 {
		t.Fatalf("rules = %#v", merged.Rules)
	}
	rule := merged.Rules[0]
	if rule.Description != "Join in the database." {
		t.Fatalf("description = %q", rule.Description)
	}
	if rule.MinConfidence == nil || *rule.MinConfidence != 0.5 {
		t.Fatalf("minConfidence = %#v", rule.MinConfidence)
	}
	if len(rule.Include) != 1 || rule.Include[0] != "src/**/*.go" {
		t.Fatalf("include = %#v", rule.Include)
	}
	if merged.ConfidenceFloor(rule) != 0.5 {
		t.Fatalf("ConfidenceFloor() = %v", merged.ConfidenceFloor(rule))
	}
}

func TestMergePackOnlyRules(t *testing.T) {
	t.Parallel()

	project := config.Config{
		Languages: map[string]config.Language{"go": {}},
		Packs:     []config.PackRef{{ID: "database-joins", Source: "local", SHA: strings.Repeat("a", 40)}},
	}
	loaded := []Loaded{{
		Ref:      project.Packs[0],
		Manifest: Manifest{ID: "database-joins"},
		Rules: []config.Rule{{
			ID:          "database-joins",
			Description: "Join in the database.",
			Severity:    config.SeverityError,
		}},
	}}
	merged, err := Merge(project, loaded)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if len(merged.Rules) != 1 {
		t.Fatalf("rules = %#v", merged.Rules)
	}
}

func TestMergeRejectsUnknownLanguage(t *testing.T) {
	t.Parallel()

	project := config.Config{
		Languages: map[string]config.Language{"go": {}},
		Packs:     []config.PackRef{{ID: "joins", Source: "local", SHA: "abc"}},
	}
	_, err := Merge(project, []Loaded{{
		Ref:      project.Packs[0],
		Manifest: Manifest{ID: "joins", Languages: []string{"rust"}},
		Rules: []config.Rule{{
			ID:          "joins",
			Description: "Join in the database.",
			Severity:    config.SeverityError,
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "requires language") {
		t.Fatalf("Merge() error = %v", err)
	}
}

func TestMergeRejectsDuplicateRuleIDs(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		ID:          "joins",
		Description: "Join in the database.",
		Severity:    config.SeverityError,
	}
	project := config.Config{
		Languages: map[string]config.Language{"go": {}},
		Packs: []config.PackRef{
			{ID: "one", Source: "local", SHA: "a"},
			{ID: "two", Source: "local", SHA: "b"},
		},
	}
	_, err := Merge(project, []Loaded{
		{Ref: project.Packs[0], Manifest: Manifest{ID: "one"}, Rules: []config.Rule{rule}},
		{Ref: project.Packs[1], Manifest: Manifest{ID: "two"}, Rules: []config.Rule{rule}},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate rule id") {
		t.Fatalf("Merge() error = %v", err)
	}
}

func TestMergeRejectsUnknownOverlay(t *testing.T) {
	t.Parallel()

	project := config.Config{
		Languages: map[string]config.Language{"go": {}},
		Packs:     []config.PackRef{{ID: "joins", Source: "local", SHA: "abc"}},
		Rules:     []config.Rule{{ID: "missing", Include: []string{"**/*.go"}}},
	}
	_, err := Merge(project, []Loaded{{
		Ref:      project.Packs[0],
		Manifest: Manifest{ID: "joins"},
		Rules: []config.Rule{{
			ID:          "joins",
			Description: "Join in the database.",
			Severity:    config.SeverityError,
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "unknown pack rule") {
		t.Fatalf("Merge() error = %v", err)
	}
}

func TestLoadDirReadsPack(t *testing.T) {
	t.Parallel()

	dir := writePackDir(t)
	loaded, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v", err)
	}
	if loaded.Manifest.ID != "codegirl-007/database-joins" || len(loaded.Rules) != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestParseSpecGitHubTree(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec("https://github.com/org/repo/tree/main/packs/joins")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Source != "https://github.com/org/repo.git" ||
		spec.Ref != "main" ||
		spec.Path != "packs/joins" {
		t.Fatalf("spec = %#v", spec)
	}
}

func writePackDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{
		"version": 1,
		"id": "codegirl-007/database-joins",
		"languages": ["go"]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, defaultRulesFile), []byte(`{
		"rules": [{
			"id": "database-joins",
			"description": "Join in the database.",
			"severity": "error",
			"kinds": ["function"]
		}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func ptr(value float64) *float64 {
	return &value
}
