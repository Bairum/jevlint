package evaluation

import (
	"context"
	"encoding/json"
	"fmt"

	"jevlint/internal/config"
	"jevlint/internal/parsing"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusPass
	StatusFail
	StatusSkip
	StatusAbstain
)

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

func (status Status) MarshalJSON() ([]byte, error) {
	if status != StatusPass &&
		status != StatusFail &&
		status != StatusSkip &&
		status != StatusAbstain {
		return nil, fmt.Errorf("invalid evaluation status %d", status)
	}
	return json.Marshal(status.String())
}

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

type Batch struct {
	Rules    []config.Rule    `json:"rules"`
	CodeUnit parsing.CodeUnit `json:"codeUnit"`
}

type Result struct {
	Status     Status  `json:"status"`
	Confidence float64 `json:"confidence"`
}

type Evaluator interface {
	Evaluate(context.Context, Batch) (map[string]Result, error)
}

type CacheStats struct {
	Hits   uint64 `json:"hits"`
	Misses uint64 `json:"misses"`
	Writes uint64 `json:"writes"`
}

type CacheStatsProvider interface {
	CacheStats() CacheStats
}

func (result Result) Validate() error {
	switch result.Status {
	case StatusPass, StatusFail, StatusSkip, StatusAbstain:
	default:
		return fmt.Errorf("invalid evaluation status %q", result.Status.String())
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}
