package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	// DefaultMinFailProbability is the reporting threshold when neither the
	// rule nor the config sets one.
	DefaultMinFailProbability = 0.5
	// MinConfidenceReplaced is the decode error for the removed field.
	MinConfidenceReplaced = `minConfidence was replaced by minFailProbability; see README "Reporting threshold"`
)

// Config is the rule file loaded from disk.
type Config struct {
	Languages          map[string]Language `json:"languages"`
	MinFailProbability *float64            `json:"minFailProbability,omitempty"`
	Packs              []PackRef           `json:"packs,omitempty"`
	Rules              []Rule              `json:"rules,omitempty"`
}

type PackRef struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Ref    string `json:"ref,omitempty"`
	SHA    string `json:"sha"`
}

// Language customizes a built in language preset.
type Language struct {
	Extensions         []string            `json:"extensions,omitempty"`
	FunctionQueries    []string            `json:"functionQueries,omitempty"`
	TypeQueries        []string            `json:"typeQueries,omitempty"`
	TypeContextQueries []string            `json:"typeContextQueries,omitempty"`
	Regions            map[string][]string `json:"regions,omitempty"`
}

// Check is one yes/no component of a decomposed rule.
type Check struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Yes      string `json:"yes,omitempty"`
	No       string `json:"no,omitempty"`
	FailWhen bool   `json:"failWhen"`
}

// Rule describes one check, the code it covers, and how it is reported.
type Rule struct {
	ID                 string       `json:"id"`
	Description        string       `json:"description"`
	Severity           Severity     `json:"severity,omitempty"`
	Include            []string     `json:"include,omitempty"`
	Exclude            []string     `json:"exclude,omitempty"`
	SourceMatch        []string     `json:"sourceMatch,omitempty"`
	Guidance           string       `json:"guidance,omitempty"`
	IncludeTests       bool         `json:"includeTests,omitempty"`
	Exceptions         []string     `json:"exceptions,omitempty"`
	Kinds              []TargetKind `json:"kinds,omitempty"`
	Localize           []TargetKind `json:"localize,omitempty"`
	MinFailProbability *float64     `json:"minFailProbability,omitempty"`
	AllowSkip          bool         `json:"allowSkip,omitempty"`
	AllowAbstain       bool         `json:"allowAbstain,omitempty"`
	Context            RuleContext  `json:"context,omitempty"`
	Checks             []Check      `json:"checks,omitempty"`
}

// RuleContext asks for extra material to send with a rule.
type RuleContext struct {
	Callees bool `json:"callees,omitempty"`
	Types   bool `json:"types,omitempty"`
}

// TargetKind names the kind of code a rule can check.
type TargetKind int

const (
	TargetKindUnknown TargetKind = iota
	TargetKindComment
	TargetKindDocComment
	TargetKindField
	TargetKindFunction
	TargetKindStatement
	TargetKindType
)

// Severity says how serious a finding is.
type Severity int

const (
	SeverityUnknown Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
)

// String returns the kind name.
func (kind TargetKind) String() string {
	switch kind {
	case TargetKindComment:
		return KindComment
	case TargetKindDocComment:
		return KindDocComment
	case TargetKindField:
		return KindField
	case TargetKindFunction:
		return KindFunction
	case TargetKindStatement:
		return KindStatement
	case TargetKindType:
		return KindType
	default:
		return "unknown"
	}
}

// MarshalJSON writes the kind as its name.
func (kind TargetKind) MarshalJSON() ([]byte, error) {
	if kind == TargetKindUnknown {
		return nil, fmt.Errorf("invalid kind %d", kind)
	}
	return json.Marshal(kind.String())
}

// UnmarshalJSON reads a kind from its name.
func (kind *TargetKind) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseTargetKind(value)
	if err != nil {
		return err
	}
	*kind = parsed
	return nil
}

// ParseTargetKind turns a kind name into a kind value. An unknown name is an
// error rather than TargetKindUnknown, which means "not specified".
func ParseTargetKind(value string) (TargetKind, error) {
	switch value {
	case KindComment:
		return TargetKindComment, nil
	case KindDocComment:
		return TargetKindDocComment, nil
	case KindField:
		return TargetKindField, nil
	case KindFunction:
		return TargetKindFunction, nil
	case KindStatement:
		return TargetKindStatement, nil
	case KindType:
		return TargetKindType, nil
	default:
		return TargetKindUnknown, fmt.Errorf("invalid kind %q", value)
	}
}

