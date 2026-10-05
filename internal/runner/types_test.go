package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

func rustPlannedFile(t *testing.T, path, source string) plannedFile {
	t.Helper()
	file, err := testExtractor(t, "rust").ExtractFile(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return plannedFile{relative: path, units: file.Units, types: file.Types, testModules: file.TestModules}
}

func TestMarkRustTestModuleFiles(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path   string
		source string
		target string
		test   bool
	}{
		{"src/lib.rs", "#[cfg(test)] mod checks;", "src/checks.rs", true},
		{"src/main.rs", "#[cfg(test)] mod checks;", "src/checks/mod.rs", true},
		{"src/mod.rs", "#[cfg(test)] mod checks;", "src/checks.rs", true},
		{"src/value.rs", "#[cfg(all(test, feature = \"x\"))] mod checks;", "src/value/checks.rs", true},
		{"src/value.rs", "#[cfg(test)] mod checks;", "src/value/checks/mod.rs", true},
		{"src/value.rs", "#[path = \"special.rs\"]\n// module contract\n#[cfg(test)] mod checks;", "src/special.rs", true},
		{"src/lib.rs", "#[cfg(not(test))] mod checks;", "src/checks.rs", false},
	}
	for _, test := range cases {
		t.Run(test.target+test.source, func(t *testing.T) {
			files := []plannedFile{
				rustPlannedFile(t, test.path, test.source),
				rustPlannedFile(t, test.target, "struct Value; fn helper() {}"),
				rustPlannedFile(t, "src/other.rs", "fn production() {}"),
			}
			markTestModuleFiles(files)
			for _, unit := range files[1].units {
				if unit.Test != test.test {
					t.Errorf("%s.Test = %v, want %v", unit.Name, unit.Test, test.test)
				}
			}
			if files[2].units[0].Test {
				t.Error("unrelated discovered file marked as test")
			}
		})
	}
}

func TestResolveTypeContextUniqueDefinitions(t *testing.T) {
	t.Parallel()
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprint(ambiguous), func(t *testing.T) {
			files := []plannedFile{
				rustPlannedFile(t, "src/value.rs", "struct Value; impl Value { fn identity(&self) -> Self { Self } }"),
				rustPlannedFile(t, "src/impl.rs", "impl Marker for Value {}"),
				rustPlannedFile(t, "src/use.rs", "fn consume(value: Value) {}"),
			}
			if ambiguous {
				files = append(files, rustPlannedFile(t, "src/other.rs", "struct Value; impl Value {}"))
			}
			resolveTypeContext(files)
			for _, unit := range append(files[0].units, files[2].units...) {
				found := false
				for _, candidate := range unit.TypeCandidates {
					if candidate.Path == "src/impl.rs" {
						found = true
					}
				}
				if found == ambiguous {
					t.Errorf("%s cross-file impl present = %v, ambiguous = %v", unit.Name, found, ambiguous)
				}
			}
		})
	}
}

func TestAllowedTypesFiltersBeforeCaps(t *testing.T) {
	t.Parallel()
	baseline := parsing.TypeDeclaration{Name: "Baseline", Path: "baseline.rs", Source: "struct Baseline;"}
	unit := parsing.CodeUnit{RelatedTypes: []parsing.TypeDeclaration{baseline}}
	unit.TypeCandidates = append(unit.TypeCandidates, baseline)
	for index := range 15 {
		unit.TypeCandidates = append(unit.TypeCandidates, parsing.TypeDeclaration{
			Name: "Hidden", Path: fmt.Sprintf("private/%02d.rs", index), Source: strings.Repeat("x", maxTypeSourceBytes),
		})
	}
	for index := 13; index >= 0; index-- {
		unit.TypeCandidates = append(unit.TypeCandidates, parsing.TypeDeclaration{
			Name: "Visible", Path: fmt.Sprintf("visible/%02d.rs", index), Source: "impl Visible {}",
		})
	}
	allowed, err := allowedTypes(unit, config.Rule{Exclude: []string{"private/**"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != maxTypeDeclarations || allowed[0].Path != "visible/00.rs" || allowed[11].Path != "visible/11.rs" {
		t.Fatalf("filtered deterministic capped types = %#v", allowed)
	}
	unit.TypeCandidates = []parsing.TypeDeclaration{
		{Name: "Hidden", Path: "private/huge.rs", Source: strings.Repeat("x", maxTypeSourceBytes)},
		{Name: "First", Path: "a.rs", Source: strings.Repeat("x", maxTypeSourceBytes-1)},
		{Name: "TooLarge", Path: "b.rs", Source: "xx"},
		{Name: "Last", Path: "c.rs", Source: "x"},
	}
	allowed, err = allowedTypes(unit, config.Rule{Exclude: []string{"private/**"}})
	if err != nil || len(allowed) != 2 || allowed[1].Name != "Last" {
		t.Fatalf("source-byte cap = %#v, %v", allowed, err)
	}
}

func TestRustTypeContextDiscoveryAndTestModulePlanning(t *testing.T) {
	root := privacyRepo(t)
	writeFile(t, filepath.Join(root, ".gitignore"), "ignored.rs\n")
	writeFile(t, filepath.Join(root, "lib.rs"), "#[cfg(test)] mod checks;\n")
	writeFile(t, filepath.Join(root, "checks.rs"), "fn helper() {}\n")
	writeFile(t, filepath.Join(root, "value.rs"), "struct Value;\n")
	writeFile(t, filepath.Join(root, "impl.rs"), "impl Marker for Value {}\n")
	writeFile(t, filepath.Join(root, "ignored.rs"), "impl Secret for Value {}\n")
	evaluator := &capturingEvaluator{}
	runner := Runner{Extractor: testExtractor(t, "rust"), Evaluator: evaluator}
	rule := config.Rule{
		ID: "types", Description: "Check primary type.", Severity: config.SeverityWarning,
		Kinds: []config.TargetKind{config.TargetKindType}, Include: []string{"value.rs"},
		Context: config.RuleContext{Types: true},
	}
	_, err := runner.Evaluate(context.Background(), config.Config{Rules: []config.Rule{rule}}, Options{Root: root, Concurrency: 1})
	if err != nil {
		t.Fatal(err)
	}
	batch, ok := batchFor(evaluator.batches, "Value", "types")
	if !ok || len(batch.CodeUnit.RelatedTypes) != 1 || batch.CodeUnit.RelatedTypes[0].Path != "impl.rs" {
		t.Fatalf("discovered context = %#v", batch.CodeUnit.RelatedTypes)
	}
	project, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer project.Close()
	testRule := config.Rule{ID: "functions", Kinds: []config.TargetKind{config.TargetKindFunction}, Include: []string{"checks.rs"}}
	_, jobs, err := runner.planEvaluations(context.Background(), config.Config{Rules: []config.Rule{testRule}}, project,
		[]string{filepath.Join(root, "lib.rs"), filepath.Join(root, "checks.rs")}, nil)
	if err != nil || len(jobs) != 0 {
		t.Fatalf("test module file planned for production rule: jobs=%d, error=%v", len(jobs), err)
	}
}
