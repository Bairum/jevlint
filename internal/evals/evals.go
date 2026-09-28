package evals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"jevlint/internal/config"
	"jevlint/internal/parsing"
)

const (
	DefaultFile    = "jevlint-evals.json"
	currentVersion = 1
	expectPassName = "pass"
	expectFailName = "fail"

	// DefaultExamplesDir is the directory next to the config that holds one
	// fixture directory per rule.
	DefaultExamplesDir = "examples"
	examplesRulesDir   = "rules"
	examplesGoodDir    = "good"
	examplesBadDir     = "bad"
)

// Expect is the outcome an eval case wants to see.
type Expect int

const (
	ExpectUnknown Expect = iota
	ExpectPass
	ExpectFail
)

// Case is one rule and fixture pairing to score.
type Case struct {
	Name   string `json:"name,omitempty"`
	Rule   string `json:"rule"`
	File   string `json:"file"`
	Expect Expect `json:"expect"`
	abs    string
}

// Document is the set of eval cases loaded from a file.
type Document struct {
	Version int    `json:"version"`
	Cases   []Case `json:"cases"`
}

// caseIdentity is the rule and file pair used to spot duplicates.
type caseIdentity struct {
	rule string
	file string
}

// String returns the name of the expected outcome.
func (expect Expect) String() string {
	switch expect {
	case ExpectPass:
		return expectPassName
	case ExpectFail:
		return expectFailName
	default:
		return "unknown"
	}
}

// parseExpect turns an outcome name into an outcome value.
func parseExpect(value string) (Expect, bool) {
	switch value {
	case expectPassName:
		return ExpectPass, true
	case expectFailName:
		return ExpectFail, true
	default:
		return ExpectUnknown, false
	}
}

// UnmarshalJSON reads an outcome from its name.
func (expect *Expect) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, ok := parseExpect(value)
	if !ok {
		return fmt.Errorf("expect must be %s or %s", expectPassName, expectFailName)
	}
	*expect = parsed
	return nil
}

// MarshalJSON writes the outcome as its name.
func (expect Expect) MarshalJSON() ([]byte, error) {
	return json.Marshal(expect.String())
}

// AbsolutePath returns the full path of the case fixture.
func (evalCase Case) AbsolutePath() string {
	return evalCase.abs
}

// Load reads the eval file and checks each case.
func Load(
	path string,
	cfg config.Config,
	extractor *parsing.Extractor,
) (Document, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Document{}, fmt.Errorf("resolve evals path: %w", err)
	}
	file, err := os.Open(absolute)
	if err != nil {
		return Document{}, fmt.Errorf("open evals %q: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("decode evals %q: %w", path, err)
	}
	if document.Version != currentVersion {
		return Document{}, fmt.Errorf(
			"evals version %d is unsupported (want %d)",
			document.Version,
			currentVersion,
		)
	}
	if len(document.Cases) == 0 {
		return Document{}, fmt.Errorf("evals %q has no cases", path)
	}
	root := filepath.Dir(absolute)
	seen := make(map[caseIdentity]struct{}, len(document.Cases))
	for index := range document.Cases {
		if err := prepareCase(&document.Cases[index], root, cfg, extractor, seen); err != nil {
			return Document{}, err
		}
	}
	return document, nil
}