// String returns the severity name.
func (severity Severity) String() string {
	switch severity {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON writes the severity as its name.
func (severity Severity) MarshalJSON() ([]byte, error) {
	if severity == SeverityUnknown {
		return nil, fmt.Errorf("invalid severity %d", severity)
	}
	return json.Marshal(severity.String())
}

// UnmarshalJSON reads a severity from its name. Invalid names fail loudly
// instead of becoming SeverityUnknown, which would look like "not specified"
// and let a typo pass silently in a partial pack overlay.
func (severity *Severity) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseSeverity(value)
	if err != nil {
		return err
	}
	*severity = parsed
	return nil
}

// ParseSeverity turns a severity name into a severity value. An unknown name is
// an error rather than SeverityUnknown, which means "not specified".
func ParseSeverity(value string) (Severity, error) {
	switch value {
	case "info":
		return SeverityInfo, nil
	case "warning":
		return SeverityWarning, nil
	case "error":
		return SeverityError, nil
	default:
		return SeverityUnknown, fmt.Errorf("invalid severity %q", value)
	}
}

var languagePresets = map[string]struct{}{
	"c":          {},
	"cpp":        {},
	"csharp":     {},
	"go":         {},
	"java":       {},
	"javascript": {},
	"kotlin":     {},
	"php":        {},
	"python":     {},
	"ruby":       {},
	"rust":       {},
	"tsx":        {},
	"typescript": {},
}

// Load reads and checks the rule file at a path.
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	return Decode(file)
}

// Decode reads and checks the rule file from a reader.
func Decode(reader io.Reader) (Config, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := RejectMinConfidence(data); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, errors.New("decode config: multiple JSON values")
		}
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// RejectMinConfidence reports the removed field anywhere in a config or pack.
func RejectMinConfidence(data []byte) error {
	var probe any
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil
	}
	if hasMinConfidence(probe) {
		return errors.New(MinConfidenceReplaced)
	}
	return nil
}

func hasMinConfidence(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		if _, ok := typed["minConfidence"]; ok {
			return true
		}
		for _, child := range typed {
			if hasMinConfidence(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if hasMinConfidence(child) {
				return true
			}
		}
	}
	return false
}

// FailProbabilityFloor returns the reporting threshold for a rule.
func (cfg Config) FailProbabilityFloor(rule Rule) float64 {
	if rule.MinFailProbability != nil {
		return *rule.MinFailProbability
	}
	if cfg.MinFailProbability != nil {
		return *cfg.MinFailProbability
	}
	return DefaultMinFailProbability
}

