package parsing

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// CodeUnit is one function or type read from a source file.
type CodeUnit struct {
	Kind         CodeKind          `json:"kind"`
	Name         string            `json:"name"`
	Language     SourceLanguage    `json:"language"`
	Path         string            `json:"path"`
	Source       string            `json:"source"`
	ParentSource string            `json:"parentSource,omitempty"`
	RegionKind   NodeKind          `json:"regionKind,omitempty"`
	StartLine    uint              `json:"startLine"`
	EndLine      uint              `json:"endLine"`
	StartColumn  uint              `json:"startColumn"`
	EndColumn    uint              `json:"endColumn"`
	StartByte    uint              `json:"startByte"`
	EndByte      uint              `json:"endByte"`
	RelatedTypes []TypeDeclaration `json:"types,omitempty"`
	Callees      []CalleeContext   `json:"callees,omitempty"`
	CallRefs     []CallRef         `json:"-"`
	Resolved     []CalleeContext   `json:"-"`
	Regions      []Region          `json:"-"`
}

// CodeKind names the kind of a code unit.
type CodeKind int

const (
	CodeKindUnknown CodeKind = iota
	CodeKindFunction
	CodeKindType
	CodeKindComment
	CodeKindField
	CodeKindStatement
	CodeKindRegion
)

// SourceLanguage names a language the parser supports.
type SourceLanguage int

const (
	SourceLanguageUnknown SourceLanguage = iota
	SourceLanguageC
	SourceLanguageCPP
	SourceLanguageCSharp
	SourceLanguageGo
	SourceLanguageJava
	SourceLanguageJavaScript
	SourceLanguageKotlin
	SourceLanguagePHP
	SourceLanguagePython
	SourceLanguageRuby
	SourceLanguageRust
	SourceLanguageTSX
	SourceLanguageTypeScript
)

// NodeKind is a grammar node name.
type NodeKind string

const maxAttachedRegions = 24

// Region is a smaller piece of a code unit, such as a comment or statement.
type Region struct {
	Category    CodeKind `json:"category"`
	Kind        NodeKind `json:"kind"`
	Source      string   `json:"source"`
	StartLine   uint     `json:"startLine"`
	EndLine     uint     `json:"endLine"`
	StartColumn uint     `json:"startColumn"`
	EndColumn   uint     `json:"endColumn"`
	StartByte   uint     `json:"startByte"`
	EndByte     uint     `json:"endByte"`
}

// TypeDeclaration is a type found in a source file.
type TypeDeclaration struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	StartLine uint   `json:"startLine"`
	EndLine   uint   `json:"endLine"`
	StartByte uint   `json:"startByte"`
	EndByte   uint   `json:"endByte"`
}

// languageSpec holds the grammar and queries for one language.
type languageSpec struct {
	name             string
	id               SourceLanguage
	language         *tree_sitter.Language
	functionQuery    *tree_sitter.Query
	typeQuery        *tree_sitter.Query
	typeContextQuery *tree_sitter.Query
	callQuery        *tree_sitter.Query
	regionKinds      map[string]CodeKind
}

// Extractor reads code units from source files.
type Extractor struct {
	byExtension map[string]languageSpec
}

// String returns the kind name.
func (kind CodeKind) String() string {
	switch kind {
	case CodeKindFunction:
		return "function"
	case CodeKindType:
		return "type"
	case CodeKindComment:
		return "comment"
	case CodeKindField:
		return "field"
	case CodeKindStatement:
		return "statement"
	case CodeKindRegion:
		return "region"
	default:
		return "unknown"
	}
}

// MarshalJSON writes the kind as its name.
func (kind CodeKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

// UnmarshalJSON reads a kind from its name.
func (kind *CodeKind) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, ok := ParseCodeKind(value)
	if !ok {
		return fmt.Errorf("invalid code kind %q", value)
	}
	*kind = parsed
	return nil
}

