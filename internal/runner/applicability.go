package runner

import (
	"fmt"
	"regexp"

	"github.com/codegirl-007/jevlint/internal/config"
)

type sourceMatchers map[string][]*regexp.Regexp

func compileSourceMatches(rules []config.Rule) (sourceMatchers, error) {
	compiled := make(sourceMatchers)
	for _, rule := range rules {
		if len(rule.SourceMatch) == 0 {
			continue
		}
		patterns := make([]*regexp.Regexp, 0, len(rule.SourceMatch))
		for _, pattern := range rule.SourceMatch {
			matcher, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("rule %q sourceMatch: %w", rule.ID, err)
			}
			patterns = append(patterns, matcher)
		}
		compiled[rule.ID] = patterns
	}
	return compiled, nil
}

func (compiled sourceMatchers) matches(ruleID, source string) bool {
	patterns := compiled[ruleID]
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if pattern.MatchString(source) {
			return true
		}
	}
	return false
}
