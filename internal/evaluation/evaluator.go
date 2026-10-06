package evaluation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

// Status is the answer for one rule on one piece of code.
type Status int

const (
	StatusUnknown Status = iota
	StatusPass
	StatusFail
	StatusSkip
	StatusAbstain
)

// Batch is one piece of code and the rules to check against it.
// Regions, when set, asks one noul per region instead of rule questions.
type Batch struct {
	Rules    []config.Rule    `json:"rules"`
	CodeUnit parsing.CodeUnit `json:"codeUnit"`
	Regions  []parsing.Region `json:"regions,omitempty"`
}

// Result is the answer and fail probability for one rule.
type Result struct {
	Status          Status                    `json:"status"`
	FailProbability float64                   `json:"failProbability"`
	Score           float64                   `json:"score"`
	Checks          float64                   `json:"checks"`
	SubjectGate     bool                      `json:"subjectGate"`
	Answers         map[string]QuestionAnswer `json:"answers,omitempty"`
	Model           string                    `json:"model,omitempty"`
}

// Evaluator answers a batch of rules for a piece of code.
type Evaluator interface {
	Evaluate(context.Context, Batch) (map[string]Result, error)
}

// CacheStats counts cache hits, misses, and writes.
type CacheStats struct {
	Hits   uint64 `json:"hits"`
	Misses uint64 `json:"misses"`
	Writes uint64 `json:"writes"`
}

// CacheStatsProvider exposes cache counts.
type CacheStatsProvider interface {
	CacheStats() CacheStats
}

// RunMetaProvider exposes models, usage, and budget fallout.
type RunMetaProvider interface {
	RunMeta() RunMeta
}

// String returns the status name.
func (status Status) String() string {
	switch status {
	case StatusPass:
		return "pass"
	case StatusFail:
		return "fail"
	case StatusSkip:
		return "skip"
	case StatusAbstain:
		return "abstain"
	default:
		return "unknown"
	}
}

// MarshalJSON writes the status as its name.
func (status Status) MarshalJSON() ([]byte, error) {
	if status != StatusPass &&
		status != StatusFail &&
		status != StatusSkip &&
		status != StatusAbstain {
		return nil, fmt.Errorf("invalid evaluation status %d", status)
	}
	return json.Marshal(status.String())
}

// UnmarshalJSON reads a status from its name.
func (status *Status) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := ParseStatus(value)
	if err != nil {
		return err
	}
	*status = parsed
	return nil
}

// ParseStatus turns a status name into a status value.
func ParseStatus(value string) (Status, error) {
	switch value {
	case "pass":
		return StatusPass, nil
	case "fail":
		return StatusFail, nil
	case "skip":
		return StatusSkip, nil
	case "abstain":
		return StatusAbstain, nil
	default:
		return StatusUnknown, fmt.Errorf("invalid evaluation status %q", value)
	}
}

// Validate checks that a result has a known status and a valid fail probability.
func (result Result) Validate() error {
	switch result.Status {
	case StatusPass, StatusFail, StatusSkip, StatusAbstain:
	default:
		return fmt.Errorf("invalid evaluation status %q", result.Status.String())
	}
	if result.FailProbability < 0 || result.FailProbability > 1 {
		return fmt.Errorf("failProbability must be between 0 and 1")
	}
	if result.Score < 0 || result.Score > 1 || result.Checks < 0 || result.Checks > 1 {
		return fmt.Errorf("score and checks must be between 0 and 1")
	}
	return nil
}