// LoadExamples builds a document from an examples directory. The directory
// holds one subdirectory per rule under "rules", and each rule directory has
// a "good" and a "bad" directory. Files under "good" must pass the rule and
// files under "bad" must fail it.
func LoadExamples(
	dir string,
	cfg config.Config,
	extractor *parsing.Extractor,
) (Document, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return Document{}, fmt.Errorf("resolve examples dir: %w", err)
	}
	rulesRoot := filepath.Join(absolute, examplesRulesDir)
	if _, err := os.Stat(rulesRoot); err != nil {
		return Document{}, fmt.Errorf("open examples %q: %w", dir, err)
	}

	document := Document{Version: currentVersion}
	seen := make(map[caseIdentity]struct{})
	groups := []struct {
		name   string
		expect Expect
	}{
		{examplesGoodDir, ExpectPass},
		{examplesBadDir, ExpectFail},
	}
	for _, rule := range cfg.Rules {
		ruleDir := filepath.Join(rulesRoot, rule.ID)
		info, err := os.Stat(ruleDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Document{}, fmt.Errorf("open examples %q: %w", rule.ID, err)
		}
		if !info.IsDir() {
			continue
		}
		for _, group := range groups {
			groupDir := filepath.Join(ruleDir, group.name)
			entries, err := os.ReadDir(groupDir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return Document{}, fmt.Errorf("open examples %q: %w", groupDir, err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				file := filepath.Join(groupDir, entry.Name())
				if !extractor.Supports(file) {
					return Document{}, fmt.Errorf(
						"example %q has an unsupported language",
						filepath.ToSlash(filepath.Join(rule.ID, group.name, entry.Name())),
					)
				}
				identity := caseIdentity{rule: rule.ID, file: file}
				if _, exists := seen[identity]; exists {
					return Document{}, fmt.Errorf(
						"duplicate example for rule %q and file %q",
						rule.ID,
						entry.Name(),
					)
				}
				seen[identity] = struct{}{}
				document.Cases = append(document.Cases, Case{
					Rule:   rule.ID,
					File:   filepath.ToSlash(filepath.Join(group.name, entry.Name())),
					Expect: group.expect,
					abs:    file,
				})
			}
		}
	}
	if len(document.Cases) == 0 {
		return Document{}, fmt.Errorf("examples %q has no cases", dir)
	}
	return document, nil
}

// FilterRule keeps only the cases for one rule.
func (document Document) FilterRule(ruleID string) (Document, error) {
	if ruleID == "" {
		return document, nil
	}
	filtered := Document{Version: document.Version}
	for _, evalCase := range document.Cases {
		if evalCase.Rule == ruleID {
			filtered.Cases = append(filtered.Cases, evalCase)
		}
	}
	if len(filtered.Cases) == 0 {
		return Document{}, fmt.Errorf("no eval cases for rule %q", ruleID)
	}
	return filtered, nil
}

// prepareCase checks one case and records the full path of its fixture.
func prepareCase(
	evalCase *Case,
	root string,
	cfg config.Config,
	extractor *parsing.Extractor,
	seen map[caseIdentity]struct{},
) error {
	if evalCase.Rule == "" {
		return fmt.Errorf("eval case is missing a rule")
	}
	if evalCase.File == "" {
		return fmt.Errorf("eval case %q is missing a file", caseLabel(*evalCase))
	}
	if evalCase.Expect != ExpectPass && evalCase.Expect != ExpectFail {
		return fmt.Errorf("eval case %q has invalid expect", caseLabel(*evalCase))
	}
	if _, ok := ruleByID(cfg, evalCase.Rule); !ok {
		return fmt.Errorf("unknown eval rule %q", evalCase.Rule)
	}
	absolute, err := filepath.Abs(filepath.Join(root, evalCase.File))
	if err != nil {
		return fmt.Errorf("resolve fixture %q: %w", evalCase.File, err)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return fmt.Errorf("eval fixture %q: %w", evalCase.File, err)
	}
	if info.IsDir() {
		return fmt.Errorf("eval fixture %q is a directory", evalCase.File)
	}
	if !extractor.Supports(absolute) {
		return fmt.Errorf(
			"eval fixture %q has an unsupported language",
			evalCase.File,
		)
	}
	identity := caseIdentity{rule: evalCase.Rule, file: absolute}
	if _, exists := seen[identity]; exists {
		return fmt.Errorf(
			"duplicate eval case for rule %q and file %q",
			evalCase.Rule,
			evalCase.File,
		)
	}
	seen[identity] = struct{}{}
	evalCase.abs = absolute
	return nil
}

// ruleByID finds a rule by id.
func ruleByID(cfg config.Config, id string) (config.Rule, bool) {
	for _, rule := range cfg.Rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return config.Rule{}, false
}

// caseLabel returns a short name for a case.
func caseLabel(evalCase Case) string {
	if evalCase.Name != "" {
		return evalCase.Name
	}
	if evalCase.Rule != "" {
		return evalCase.Rule
	}
	return evalCase.File
}
