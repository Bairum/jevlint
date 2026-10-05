package parsing

import (
	"path/filepath"
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// RustModule records possible discovered files for an external module.
type RustModule struct {
	Paths []string
	Test  bool
}

func rustAttributes(node *tree_sitter.Node, source []byte) []string {
	var attributes []string
	for previous := node.PrevNamedSibling(); previous != nil; previous = previous.PrevNamedSibling() {
		switch previous.Kind() {
		case "attribute_item":
			attributes = append(attributes, rustAttributeText(previous, source))
		case "line_comment", "block_comment":
		default:
			return attributes
		}
	}
	return attributes
}

func rustAttributeText(node *tree_sitter.Node, source []byte) string {
	text := strings.TrimSpace(node.Utf8Text(source))
	text = strings.TrimPrefix(text, "#")
	text = strings.TrimPrefix(text, "!")
	text = strings.TrimPrefix(text, "[")
	return strings.TrimSpace(strings.TrimSuffix(text, "]"))
}

// rustTestRanges returns the byte ranges that structurally belong to tests:
// #[test]-style functions, cfg(test) modules, and files or module bodies with
// #![cfg(test)]. Each range starts at the item's first outer attribute.
func rustTestRanges(root *tree_sitter.Node, source []byte) [][2]uint {
	if rustInnerCfgTest(root, source) {
		return [][2]uint{{0, uint(len(source))}}
	}
	var ranges [][2]uint
	var visit func(*tree_sitter.Node)
	visit = func(node *tree_sitter.Node) {
		if rustTestItem(node, source) {
			ranges = append(ranges, [2]uint{rustAttributedStart(node), node.EndByte()})
			return
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	return ranges
}

// markRustTests marks every unit inside a test range, however the unit was
// built, so later unit kinds inherit test detection without their own checks.
func markRustTests(units []CodeUnit, ranges [][2]uint) {
	for index := range units {
		units[index].Test = rustInTestRange(units[index], ranges)
	}
}

func rustInTestRange(unit CodeUnit, ranges [][2]uint) bool {
	start := max(unit.StartByte, unit.docStart)
	for _, testRange := range ranges {
		if start >= testRange[0] && unit.EndByte <= testRange[1] {
			return true
		}
	}
	return false
}

func rustTestItem(node *tree_sitter.Node, source []byte) bool {
	switch node.Kind() {
	case "function_item":
		for _, attribute := range rustAttributes(node, source) {
			name := attribute
			if index := strings.IndexAny(name, "(="); index >= 0 {
				name = name[:index]
			}
			name = strings.Join(strings.Fields(name), "")
			if name == "test" || strings.HasSuffix(name, "::test") {
				return true
			}
		}
	case "mod_item":
		for _, attribute := range rustAttributes(node, source) {
			if rustTestCfg(attribute) {
				return true
			}
		}
		if body := node.ChildByFieldName("body"); body != nil {
			return rustInnerCfgTest(body, source)
		}
	}
	return false
}

// rustInnerCfgTest reports whether a file or module body carries #![cfg(test)].
func rustInnerCfgTest(node *tree_sitter.Node, source []byte) bool {
	for index := range node.NamedChildCount() {
		child := node.NamedChild(index)
		if child.Kind() == "inner_attribute_item" && rustTestCfg(rustAttributeText(child, source)) {
			return true
		}
	}
	return false
}

// rustAttributedStart includes an item's outer attributes in its range.
func rustAttributedStart(node *tree_sitter.Node) uint {
	start := node.StartByte()
	for previous := node.PrevNamedSibling(); previous != nil; previous = previous.PrevNamedSibling() {
		switch previous.Kind() {
		case "attribute_item":
			start = previous.StartByte()
		case "line_comment", "block_comment":
		default:
			return start
		}
	}
	return start
}

func rustTestCfg(attribute string) bool {
	name, arguments, ok := strings.Cut(attribute, "(")
	if !ok || strings.TrimSpace(name) != "cfg" || !strings.HasSuffix(arguments, ")") {
		return false
	}
	return rustCfgTest(strings.TrimSuffix(arguments, ")"), false)
}

// rustCfgTest finds a positive test predicate, not strings or negated predicates.
func rustCfgTest(predicate string, negated bool) bool {
	predicate = strings.TrimSpace(predicate)
	if predicate == "test" {
		return !negated
	}
	name, arguments, ok := strings.Cut(predicate, "(")
	if !ok || !strings.HasSuffix(arguments, ")") {
		return false
	}
	if strings.TrimSpace(name) == "not" {
		negated = !negated
	}
	arguments = strings.TrimSuffix(arguments, ")")
	depth, start := 0, 0
	quoted, escaped := false, false
	for index := 0; index <= len(arguments); index++ {
		if index == len(arguments) || (arguments[index] == ',' && depth == 0 && !quoted) {
			if rustCfgTest(arguments[start:index], negated) {
				return true
			}
			start = index + 1
			continue
		}
		value := arguments[index]
		if quoted {
			if value == '"' && !escaped {
				quoted = false
			}
			escaped = value == '\\' && !escaped
			continue
		}
		switch value {
		case '"':
			quoted = true
		case '(':
			depth++
		case ')':
			depth--
		}
	}
	return false
}

func rustModules(path string, root *tree_sitter.Node, source []byte, tests [][2]uint) []RustModule {
	var modules []RustModule
	var visit func(*tree_sitter.Node)
	visit = func(node *tree_sitter.Node) {
		if node.Kind() == "mod_item" && node.ChildByFieldName("body") == nil {
			name := node.ChildByFieldName("name")
			if name != nil {
				directory := filepath.Dir(path)
				base := filepath.Base(path)
				if base != "mod.rs" && base != "lib.rs" && base != "main.rs" {
					directory = filepath.Join(directory, strings.TrimSuffix(base, filepath.Ext(base)))
				}
				module := RustModule{Test: rustInTestRange(CodeUnit{StartByte: node.StartByte(), EndByte: node.EndByte()}, tests)}
				for _, attribute := range rustAttributes(node, source) {
					key, value, ok := strings.Cut(attribute, "=")
					if ok && strings.TrimSpace(key) == "path" {
						if target, err := strconv.Unquote(strings.TrimSpace(value)); err == nil {
							module.Paths = []string{filepath.ToSlash(filepath.Join(filepath.Dir(path), target))}
						}
					}
				}
				if len(module.Paths) == 0 {
					moduleName := name.Utf8Text(source)
					module.Paths = []string{
						filepath.ToSlash(filepath.Join(directory, moduleName+".rs")),
						filepath.ToSlash(filepath.Join(directory, moduleName, "mod.rs")),
					}
				}
				modules = append(modules, module)
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	return modules
}
