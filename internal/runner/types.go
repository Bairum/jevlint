package runner

import (
	"sort"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
	"github.com/codegirl-007/jevlint/internal/scoping"
)

const (
	maxTypeDeclarations = 12
	maxTypeSourceBytes  = 16 * 1024
)

type typeIdentity struct {
	path  string
	start uint
	end   uint
}

func declarationIdentity(declaration parsing.TypeDeclaration) typeIdentity {
	return typeIdentity{declaration.Path, declaration.StartByte, declaration.EndByte}
}

func unitIdentity(unit parsing.CodeUnit) typeIdentity {
	return typeIdentity{unit.Path, unit.StartByte, unit.EndByte}
}

func rulesWantTypes(rules []config.Rule) bool {
	for _, rule := range rules {
		if rule.Context.Types {
			return true
		}
	}
	return false
}

// resolveTypeContext deliberately avoids binding ambiguous bare type names.
func resolveTypeContext(files []plannedFile) {
	definitions := make(map[string]map[typeIdentity]bool)
	declarations := make(map[string][]parsing.TypeDeclaration)
	seen := make(map[typeIdentity]bool)
	for _, file := range files {
		for _, unit := range file.units {
			if unit.Kind != parsing.CodeKindType {
				continue
			}
			if definitions[unit.Name] == nil {
				definitions[unit.Name] = make(map[typeIdentity]bool)
			}
			definitions[unit.Name][unitIdentity(unit)] = true
		}
		for _, declaration := range file.types {
			identity := declarationIdentity(declaration)
			if !seen[identity] {
				declarations[declaration.Name] = append(declarations[declaration.Name], declaration)
				seen[identity] = true
			}
		}
	}
	names := make([]string, 0, len(definitions))
	for name, matches := range definitions {
		if len(matches) == 1 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for fileIndex := range files {
		for unitIndex := range files[fileIndex].units {
			unit := &files[fileIndex].units[unitIndex]
			owners := make(map[string]bool)
			if unit.Kind == parsing.CodeKindType {
				owners[unit.Name] = true
			}
			for _, declaration := range files[fileIndex].types {
				if declaration.StartByte <= unit.StartByte && declaration.EndByte >= unit.EndByte {
					owners[declaration.Name] = true
				}
			}
			attached := make(map[typeIdentity]bool)
			attached[unitIdentity(*unit)] = true
			for _, candidate := range unit.TypeCandidates {
				attached[declarationIdentity(candidate)] = true
			}
			for _, name := range names {
				if !owners[name] && !parsing.ContainsIdentifier(unit.Source, name) {
					continue
				}
				for _, declaration := range declarations[name] {
					identity := declarationIdentity(declaration)
					if !attached[identity] {
						unit.TypeCandidates = append(unit.TypeCandidates, declaration)
						attached[identity] = true
					}
				}
			}
		}
	}
}

func allowedTypes(unit parsing.CodeUnit, rule config.Rule) ([]parsing.TypeDeclaration, error) {
	seen := make(map[typeIdentity]bool, len(unit.RelatedTypes))
	for _, declaration := range unit.RelatedTypes {
		seen[declarationIdentity(declaration)] = true
	}
	candidates := make([]parsing.TypeDeclaration, 0, len(unit.TypeCandidates))
	for _, declaration := range unit.TypeCandidates {
		excluded, err := scoping.Excluded(rule, declaration.Path)
		if err != nil {
			return nil, err
		}
		identity := declarationIdentity(declaration)
		if excluded || seen[identity] {
			continue
		}
		seen[identity] = true
		candidates = append(candidates, declaration)
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.StartByte != right.StartByte {
			return left.StartByte < right.StartByte
		}
		if left.EndByte != right.EndByte {
			return left.EndByte < right.EndByte
		}
		return left.Name < right.Name
	})
	var allowed []parsing.TypeDeclaration
	size := 0
	for _, declaration := range candidates {
		if len(allowed) == maxTypeDeclarations {
			break
		}
		if len(declaration.Source) > maxTypeSourceBytes-size {
			continue
		}
		allowed = append(allowed, declaration)
		size += len(declaration.Source)
	}
	return allowed, nil
}

// markTestModuleFiles follows only module targets present in the discovered set.
func markTestModuleFiles(files []plannedFile) {
	byPath := make(map[string]int, len(files))
	for index, file := range files {
		byPath[file.relative] = index
	}
	marked := make(map[int]bool)
	var mark func(int)
	mark = func(index int) {
		if marked[index] {
			return
		}
		marked[index] = true
		for unitIndex := range files[index].units {
			files[index].units[unitIndex].Test = true
		}
		for _, module := range files[index].testModules {
			for _, path := range module.Paths {
				if target, ok := byPath[path]; ok {
					mark(target)
				}
			}
		}
	}
	for _, file := range files {
		for _, module := range file.testModules {
			if !module.Test {
				continue
			}
			for _, path := range module.Paths {
				if target, ok := byPath[path]; ok {
					mark(target)
				}
			}
		}
	}
}
