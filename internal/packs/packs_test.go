package packs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
)

func TestMergeOverlaysPackRule(t *testing.T) {
	t.Parallel()

	floor := 0.5
	project := config.Config{
		Languages:          map[string]config.Language{"go": {}},
		MinFailProbability: ptr(0.8),
		Packs: []config.PackRef{{
			ID:     "database-joins",
			Source: "local",
			SHA:    strings.Repeat("a", 40),
		}},
		Rules: []config.Rule{{
			ID:                 "database-joins",
			MinFailProbability: &floor,
			Include:            []string{"src/**/*.go"},
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
	if rule.MinFailProbability == nil || *rule.MinFailProbability != 0.5 {
		t.Fatalf("minFailProbability = %#v", rule.MinFailProbability)
	}
	if len(rule.Include) != 1 || rule.Include[0] != "src/**/*.go" {
		t.Fatalf("include = %#v", rule.Include)
	}
	if merged.FailProbabilityFloor(rule) != 0.5 {
		t.Fatalf("FailProbabilityFloor() = %v", merged.FailProbabilityFloor(rule))
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

func TestPackGuidanceAndProjectOverlays(t *testing.T) {
	t.Parallel()

	dir := writePackDir(t)
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{
		"version": 1, "id": "codegirl-007/database-joins", "guidance": "guidance.txt"
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "guidance.txt"), []byte("Pack guidance.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Guidance != "Pack guidance.\n" {
		t.Fatalf("loaded guidance = %q", loaded.Guidance)
	}
	loaded.Rules[0].SourceMatch = []string{"old"}
	base := loaded.Rules[0]
	base.ID = "inherited"
	loaded.Rules = append(loaded.Rules, base)
	base.ID = "explicit"
	base.Guidance = "Rule guidance."
	loaded.Rules = append(loaded.Rules, base)
	project := config.Config{
		Languages: map[string]config.Language{"go": {}},
		Rules: []config.Rule{{
			ID: "database-joins", Guidance: "Project guidance.",
			SourceMatch: []string{"new"}, IncludeTests: true,
			Context: config.RuleContext{Types: true},
		}},
	}
	merged, err := Merge(project, []Loaded{loaded})
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{"Project guidance.", "Pack guidance.\n", "Rule guidance."} {
		if merged.Rules[index].Guidance != want {
			t.Fatalf("rule %d guidance = %q, want %q", index, merged.Rules[index].Guidance, want)
		}
	}
	rule := merged.Rules[0]
	if len(rule.SourceMatch) != 1 || rule.SourceMatch[0] != "new" ||
		!rule.IncludeTests || !rule.Context.Types {
		t.Fatalf("overlay rule = %#v", rule)
	}
	project.Rules[0].SourceMatch = []string{}
	project.Rules[0].Guidance = ""
	merged, err = Merge(project, []Loaded{loaded})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Rules[0].SourceMatch) != 0 || merged.Rules[0].Guidance != loaded.Guidance {
		t.Fatalf("cleared filter / inherited guidance = %#v", merged.Rules[0])
	}
}

func TestLoadDirRejectsInvalidGuidance(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"../outside.txt", "/outside.txt", "missing.txt"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := writePackDir(t)
			manifest := `{"version":1,"id":"org/rules","guidance":"` + name + `"}`
			if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadDir(dir); err == nil {
				t.Fatal("LoadDir accepted invalid guidance")
			}
		})
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

func TestCloneSpecResolvesSlashRefsInTreeURLs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack install tests")
	}
	t.Parallel()

	repo := writePackRepo(t,
		`{"version": 1, "id": "codegirl-007/joins", "languages": ["go"]}`,
		`{"rules": [{"id": "joins", "description": "Join in the database.", "severity": "error", "kinds": ["function"]}]}`,
	)
	packsRunGit(t, repo, "checkout", "-q", "-b", "feat/rust-packs")
	if err := os.MkdirAll(filepath.Join(repo, "packs", "core"), 0o700); err != nil {
		t.Fatal(err)
	}
	packsRunGit(t, repo, "mv", ManifestFile, defaultRulesFile, "packs/core/")
	packsRunGit(t, repo, "commit", "-q", "-m", "move pack")
	packsRunGit(t, repo, "tag", "v1/stable")
	packsRunGit(t, repo, "checkout", "-q", "main")

	tests := []struct {
		tail     string
		wantRef  string
		wantPath string
	}{
		{"feat/rust-packs/packs/core", "feat/rust-packs", "packs/core"},
		{"v1/stable/packs/core", "v1/stable", "packs/core"},
		{"main", "main", ""},
	}
	for _, test := range tests {
		t.Run(test.tail, func(t *testing.T) {
			t.Parallel()
			checkout, err := cloneSpec(Spec{Source: repo, treeSegments: strings.Split(test.tail, "/")}, "")
			if err != nil {
				t.Fatalf("cloneSpec() error = %v", err)
			}
			defer os.RemoveAll(filepath.Dir(checkout.dir))
			if checkout.spec.Ref != test.wantRef || checkout.spec.Path != test.wantPath {
				t.Fatalf("ref, path = %q, %q; want %q, %q", checkout.spec.Ref, checkout.spec.Path, test.wantRef, test.wantPath)
			}
			if _, err := LoadDir(filepath.Join(checkout.dir, filepath.FromSlash(test.wantPath))); err != nil {
				t.Fatalf("pack not at resolved path: %v", err)
			}
		})
	}

	if _, err := cloneSpec(Spec{Source: repo, treeSegments: []string{"missing", "packs", "core"}}, ""); err == nil ||
		!strings.Contains(err.Error(), "no branch, tag or commit") {
		t.Fatalf("cloneSpec() error = %v, want an unknown-ref error", err)
	}
}

// A branch and a tag may share a name and point to different commits. Both
// tree URLs and stored refs (used by plugin update) must install the commit
// that resolution chose, not whatever git checkout's own lookup prefers.
func TestCloneSpecInstallsResolvedCommitWhenBranchAndTagCollide(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack install tests")
	}
	t.Parallel()

	repo := writePackRepo(t,
		`{"version": 1, "id": "codegirl-007/joins", "languages": ["go"]}`,
		`{"rules": [{"id": "joins", "description": "Join in the database.", "severity": "error", "kinds": ["function"]}]}`,
	)
	packsRunGit(t, repo, "tag", "release/next")
	packsRunGit(t, repo, "checkout", "-q", "-b", "release/next")
	packsRunGit(t, repo, "commit", "-q", "--allow-empty", "-m", "branch moves past the tag")
	branch, err := gitOutput(repo, "rev-parse", "refs/heads/release/next")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := gitOutput(repo, "rev-parse", "refs/tags/release/next^{commit}")
	if err != nil || tag == branch {
		t.Fatalf("tag = %q, branch = %q, err = %v; want distinct commits", tag, branch, err)
	}
	packsRunGit(t, repo, "checkout", "-q", "main")

	for name, spec := range map[string]Spec{
		"tree url":   {Source: repo, treeSegments: []string{"release", "next"}},
		"stored ref": {Source: repo, Ref: "release/next"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			checkout, err := cloneSpec(spec, "")
			if err != nil {
				t.Fatalf("cloneSpec() error = %v", err)
			}
			defer os.RemoveAll(filepath.Dir(checkout.dir))
			if checkout.sha != branch || checkout.spec.Ref != "release/next" {
				t.Fatalf("sha, ref = %s, %q; want branch commit %s, %q (tag commit is %s)", checkout.sha, checkout.spec.Ref, branch, "release/next", tag)
			}
		})
	}
}