// ParseCodeKind turns a kind name into a kind value.
func ParseCodeKind(value string) (CodeKind, bool) {
	switch value {
	case "function":
		return CodeKindFunction, true
	case "type":
		return CodeKindType, true
	case "comment":
		return CodeKindComment, true
	case "field":
		return CodeKindField, true
	case "statement":
		return CodeKindStatement, true
	case "region":
		return CodeKindRegion, true
	default:
		return CodeKindUnknown, false
	}
}

// String returns the language name.
func (language SourceLanguage) String() string {
	switch language {
	case SourceLanguageC:
		return "c"
	case SourceLanguageCPP:
		return "cpp"
	case SourceLanguageCSharp:
		return "csharp"
	case SourceLanguageGo:
		return "go"
	case SourceLanguageJava:
		return "java"
	case SourceLanguageJavaScript:
		return "javascript"
	case SourceLanguageKotlin:
		return "kotlin"
	case SourceLanguagePHP:
		return "php"
	case SourceLanguagePython:
		return "python"
	case SourceLanguageRuby:
		return "ruby"
	case SourceLanguageRust:
		return "rust"
	case SourceLanguageTSX:
		return "tsx"
	case SourceLanguageTypeScript:
		return "typescript"
	default:
		return "unknown"
	}
}

// MarshalJSON writes the language as its name.
func (language SourceLanguage) MarshalJSON() ([]byte, error) {
	return json.Marshal(language.String())
}

// UnmarshalJSON reads a language from its name.
func (language *SourceLanguage) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, ok := ParseSourceLanguage(value)
	if !ok {
		return fmt.Errorf("invalid source language %q", value)
	}
	*language = parsed
	return nil
}

// ParseSourceLanguage turns a language name into a language value.
func ParseSourceLanguage(value string) (SourceLanguage, bool) {
	switch value {
	case "c":
		return SourceLanguageC, true
	case "cpp":
		return SourceLanguageCPP, true
	case "csharp":
		return SourceLanguageCSharp, true
	case "go":
		return SourceLanguageGo, true
	case "java":
		return SourceLanguageJava, true
	case "javascript":
		return SourceLanguageJavaScript, true
	case "kotlin":
		return SourceLanguageKotlin, true
	case "php":
		return SourceLanguagePHP, true
	case "python":
		return SourceLanguagePython, true
	case "ruby":
		return SourceLanguageRuby, true
	case "rust":
		return SourceLanguageRust, true
	case "tsx":
		return SourceLanguageTSX, true
	case "typescript":
		return SourceLanguageTypeScript, true
	default:
		return SourceLanguageUnknown, false
	}
}

// Supports reports whether a file extension has a language.
func (extractor *Extractor) Supports(path string) bool {
	_, ok := extractor.byExtension[strings.ToLower(filepath.Ext(path))]
	return ok
}

// Extensions returns the file extensions the parser knows.
func (extractor *Extractor) Extensions() []string {
	extensions := make([]string, 0, len(extractor.byExtension))
	for extension := range extractor.byExtension {
		extensions = append(extensions, extension)
	}
	sort.Strings(extensions)
	return extensions
}

// Extract reads the code units from one source file.
func (extractor *Extractor) Extract(path string, source []byte) ([]CodeUnit, error) {
	spec, ok := extractor.byExtension[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return nil, fmt.Errorf("unsupported source file %q", path)
	}

	tree, err := parseSource(spec, path, source)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()
	functions := extractMatches(
		spec,
		path,
		source,
		root,
		spec.functionQuery,
		"function",
		CodeKindFunction,
	)
	attachCallRefs(spec, source, root, functions)
	types := extractMatches(
		spec,
		path,
		source,
		root,
		spec.typeQuery,
		"type",
		CodeKindType,
	)
	declarations := extractTypeDeclarations(spec, path, source, root, types)
	attachRelatedTypes(functions, declarations)

	units := append(functions, types...)
	attachRegions(units, extractRegions(root, source, spec.regionKinds))
	sortCodeUnits(units)
	return units, nil
}

