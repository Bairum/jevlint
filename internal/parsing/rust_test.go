package parsing

import (
	"strings"
	"testing"
)

func TestRustTestDetection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		source string
		test   bool
	}{
		{"plain", "fn example() {}", false},
		{"test attribute", "#[test]\nfn example() {}", true},
		{"qualified attribute", "#[tokio::test]\n// setup\n#[allow(unused)]\nfn example() {}", true},
		{"module cfg", "#[cfg(test)]\nmod checks { struct Value; fn example() {} }", true},
		{"nested cfg", "#[cfg(all(test, not(feature = \"slow\")))]\nmod checks { fn example() {} }", true},
		{"inner cfg", "mod checks { #![cfg(test)] struct Value; fn example() {} }", true},
		{"crate cfg", "#![cfg(test)]\nfn example() {}", true},
		{"negated cfg", "#[cfg(not(test))]\nmod checks { fn example() {} }", false},
		{"test in string", "#[cfg(feature = \"test\")]\nmod checks { fn example() {} }", false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			units, err := testExtractor(t, "rust").Extract("src/lib.rs", []byte(test.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(units) == 0 {
				t.Fatal("no extracted units")
			}
			for _, unit := range units {
				if unit.Test != test.test {
					t.Errorf("%s.Test = %v, want %v", unit.Name, unit.Test, test.test)
				}
			}
		})
	}
}

// Test marking is a pass over byte ranges, so units built outside the
// function/type queries (for example standalone comment units) inherit it.
func TestRustTestRangesMarkAnyUnitKind(t *testing.T) {
	t.Parallel()
	source := []byte("// header\nconst LIMIT: u8 = 3;\n\n#[cfg(test)]\nmod tests {\n    // helper notes\n    const SEED: u8 = 1;\n}\n")
	extractor := testExtractor(t, "rust")
	tree, err := parseSource(extractor.byExtension[".rs"], "src/lib.rs", source)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()
	comment := func(text string) CodeUnit {
		start := uint(strings.Index(string(source), text))
		return CodeUnit{Kind: CodeKindComment, StartByte: start, EndByte: start + uint(len(text))}
	}
	units := []CodeUnit{comment("// header"), comment("// helper notes")}
	markRustTests(units, rustTestRanges(tree.RootNode(), source))
	if units[0].Test || !units[1].Test {
		t.Fatalf("header.Test = %v, helper.Test = %v; want false, true", units[0].Test, units[1].Test)
	}
}

func TestRustTypeCandidatesAndAttributeDocs(t *testing.T) {
	t.Parallel()
	source := `/// Value contract.
#[derive(Clone)]
struct Value;
impl Value { fn identity(&self) -> Self { Self } }
impl Default for Value { fn default() -> Self { Self } }
impl<T> Wrapper<T> { fn identity(&self) -> &Self { self } }
`
	file, err := testExtractor(t, "rust").ExtractFile("src/value.rs", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Types {
		if declaration.Path != "src/value.rs" {
			t.Errorf("declaration %s path = %q", declaration.Name, declaration.Path)
		}
	}
	for _, unit := range file.Units {
		if unit.Name == "Value" {
			if !strings.HasPrefix(unit.Source, "/// Value contract.") {
				t.Errorf("type source lost leading documentation: %q", unit.Source)
			}
			if len(unit.Regions) == 0 || unit.Regions[0].Category != CodeKindDocComment {
				t.Error("leading comment across attribute was not marked as documentation")
			}
			if len(unit.TypeCandidates) != 2 {
				t.Errorf("type candidates = %d, want its two impls", len(unit.TypeCandidates))
			}
		}
		if unit.Name == "identity" && strings.Contains(unit.Source, "Self { Self }") {
			if len(unit.TypeCandidates) != 3 {
				t.Errorf("Self-only method candidates = %d, want definition and two impls", len(unit.TypeCandidates))
			}
			foundSibling := false
			for _, candidate := range unit.TypeCandidates {
				if strings.Contains(candidate.Source, "impl Default for Value") {
					foundSibling = true
				}
			}
			if !foundSibling {
				t.Error("Self-only method did not receive its sibling Default impl")
			}
		}
	}
	foundGeneric := false
	for _, declaration := range file.Types {
		if declaration.Name == "Wrapper" {
			foundGeneric = true
		}
	}
	if !foundGeneric {
		t.Error("generic impl was not extracted")
	}
}
