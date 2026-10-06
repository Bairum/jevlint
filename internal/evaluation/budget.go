package evaluation

import (
	"fmt"
	"strings"

	"github.com/codegirl-007/jevlint/internal/parsing"
)

// Documented Jev 1.13 limits. They sit 743 and 1,533 tokens under the hard
// caps measured live (32,743 state+longest question, 65,533 total), and the
// default estimator (jevtok) reproduces usage.input_tokens exactly, so no
// further margin is applied.
const (
	jev13TotalLimit = 64000
	jev13StateLimit = 32000
)

func limitsFor(model string) (total int, stateLongest int, ok bool) {
	if strings.HasPrefix(model, "jev-1.13") || strings.HasPrefix(model, "typesafe/jev-1.13") {
		return jev13TotalLimit, jev13StateLimit, true
	}
	return 0, 0, false
}

// planRequests splits questions to the provider cap and, when a known model
// has a budget, to the token limits. Over budget it drops callees, then types.
func (client *Client) planRequests(batch Batch) ([][]byte, []ContextDrop, *Oversized, error) {
	state := stateFor(batch)
	questions, err := questionsFor(batch, state)
	if err != nil {
		return nil, nil, nil, err
	}
	bodies, fits, err := client.pack(questions, state)
	if err != nil {
		return nil, nil, nil, err
	}
	if fits {
		return bodies, nil, nil, nil
	}

	var dropped []ContextDrop
	if len(state.Callees) > 0 {
		state.Callees = nil
		dropped = append(dropped, contextDrop(batch.CodeUnit, "callees"))
		questions, err = questionsFor(batch, state)
		if err != nil {
			return nil, nil, nil, err
		}
		bodies, fits, err = client.pack(questions, state)
		if err != nil {
			return nil, nil, nil, err
		}
		if fits {
			return bodies, dropped, nil, nil
		}
	}
	if len(state.RelatedTypes) > 0 {
		state.RelatedTypes = nil
		dropped = append(dropped, contextDrop(batch.CodeUnit, "types"))
		questions, err = questionsFor(batch, state)
		if err != nil {
			return nil, nil, nil, err
		}
		bodies, fits, err = client.pack(questions, state)
		if err != nil {
			return nil, nil, nil, err
		}
		if fits {
			return bodies, dropped, nil, nil
		}
	}
	tokens, limit := client.oversizeDetail(questions, state)
	return nil, nil, &Oversized{
		Path:   batch.CodeUnit.Path,
		Name:   batch.CodeUnit.Name,
		Kind:   batch.CodeUnit.Kind.String(),
		Tokens: tokens,
		Limit:  limit,
	}, nil
}

func contextDrop(unit parsing.CodeUnit, dropped string) ContextDrop {
	return ContextDrop{
		Path:    unit.Path,
		Name:    unit.Name,
		Kind:    unit.Kind.String(),
		Dropped: dropped,
	}
}

// pack returns request bodies that satisfy the question cap and token budget.
// fits is false when even one question exceeds the token budget.
func (client *Client) pack(
	questions map[string]question,
	state requestState,
) ([][]byte, bool, error) {
	ids := sortedQuestionIDs(questions)
	if len(ids) == 0 {
		return nil, false, fmt.Errorf("at least one question is required")
	}
	cap := client.provider.MaxQuestions()
	if cap <= 0 || cap > len(ids) {
		cap = len(ids)
	}
	var bodies [][]byte
	for start := 0; start < len(ids); {
		best := 0
		var bestBody []byte
		limit := start + cap
		if limit > len(ids) {
			limit = len(ids)
		}
		for end := start + 1; end <= limit; end++ {
			subset := subsetQuestions(ids[start:end], questions)
			if err := client.provider.ValidateQuestions(subset); err != nil {
				return nil, false, err
			}
			body, err := marshalRequest(client.model, state, subset)
			if err != nil {
				return nil, false, err
			}
			if max := client.provider.MaxBodyBytes(); max > 0 && len(body) > max {
				if end == start+1 {
					return nil, false, fmt.Errorf(
						"request body is %d bytes, over the %d-byte limit",
						len(body),
						max,
					)
				}
				break
			}
			fits, err := client.fitsBudget(body)
			if err != nil {
				return nil, false, err
			}
			if !fits {
				break
			}
			best = end
			bestBody = body
		}
		if best == 0 {
			return nil, false, nil
		}
		bodies = append(bodies, bestBody)
		start = best
	}
	return bodies, true, nil
}

func (client *Client) fitsBudget(body []byte) (bool, error) {
	if client.budget == nil {
		return true, nil
	}
	totalLimit, stateLimit, known := limitsFor(client.model)
	if !known {
		return true, nil
	}
	estimate, err := client.budget(body)
	if err != nil {
		return false, fmt.Errorf("estimate request: %w", err)
	}
	if estimate.Total > totalLimit || estimate.StateAndLongestQuestion > stateLimit {
		return false, nil
	}
	return true, nil
}

func (client *Client) oversizeDetail(questions map[string]question, state requestState) (int, int) {
	totalLimit, stateLimit, known := limitsFor(client.model)
	if !known || client.budget == nil {
		return 0, 0
	}
	ids := sortedQuestionIDs(questions)
	if len(ids) == 0 {
		return 0, totalLimit
	}
	body, err := marshalRequest(client.model, state, subsetQuestions(ids[:1], questions))
	if err != nil {
		return 0, totalLimit
	}
	estimate, err := client.budget(body)
	if err != nil {
		return 0, totalLimit
	}
	if estimate.Total > totalLimit {
		return estimate.Total, totalLimit
	}
	return estimate.StateAndLongestQuestion, stateLimit
}