// parseSource builds a syntax tree and rejects files with syntax errors.
func parseSource(
	spec languageSpec,
	path string,
	source []byte,
) (*tree_sitter.Tree, error) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(spec.language); err != nil {
		return nil, fmt.Errorf("set %s grammar: %w", spec.name, err)
	}

	tree := parser.Parse(source, nil)
	if tree == nil {
		return nil, fmt.Errorf("parse %q: parser returned no syntax tree", path)
	}

	root := tree.RootNode()
	if root.HasError() {
		return nil, fmt.Errorf("parse %q: source contains syntax errors", path)
	}
	return tree, nil
}

// extractTypeDeclarations lists the types found in a file.
func extractTypeDeclarations(
	spec languageSpec,
	path string,
	source []byte,
	root *tree_sitter.Node,
	types []CodeUnit,
) []TypeDeclaration {
	declarations := make([]TypeDeclaration, 0, len(types))
	for _, unit := range types {
		declarations = append(declarations, TypeDeclaration{
			Name:      unit.Name,
			Source:    unit.Source,
			StartLine: unit.StartLine,
			EndLine:   unit.EndLine,
			StartByte: unit.StartByte,
			EndByte:   unit.EndByte,
		})
	}
	if spec.typeContextQuery != nil {
		contextTypes := extractMatches(
			spec,
			path,
			source,
			root,
			spec.typeContextQuery,
			"type",
			CodeKindType,
		)
		for _, unit := range contextTypes {
			declarations = append(declarations, TypeDeclaration{
				Name:      unit.Name,
				Source:    unit.Source,
				StartLine: unit.StartLine,
				EndLine:   unit.EndLine,
				StartByte: unit.StartByte,
				EndByte:   unit.EndByte,
			})
		}
	}
	return declarations
}

// attachRelatedTypes adds the types that a function mentions.
func attachRelatedTypes(functions []CodeUnit, declarations []TypeDeclaration) {
	for index := range functions {
		for _, declaration := range declarations {
			if declarationContains(declaration, functions[index]) ||
				containsIdentifier(functions[index].Source, declaration.Name) {
				functions[index].RelatedTypes = append(
					functions[index].RelatedTypes,
					declaration,
				)
			}
		}
	}
}

// attachRegions adds the smaller pieces found inside each code unit.
func attachRegions(units []CodeUnit, regions []Region) {
	for index := range units {
		for _, region := range regions {
			if region.StartByte < units[index].StartByte ||
				region.EndByte > units[index].EndByte {
				continue
			}
			units[index].Regions = append(units[index].Regions, region)
			if len(units[index].Regions) == maxAttachedRegions {
				break
			}
		}
	}
}

// sortCodeUnits orders code units by where they start.
func sortCodeUnits(units []CodeUnit) {
	sort.SliceStable(units, func(i, j int) bool {
		if units[i].StartByte == units[j].StartByte {
			return units[i].Kind == CodeKindType
		}
		return units[i].StartByte < units[j].StartByte
	})
}

// extractMatches runs one query and builds a code unit for each match.
func extractMatches(
	spec languageSpec,
	path string,
	source []byte,
	root *tree_sitter.Node,
	query *tree_sitter.Query,
	captureName string,
	kind CodeKind,
) []CodeUnit {
	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(query, root, source)
	unitIndex, nameIndex, haveCaptures := captureIndexes(query, captureName)
	units := make([]CodeUnit, 0)
	for {
		match := matches.Next()
		if match == nil {
			break
		}
		if !haveCaptures {
			continue
		}

		unitNode, nameNode := capturePair(match, unitIndex, nameIndex)
		if unitNode == nil || nameNode == nil {
			continue
		}
		units = append(units, codeUnitFromNodes(
			spec,
			path,
			source,
			kind,
			unitNode,
			nameNode,
		))
	}
	return units
}

