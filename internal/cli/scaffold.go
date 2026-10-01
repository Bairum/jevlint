package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codegirl-007/jevlint/internal/evals"
	"github.com/codegirl-007/jevlint/internal/packs"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

// pluginInitOptions holds the settings for `jevlint plugin init`.
type pluginInitOptions struct {
	id        string
	dir       string
	languages []string
	force     bool
}

type scaffoldManifest struct {
	Version   int      `json:"version"`
	ID        string   `json:"id"`
	Languages []string `json:"languages"`
	Rules     string   `json:"rules"`
	Evals     string   `json:"evals,omitempty"`
}

type scaffoldRules struct {
	Rules []scaffoldRule `json:"rules"`
}

type scaffoldRule struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Kinds       []string `json:"kinds"`
}

type scaffoldEvals struct {
	Version int                `json:"version"`
	Cases   []scaffoldEvalCase `json:"cases"`
}

type scaffoldEvalCase struct {
	Rule   string `json:"rule"`
	File   string `json:"file"`
	Expect string `json:"expect"`
}

const (
	scaffoldBadFixture = `package fixtures

// BadExample is deliberately named to trip the example rule.
func BadExample() {}
`

	scaffoldGoodFixture = `package fixtures

func GoodExample() {}
`
)

// scaffoldPack writes a starter pack and returns its directory and the files
// it created.
func scaffoldPack(options pluginInitOptions) (string, []string, error) {
	_, name, err := packs.SplitPackID(options.id)
	if err != nil {
		return "", nil, err
	}
	if len(options.languages) == 0 {
		return "", nil, fmt.Errorf("at least one language is required")
	}
	for _, language := range options.languages {
		if _, err := parsing.ParseSourceLanguage(language); err != nil {
			return "", nil, fmt.Errorf("unknown language %q", language)
		}
	}

	dir := options.dir
	if dir == "" {
		dir = name
	}
	if err := ensureScaffoldDir(dir, options.force); err != nil {
		return "", nil, err
	}

	ruleID := name + "-example"
	withEvals := containsString(options.languages, "go")

	files := map[string]string{
		"pack.json": jsonDocument(scaffoldManifest{
			Version:   1,
			ID:        options.id,
			Languages: options.languages,
			Rules:     "rules.json",
			Evals:     scaffoldEvalsName(withEvals),
		}),
		"rules.json": jsonDocument(scaffoldRules{
			Rules: []scaffoldRule{{
				ID:          ruleID,
				Description: "Fail when a function is named `BadExample`.",
				Severity:    "warning",
				Kinds:       []string{"function"},
			}},
		}),
		"README.md": scaffoldReadme(options.id, options.languages, withEvals),
	}
	if withEvals {
		files[evals.DefaultFile] = jsonDocument(scaffoldEvals{
			Version: 1,
			Cases: []scaffoldEvalCase{
				{Rule: ruleID, File: filepath.ToSlash(filepath.Join("fixtures", "bad.go")), Expect: "fail"},
				{Rule: ruleID, File: filepath.ToSlash(filepath.Join("fixtures", "good.go")), Expect: "pass"},
			},
		})
		files[filepath.Join("fixtures", "bad.go")] = scaffoldBadFixture
		files[filepath.Join("fixtures", "good.go")] = scaffoldGoodFixture
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	created := make([]string, 0, len(names))
	for _, relative := range names {
		target := filepath.Join(dir, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", nil, fmt.Errorf("create %s: %w", filepath.Dir(target), err)
		}
		if err := os.WriteFile(target, []byte(files[relative]), 0o644); err != nil {
			return "", nil, fmt.Errorf("write %s: %w", target, err)
		}
		created = append(created, target)
	}

	if _, err := packs.LoadDir(dir); err != nil {
		return "", nil, fmt.Errorf("generated pack is invalid: %w", err)
	}
	return dir, created, nil
}

func scaffoldEvalsName(withEvals bool) string {
	if withEvals {
		return evals.DefaultFile
	}
	return ""
}

func scaffoldReadme(id string, languages []string, withEvals bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nA [jevlint](https://github.com/codegirl-007/jevlint) pack for %s.\n\n",
		id, strings.Join(languages, ", "))
	b.WriteString("## Layout\n\n")
	b.WriteString("- `pack.json` — pack manifest.\n")
	b.WriteString("- `rules.json` — rule definitions.\n")
	if withEvals {
		fmt.Fprintf(&b, "- `%s` — eval cases.\n", evals.DefaultFile)
		b.WriteString("- `fixtures/` — eval fixtures.\n")
	}
	b.WriteString("\n## Use\n\n")
	b.WriteString("Commit this directory to a git repository, then from your project:\n\n")
	b.WriteString("```sh\njevlint plugin install <git-url>#<path-to-this-pack>\n```\n\n")
	if withEvals {
		b.WriteString("## Test\n\n```sh\njevlint eval --packs\n```\n")
	}
	return b.String()
}

func jsonDocument(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		// All scaffold values are plain structs, so this cannot fail.
		panic(err)
	}
	return string(data) + "\n"
}

func ensureScaffoldDir(dir string, force bool) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o755)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s exists and is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("%s is not empty; pass --force to write anyway", dir)
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
