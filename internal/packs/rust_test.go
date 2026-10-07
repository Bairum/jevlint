package packs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
	"github.com/codegirl-007/jevlint/internal/scoping"
)

var rustPacks = []string{"rust-core", "rust-core-advisory", "rust-performance", "rust-readability"}

func TestRustPacksDeclareGuidanceAndApplyOnlyToRustPaths(t *testing.T) {
	t.Parallel()

	paths := []struct {
		path string
		want bool
	}{
		{"sample.rs", true},
		{"src/nested/sample.rs", true},
		{"sample.go", false},
		{"src/nested/sample.go", false},
		{"sample.py", false},
		{"src/nested/sample.py", false},
		{"sample.ts", false},
		{"src/nested/sample.ts", false},
		{"sample.rs.txt", false},
		{"src/nested/sample.rs.txt", false},
	}
	for _, pack := range rustPacks {
		t.Run(pack, func(t *testing.T) {
			loaded, err := LoadDir(filepath.Join("..", "..", "packs", pack))
			if err != nil {
				t.Fatalf("pack %q: LoadDir() error = %v", pack, err)
			}
			if strings.TrimSpace(loaded.Manifest.Guidance) == "" {
				t.Errorf("pack %q: manifest guidance path is empty", pack)
			}
			if strings.TrimSpace(loaded.Guidance) == "" {
				t.Errorf("pack %q: loaded guidance is empty", pack)
			}
			for _, rule := range loaded.Rules {
				if len(rule.SourceMatch) == 0 {
					t.Errorf("pack %q rule %q: sourceMatch is empty", pack, rule.ID)
				}
				for _, tc := range paths {
					got, err := scoping.Applies(rule, tc.path)
					if err != nil {
						t.Errorf("pack %q rule %q path %q: Applies() error = %v", pack, rule.ID, tc.path, err)
						continue
					}
					if got != tc.want {
						t.Errorf("pack %q rule %q path %q: Applies() = %t, want %t", pack, rule.ID, tc.path, got, tc.want)
					}
				}
			}
		})
	}
}

// Eval fails with no applicable units when every fixture unit is a test or
// misses the rule's sourceMatch, so check that offline for every pack case.
func TestRustPackEvalCasesHaveApplicableUnits(t *testing.T) {
	t.Parallel()

	extractor, err := parsing.NewExtractor(map[string]config.Language{"rust": {}})
	if err != nil {
		t.Fatalf("NewExtractor() error = %v", err)
	}
	for _, pack := range rustPacks {
		t.Run(pack, func(t *testing.T) {
			dir := filepath.Join("..", "..", "packs", pack)
			loaded, err := LoadDir(dir)
			if err != nil {
				t.Fatalf("LoadDir() error = %v", err)
			}
			evalPath, err := loaded.EvalPath()
			if err != nil {
				t.Fatalf("EvalPath() error = %v", err)
			}
			data, err := os.ReadFile(evalPath)
			if err != nil {
				t.Fatalf("read evals: %v", err)
			}
			var document struct {
				Cases []struct {
					Name string `json:"name"`
					Rule string `json:"rule"`
					File string `json:"file"`
				} `json:"cases"`
			}
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatalf("decode evals: %v", err)
			}
			rules := make(map[string]config.Rule, len(loaded.Rules))
			for _, rule := range loaded.Rules {
				rules[rule.ID] = rule
			}
			for _, evalCase := range document.Cases {
				rule, ok := rules[evalCase.Rule]
				if !ok {
					t.Errorf("case %q: unknown rule %q", evalCase.Name, evalCase.Rule)
					continue
				}
				path := filepath.Join(filepath.Dir(evalPath), evalCase.File)
				source, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("case %q: %v", evalCase.Name, err)
					continue
				}
				file, err := extractor.ExtractFile(path, source)
				if err != nil {
					t.Errorf("case %q: extract: %v", evalCase.Name, err)
					continue
				}
				if !hasApplicableUnit(rule, file.Units) {
					t.Errorf("case %q: no non-test %v unit matches sourceMatch of %q", evalCase.Name, rule.Kinds, rule.ID)
				}
			}
		})
	}
}

func hasApplicableUnit(rule config.Rule, units []parsing.CodeUnit) bool {
	for _, unit := range units {
		if unit.Test && !rule.IncludeTests {
			continue
		}
		kindOK := len(rule.Kinds) == 0 && (unit.Kind == parsing.CodeKindFunction || unit.Kind == parsing.CodeKindType)
		for _, kind := range rule.Kinds {
			if kind.String() == unit.Kind.String() {
				kindOK = true
			}
		}
		if !kindOK {
			continue
		}
		if len(rule.SourceMatch) == 0 {
			return true
		}
		for _, pattern := range rule.SourceMatch {
			if regexp.MustCompile(pattern).MatchString(unit.Source) {
				return true
			}
		}
	}
	return false
}