const identifierCapture = "name"

// captureIndexes finds the positions of the captures a query needs.
func captureIndexes(query *tree_sitter.Query, unitCapture string) (uint32, uint32, bool) {
	var unitIndex, nameIndex uint32
	var haveUnit, haveName bool
	for index, name := range query.CaptureNames() {
		if name == unitCapture {
			unitIndex = uint32(index)
			haveUnit = true
			continue
		}
		if name == identifierCapture {
			nameIndex = uint32(index)
			haveName = true
		}
	}
	return unitIndex, nameIndex, haveUnit && haveName
}

// capturePair pulls the unit and name nodes from one match.
func capturePair(
	match *tree_sitter.QueryMatch,
	unitIndex uint32,
	nameIndex uint32,
) (*tree_sitter.Node, *tree_sitter.Node) {
	var unitNode, nameNode *tree_sitter.Node
	for index := range match.Captures {
		capture := &match.Captures[index]
		node := capture.Node
		switch capture.Index {
		case unitIndex:
			unitNode = &node
		case nameIndex:
			nameNode = &node
		}
	}
	return unitNode, nameNode
}

// codeUnitFromNodes builds a code unit from its grammar nodes.
func codeUnitFromNodes(
	spec languageSpec,
	path string,
	source []byte,
	kind CodeKind,
	unitNode *tree_sitter.Node,
	nameNode *tree_sitter.Node,
) CodeUnit {
	sourceNode := documentationAnchor(unitNode)
	sourceStartByte, sourceStartPosition := leadingCommentStart(sourceNode, source, spec.regionKinds)
	end := unitNode.EndPosition()
	return CodeUnit{
		Kind:        kind,
		Name:        nameNode.Utf8Text(source),
		Language:    spec.id,
		Path:        path,
		Source:      string(source[sourceStartByte:unitNode.EndByte()]),
		StartLine:   sourceStartPosition.Row + 1,
		EndLine:     end.Row + 1,
		StartColumn: sourceStartPosition.Column,
		EndColumn:   end.Column,
		StartByte:   sourceStartByte,
		EndByte:     unitNode.EndByte(),
	}
}