// Cache rehydration must install the pinned object itself. A publisher can
// create a branch or tag named exactly like the pinned SHA that points at
// other content; it must never replace the pinned tree.
func TestFetchRefIgnoresRefsNamedLikeThePinnedCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for pack install tests")
	}
	t.Parallel()

	const manifest = `{"version": 1, "id": "codegirl-007/joins", "languages": ["go"]}`
	rules := func(description string) string {
		return `{"rules": [{"id": "joins", "description": "` + description + `", "severity": "error", "kinds": ["function"]}]}`
	}
	for _, kind := range []string{"branch", "tag"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			repo := writePackRepo(t, manifest, rules("Pinned rules."))
			pinned, err := gitOutput(repo, "rev-parse", "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, defaultRulesFile), []byte(rules("Replacement rules.")), 0o600); err != nil {
				t.Fatal(err)
			}
			packsRunGit(t, repo, "commit", "-q", "-am", "replacement")
			packsRunGit(t, repo, kind, pinned, "HEAD")

			dest := filepath.Join(t.TempDir(), "pack")
			ref := config.PackRef{ID: "codegirl-007/joins", Source: repo, SHA: pinned}
			if err := fetchRef(ref, dest); err != nil {
				t.Fatalf("fetchRef() error = %v", err)
			}
			data, err := os.ReadFile(filepath.Join(dest, defaultRulesFile))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "Pinned rules.") {
				t.Fatalf("rehydrated rules = %s, want the pinned commit's content", data)
			}
		})
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