func Write(path string, cfg Config) error {
	data, err := encodeConfig(cfg)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(path, data); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// encodeConfig renders the config as formatted JSON.
func encodeConfig(cfg Config) ([]byte, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return append(data, '\n'), nil
}

// writeFileAtomic writes data to path by renaming a temporary file over it, so
// an interrupted or failed write never truncates the existing config.
func writeFileAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	temporary, err := os.CreateTemp(directory, ".jevlint-config-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	succeeded := false
	defer func() {
		_ = temporary.Close()
		if !succeeded {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	succeeded = true
	return nil
}

// Validate checks the config for mistakes.
func (cfg Config) Validate() error {
	if err := validateLanguages(cfg.Languages); err != nil {
		return err
	}
	if err := validateFailProbability(cfg.MinFailProbability); err != nil {
		return err
	}
	if err := validatePackRefs(cfg.Packs); err != nil {
		return err
	}
	ids := make(map[string]struct{}, len(cfg.Rules))
	for index, rule := range cfg.Rules {
		prefix := fmt.Sprintf("rules[%d]", index)
		if err := validateRuleID(rule, prefix); err != nil {
			return err
		}
		if _, exists := ids[rule.ID]; exists {
			return fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		ids[rule.ID] = struct{}{}
		if err := validateFailProbability(rule.MinFailProbability); err != nil {
			return fmt.Errorf("%s.minFailProbability must be between 0 and 1", prefix)
		}
		if err := ValidateChecks(rule, prefix); err != nil {
			return err
		}
		if len(cfg.Packs) > 0 && !ruleLooksComplete(rule) {
			if err := validateRulePatterns(rule, prefix); err != nil {
				return err
			}
			if err := validateRuleLocalization(rule, prefix); err != nil {
				return err
			}
			if err := validateRuleKinds(rule, prefix); err != nil {
				return err
			}
			continue
		}
		if err := validateRuleDescription(rule, prefix); err != nil {
			return err
		}
		if err := validateRuleSeverity(rule, prefix); err != nil {
			return err
		}
		if err := validateRulePatterns(rule, prefix); err != nil {
			return err
		}
		if err := validateRuleLocalization(rule, prefix); err != nil {
			return err
		}
		if err := validateRuleKinds(rule, prefix); err != nil {
			return err
		}
	}
	return nil
}

func ruleLooksComplete(rule Rule) bool {
	return strings.TrimSpace(rule.Description) != "" &&
		(rule.Severity == SeverityInfo ||
			rule.Severity == SeverityWarning ||
			rule.Severity == SeverityError)
}

func validatePackRefs(refs []PackRef) error {
	seen := make(map[string]struct{}, len(refs))
	for index, ref := range refs {
		prefix := fmt.Sprintf("packs[%d]", index)
		if strings.TrimSpace(ref.ID) == "" {
			return fmt.Errorf("%s.id is required", prefix)
		}
		if strings.TrimSpace(ref.Source) == "" {
			return fmt.Errorf("%s.source is required", prefix)
		}
		if strings.TrimSpace(ref.SHA) == "" {
			return fmt.Errorf("%s.sha is required", prefix)
		}
		if err := ValidatePackSHA(ref.SHA); err != nil {
			return fmt.Errorf("%s.sha: %w", prefix, err)
		}
		if _, exists := seen[ref.ID]; exists {
			return fmt.Errorf("duplicate pack id %q", ref.ID)
		}
		seen[ref.ID] = struct{}{}
	}
	return nil
}

// ValidatePackSHA accepts full hexadecimal Git SHA-1 or SHA-256 object IDs.
func ValidatePackSHA(sha string) error {
	if len(sha) != 40 && len(sha) != 64 {
		return fmt.Errorf("invalid pack sha %q; want a full Git commit ID", sha)
	}
	for _, c := range sha {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return fmt.Errorf("invalid pack sha %q; want a full Git commit ID", sha)
		}
	}
	return nil
}

// validateFailProbability checks that a reporting threshold is between zero and one.
func validateFailProbability(value *float64) error {
	if value == nil {
		return nil
	}
	if *value < 0 || *value > 1 {
		return errors.New("minFailProbability must be between 0 and 1")
	}
	return nil
}

var checkID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,40}$`)

// ValidateChecks checks a decomposed rule. A rule without checks is unchanged.
func ValidateChecks(rule Rule, prefix string) error {
	if len(rule.Checks) == 0 {
		return nil
	}
	if len(rule.Exceptions) > 0 || rule.AllowSkip || rule.AllowAbstain {
		return fmt.Errorf("%s with checks must not set exceptions, allowSkip, or allowAbstain", prefix)
	}
	seen := make(map[string]struct{}, len(rule.Checks))
	failing := false
	for index, check := range rule.Checks {
		checkPrefix := fmt.Sprintf("%s.checks[%d]", prefix, index)
		if !checkID.MatchString(check.ID) {
			return fmt.Errorf("%s.id must match %s", checkPrefix, checkID.String())
		}
		if _, exists := seen[check.ID]; exists {
			return fmt.Errorf("%s.id %q is duplicated", checkPrefix, check.ID)
		}
		seen[check.ID] = struct{}{}
		if strings.TrimSpace(check.Question) == "" {
			return fmt.Errorf("%s.question is required", checkPrefix)
		}
		if check.FailWhen {
			failing = true
		}
	}
	if !failing {
		return fmt.Errorf("%s.checks must include at least one failWhen check", prefix)
	}
	return nil
}

// validateLanguages checks the enabled languages and their settings.
func validateLanguages(languages map[string]Language) error {
	if len(languages) == 0 {
		return errors.New("config must enable at least one language")
	}

	ids := make([]string, 0, len(languages))
	for id := range languages {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	extensions := make(map[string]string)
	for _, id := range ids {
		language := languages[id]
		if _, ok := languagePresets[id]; !ok {
			return fmt.Errorf("languages contains unknown preset %q", id)
		}
		if err := validateLanguageExtensions(id, language.Extensions, extensions); err != nil {
			return err
		}
		if err := validateQueryOverride(id, "functionQueries", language.FunctionQueries); err != nil {
			return err
		}
		if err := validateQueryOverride(id, "typeQueries", language.TypeQueries); err != nil {
			return err
		}
		if err := validateQueryOverride(
			id,
			"typeContextQueries",
			language.TypeContextQueries,
		); err != nil {
			return err
		}
		if err := validateRegionOverrides(id, language.Regions); err != nil {
			return err
		}
	}
	return nil
}

// validateLanguageExtensions checks the file extensions for one language.
func validateLanguageExtensions(
	id string,
	values []string,
	seen map[string]string,
) error {
	if values != nil && len(values) == 0 {
		return fmt.Errorf("languages.%s.extensions cannot be empty", id)
	}
	for _, extension := range values {
		if extension == "" ||
			!strings.HasPrefix(extension, ".") ||
			extension != strings.ToLower(extension) ||
			strings.ContainsAny(extension, `/\`) {
			return fmt.Errorf(
				"languages.%s.extensions contains invalid extension %q",
				id,
				extension,
			)
		}
		if owner, exists := seen[extension]; exists {
			return fmt.Errorf(
				"language extension %q is assigned to both %q and %q",
				extension,
				owner,
				id,
			)
		}
		seen[extension] = id
	}
	return nil
}

// validateQueryOverride checks the custom parser queries for one language.
func validateQueryOverride(id string, field string, values []string) error {
	if values == nil {
		return nil
	}
	if len(values) == 0 {
		return fmt.Errorf("languages.%s.%s cannot be empty", id, field)
	}
	for _, query := range values {
		if strings.TrimSpace(query) == "" {
			return fmt.Errorf("languages.%s.%s contains an empty query", id, field)
		}
	}
	return nil
}

// validateRegionOverrides checks the custom region settings for one language.
func validateRegionOverrides(id string, regions map[string][]string) error {
	if regions == nil {
		return nil
	}
	for category, kinds := range regions {
		switch category {
		case KindComment, KindDocComment, KindField, KindStatement:
		default:
			return fmt.Errorf(
				"languages.%s.regions contains invalid category %q",
				id,
				category,
			)
		}
		if len(kinds) == 0 {
			return fmt.Errorf(
				"languages.%s.regions.%s cannot be empty",
				id,
				category,
			)
		}
		for _, kind := range kinds {
			if strings.TrimSpace(kind) == "" {
				return fmt.Errorf(
					"languages.%s.regions.%s contains an empty node kind",
					id,
					category,
				)
			}
		}
	}
	return nil
}

// validateRuleID checks that a rule has an id.
func validateRuleID(rule Rule, prefix string) error {
	if strings.TrimSpace(rule.ID) == "" {
		return fmt.Errorf("%s.id is required", prefix)
	}
	return nil
}

// validateRuleDescription checks that a rule has a description.
func validateRuleDescription(rule Rule, prefix string) error {
	if strings.TrimSpace(rule.Description) == "" {
		return fmt.Errorf("%s.description is required", prefix)
	}
	return nil
}

// validateRuleSeverity checks that a rule has a known severity.
func validateRuleSeverity(rule Rule, prefix string) error {
	switch rule.Severity {
	case SeverityInfo, SeverityWarning, SeverityError:
		return nil
	default:
		return fmt.Errorf("%s.severity must be info, warning, or error", prefix)
	}
}

// validateRulePatterns checks that the rule file patterns are not empty.
func validateRulePatterns(rule Rule, prefix string) error {
	patterns := append(append([]string{}, rule.Include...), rule.Exclude...)
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("%s contains an empty file pattern", prefix)
		}
	}
	for index, pattern := range rule.SourceMatch {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("%s.sourceMatch[%d] must not be empty", prefix, index)
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("%s.sourceMatch[%d]: %w", prefix, index, err)
		}
	}
	return nil
}

const (
	KindComment    = "comment"
	KindDocComment = "docComment"
	KindField      = "field"
	KindFunction   = "function"
	KindStatement  = "statement"
	KindType       = "type"
)

// validateRuleLocalization checks the places a rule can point at.
func validateRuleLocalization(rule Rule, prefix string) error {
	for _, category := range rule.Localize {
		switch category {
		case TargetKindComment, TargetKindDocComment, TargetKindField, TargetKindStatement:
		default:
			return fmt.Errorf(
				"%s.localize contains invalid category %q; want comment, docComment, field, or statement",
				prefix,
				category,
			)
		}
	}
	return nil
}

// validateRuleKinds checks the code kinds a rule can check.
func validateRuleKinds(rule Rule, prefix string) error {
	for _, kind := range rule.Kinds {
		if kind == TargetKindUnknown {
			return fmt.Errorf(
				"%s.kinds contains invalid kind %q; "+
					"want comment, field, function, statement, or type",
				prefix,
				kind,
			)
		}
	}
	return nil
}