// extractRegions walks the tree and collects the smaller pieces it finds.
func extractRegions(
	root *tree_sitter.Node,
	source []byte,
	kinds map[string]CodeKind,
) []Region {
	regions := make([]Region, 0)
	var walk func(*tree_sitter.Node)
	walk = func(node *tree_sitter.Node) {
		if category, ok := kinds[node.Kind()]; ok {
			start := node.StartPosition()
			end := node.EndPosition()
			regions = append(regions, Region{
				Category:    category,
				Kind:        NodeKind(node.Kind()),
				Source:      node.Utf8Text(source),
				StartLine:   start.Row + 1,
				EndLine:     end.Row + 1,
				StartColumn: start.Column,
				EndColumn:   end.Column,
				StartByte:   node.StartByte(),
				EndByte:     node.EndByte(),
			})
		}
		for index := uint(0); index < node.NamedChildCount(); index++ {
			child := node.NamedChild(index)
			if child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	sort.SliceStable(regions, func(i, j int) bool {
		if regions[i].StartByte == regions[j].StartByte {
			return regions[i].EndByte < regions[j].EndByte
		}
		return regions[i].StartByte < regions[j].StartByte
	})
	return regions
}

// documentationAnchor returns the node that should carry a doc comment.
func documentationAnchor(node *tree_sitter.Node) *tree_sitter.Node {
	parent := node.Parent()
	if parent != nil && parent.Kind() == "decorated_definition" {
		return parent
	}
	return node
}

// leadingCommentStart walks back over the comments that belong to a node.
func leadingCommentStart(
	node *tree_sitter.Node,
	source []byte,
	regionKinds map[string]CodeKind,
) (uint, tree_sitter.Point) {
	startByte := node.StartByte()
	startPosition := node.StartPosition()

	for previous := node.PrevNamedSibling(); previous != nil; previous = previous.PrevNamedSibling() {
		if regionKinds[previous.Kind()] != CodeKindComment ||
			!isAdjacentCommentGap(source[previous.EndByte():startByte]) {
			break
		}
		startByte = previous.StartByte()
		startPosition = previous.StartPosition()
	}
	return startByte, startPosition
}

// isAdjacentCommentGap reports whether one line separates a comment from code.
func isAdjacentCommentGap(gap []byte) bool {
	newlines := 0
	for _, value := range gap {
		switch value {
		case '\n':
			newlines++
			if newlines > 1 {
				return false
			}
		case ' ', '\t', '\r':
		default:
			return false
		}
	}
	return true
}

// declarationContains reports whether a type holds a code unit.
func declarationContains(declaration TypeDeclaration, unit CodeUnit) bool {
	return declaration.StartByte <= unit.StartByte && declaration.EndByte >= unit.EndByte
}

// containsIdentifier reports whether a name appears as a whole word.
func containsIdentifier(source string, identifier string) bool {
	for index := 0; index < len(source); {
		start := strings.Index(source[index:], identifier)
		if start < 0 {
			return false
		}
		start += index
		end := start + len(identifier)
		beforeBoundary := start == 0 || !isIdentifierByte(source[start-1])
		afterBoundary := end == len(source) || !isIdentifierByte(source[end])
		if beforeBoundary && afterBoundary {
			return true
		}
		index = end
	}
	return false
}

// isIdentifierByte reports whether a byte can be part of a name.
func isIdentifierByte(value byte) bool {
	return value == '_' ||
		value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9'
}

const javascriptFunctionQuery = `
(function_declaration
  name: (identifier) @name) @function

(generator_function_declaration
  name: (identifier) @name) @function

(method_definition
  name: [(property_identifier) (private_property_identifier)] @name) @function

(variable_declarator
  name: (identifier) @name
  value: [(arrow_function) (function_expression)] @function)
`

const javascriptTypeQuery = `
(class_declaration
  name: (identifier) @name) @type
`

const javascriptCallQuery = `
(call_expression
  function: [
    (identifier) @call
    (member_expression) @call
  ])
`

const typescriptTypeQuery = `
(class_declaration
  name: (type_identifier) @name) @type

(abstract_class_declaration
  name: (type_identifier) @name) @type

(interface_declaration
  name: (type_identifier) @name) @type

(type_alias_declaration
  name: (type_identifier) @name) @type

(enum_declaration
  name: (identifier) @name) @type
`

const pythonFunctionQuery = `
(function_definition
  name: (identifier) @name) @function
`

const pythonTypeQuery = `
(class_definition
  name: (identifier) @name) @type
`

const pythonCallQuery = `
(call
  function: [
    (identifier) @call
    (attribute) @call
  ])
`

const goFunctionQuery = `
(function_declaration
  name: (identifier) @name) @function

(method_declaration
  name: (field_identifier) @name) @function
`

const goTypeQuery = `
(type_declaration
  (type_spec
    name: (type_identifier) @name)) @type

(type_declaration
  (type_alias
    name: (type_identifier) @name)) @type
`

const goCallQuery = `
(call_expression
  function: [
    (identifier) @call
    (selector_expression) @call
  ])
`

const rustFunctionQuery = `
(function_item
  name: (identifier) @name) @function
`

const rustCallQuery = `
(call_expression
  function: [
    (identifier) @call
    (field_expression) @call
    (scoped_identifier) @call
  ])
`

const rustTypeQuery = `
(struct_item
  name: (type_identifier) @name) @type

(enum_item
  name: (type_identifier) @name) @type

(union_item
  name: (type_identifier) @name) @type

(trait_item
  name: (type_identifier) @name) @type

(type_item
  name: (type_identifier) @name) @type
`

const rustImplQuery = `
(impl_item
  type: (type_identifier) @name) @type
`
