package packs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/evals"
)

const (
	ManifestFile     = "pack.json"
	defaultRulesFile = "rules.json"
	currentVersion   = 1
)

type Manifest struct {
	Version   int      `json:"version"`
	ID        string   `json:"id"`
	Languages []string `json:"languages,omitempty"`
	Rules     string   `json:"rules,omitempty"`
	Evals     string   `json:"evals,omitempty"`
	Guidance  string   `json:"guidance,omitempty"`
}

type Loaded struct {
	Ref      config.PackRef
	Manifest Manifest
	Rules    []config.Rule
	Guidance string
	Dir      string
}

func (loaded Loaded) EvalPath() (string, error) {
	name := loaded.Manifest.Evals
	if name == "" {
		name = evals.DefaultFile
	}
	path, err := safeJoin(loaded.Dir, name)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("pack eval file %q is a symbolic link", name)
	}
	return path, nil
}

// CacheDir returns the cache location for one pinned pack. The sha and
// owner/name id come from config or pack metadata, so each path component is
// validated before use.
func CacheDir(userCache string, sha string, id string) (string, error) {
	owner, name, err := SplitPackID(id)
	if err != nil {
		return "", err
	}
	if err := config.ValidatePackSHA(sha); err != nil {
		return "", err
	}
	// Legacy caches were populated without commit verification; do not reuse them.
	packsDir, err := safeJoin(userCache, filepath.Join("jevlint", "packs", "v2"))
	if err != nil {
		return "", err
	}
	bySHA, err := safeJoin(packsDir, sha)
	if err != nil {
		return "", err
	}
	byOwner, err := safeJoin(bySHA, owner)
	if err != nil {
		return "", err
	}
	return safeJoin(byOwner, name)
}

func LoadDir(dir string) (Loaded, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve pack dir: %w", err)
	}
	if err := rejectSymlinks(absolute); err != nil {
		return Loaded{}, err
	}
	manifest, err := loadManifest(filepath.Join(absolute, ManifestFile))
	if err != nil {
		return Loaded{}, err
	}
	rulesPath := manifest.Rules
	if rulesPath == "" {
		rulesPath = defaultRulesFile
	}
	rulesFile, err := safeJoin(absolute, rulesPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("pack %q: %w", manifest.ID, err)
	}
	rules, err := loadRules(rulesFile)
	if err != nil {
		return Loaded{}, fmt.Errorf("pack %q: %w", manifest.ID, err)
	}
	var guidance string
	if manifest.Guidance != "" {
		path, err := safeJoin(absolute, manifest.Guidance)
		if err != nil {
			return Loaded{}, fmt.Errorf("pack %q: %w", manifest.ID, err)
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return Loaded{}, fmt.Errorf("pack %q: read guidance: %w", manifest.ID, err)
		}
		guidance = string(text)
	}
	return Loaded{
		Manifest: manifest,
		Rules:    rules,
		Guidance: guidance,
		Dir:      absolute,
	}, nil
}

func loadManifest(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open pack manifest: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode pack manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errorsIsEOF(err) {
		if err == nil {
			return Manifest{}, fmt.Errorf("decode pack manifest: multiple JSON values")
		}
		return Manifest{}, fmt.Errorf("decode pack manifest: %w", err)
	}
	if manifest.Version != currentVersion {
		return Manifest{}, fmt.Errorf(
			"pack version %d is unsupported (want %d)",
			manifest.Version,
			currentVersion,
		)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return Manifest{}, fmt.Errorf("pack id is required")
	}
	if err := validatePackID(manifest.ID); err != nil {
		return Manifest{}, err
	}
	for _, name := range []string{manifest.Rules, manifest.Evals, manifest.Guidance} {
		if name == "" {
			continue
		}
		if err := validateRelativeName(name); err != nil {
			return Manifest{}, fmt.Errorf("pack manifest: %w", err)
		}
	}
	return manifest, nil
}

func errorsIsEOF(err error) bool {
	return err == io.EOF
}

func loadRules(path string) ([]config.Rule, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pack rules: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document struct {
		Rules []config.Rule `json:"rules"`
	}
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode pack rules: %w", err)
	}
	if len(document.Rules) == 0 {
		return nil, fmt.Errorf("pack rules file has no rules")
	}
	return document.Rules, nil
}

func Merge(project config.Config, loaded []Loaded) (config.Config, error) {
	merged := project
	merged.Rules = make([]config.Rule, 0)
	byID := make(map[string]config.Rule)
	order := make([]string, 0)
	for _, pack := range loaded {
		if err := ensureLanguages(project, pack); err != nil {
			return config.Config{}, err
		}
		if pack.Manifest.ID != pack.Ref.ID && pack.Ref.ID != "" {
			return config.Config{}, fmt.Errorf(
				"pack id %q does not match pin %q",
				pack.Manifest.ID,
				pack.Ref.ID,
			)
		}
		for _, rule := range pack.Rules {
			if _, exists := byID[rule.ID]; exists {
				return config.Config{}, fmt.Errorf(
					"duplicate rule id %q across packs",
					rule.ID,
				)
			}
			if rule.Guidance == "" {
				rule.Guidance = pack.Guidance
			}
			byID[rule.ID] = rule
			order = append(order, rule.ID)
		}
	}
	for _, rule := range project.Rules {
		existing, ok := byID[rule.ID]
		if !ok {
			if !ruleLooksComplete(rule) {
				return config.Config{}, fmt.Errorf("unknown pack rule %q", rule.ID)
			}
			if _, exists := byID[rule.ID]; exists {
				return config.Config{}, fmt.Errorf("duplicate rule id %q", rule.ID)
			}
			byID[rule.ID] = rule
			order = append(order, rule.ID)
			continue
		}
		byID[rule.ID] = overlayRule(existing, rule)
	}
	for _, id := range order {
		merged.Rules = append(merged.Rules, byID[id])
	}
	if err := merged.Validate(); err != nil {
		return config.Config{}, err
	}
	return merged, nil
}

func ruleLooksComplete(rule config.Rule) bool {
	return strings.TrimSpace(rule.Description) != "" &&
		(rule.Severity == config.SeverityInfo ||
			rule.Severity == config.SeverityWarning ||
			rule.Severity == config.SeverityError)
}

func overlayRule(base config.Rule, overlay config.Rule) config.Rule {
	if overlay.Description != "" {
		base.Description = overlay.Description
	}
	if overlay.Severity != config.SeverityUnknown {
		base.Severity = overlay.Severity
	}
	if overlay.Include != nil {
		base.Include = overlay.Include
	}
	if overlay.Exclude != nil {
		base.Exclude = overlay.Exclude
	}
	if overlay.SourceMatch != nil {
		base.SourceMatch = overlay.SourceMatch
	}
	if overlay.Guidance != "" {
		base.Guidance = overlay.Guidance
	}
	if overlay.IncludeTests {
		base.IncludeTests = true
	}
	if overlay.Context.Types {
		base.Context.Types = true
	}
	if overlay.Exceptions != nil {
		base.Exceptions = overlay.Exceptions
	}
	if overlay.Kinds != nil {
		base.Kinds = overlay.Kinds
	}
	if overlay.Localize != nil {
		base.Localize = overlay.Localize
	}
	if overlay.MinConfidence != nil {
		base.MinConfidence = overlay.MinConfidence
	}
	return base
}

func ensureLanguages(project config.Config, pack Loaded) error {
	for _, language := range pack.Manifest.Languages {
		if _, ok := project.Languages[language]; !ok {
			return fmt.Errorf(
				"pack %q requires language %q",
				pack.Manifest.ID,
				language,
			)
		}
	}
	return nil
}
