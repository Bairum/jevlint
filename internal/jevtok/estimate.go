package jevtok

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Estimate is the provider's input-token accounting for one /v1/systemone body.
// Total matches usage.input_tokens. StateAndLongestQuestion is state plus the
// longest question, excluding the shared frame: the 32k-limit quantity.
// Live hard caps (jev-1.13.0, 2026-10-06) are 32743 and 65533 on that quantity
// and on state+all questions; 32744 and 65534 return 400
// {"detail":{"error_type":"max_tokens_exceeded"}}. Documented 32000/64000 sit under those.
type Estimate struct {
	Total                   int
	StateAndLongestQuestion int
}

// EstimateRequest fits live jev-1.13.0 usage.input_tokens.
//
// The provider parses JSON and counts each string with Count, plus structural
// tokens that do not depend on key order: an object field is Count(key)+Count(value)+5
// (+3 when the value is a number, bool, or null); an array element is Count(value)+2
// (scalars +0); each extra sibling adds 1. Empty containers cost 0. A string state
// is just Count(state).
//
// Total = 259 + state + sum(question). A noul is 7 + instructions, and its criteria
// counts only "true"/"false", plus 5 if exactly one of those keys is set and 3 if
// both. A choice is 18 + instructions + criteria + 4 per option. A score is 14 +
// instructions + criteria + 2 per level. Question ids are not counted.
func EstimateRequest(body []byte) (Estimate, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return Estimate{}, fmt.Errorf("jevtok: request: %w", err)
	}
	stateRaw, ok := raw["state"]
	if !ok {
		return Estimate{}, fmt.Errorf("jevtok: request missing state")
	}
	qRaw, ok := raw["questions"]
	if !ok {
		return Estimate{}, fmt.Errorf("jevtok: request missing questions")
	}
	state, err := decodeValue(stateRaw)
	if err != nil {
		return Estimate{}, err
	}
	var questions map[string]json.RawMessage
	if err := json.Unmarshal(qRaw, &questions); err != nil {
		return Estimate{}, fmt.Errorf("jevtok: questions: %w", err)
	}
	if len(questions) == 0 {
		return Estimate{}, fmt.Errorf("jevtok: request has no questions")
	}
	st := countValue(state)
	sum, longest := 0, 0
	for id, rawQ := range questions {
		cost, err := questionCost(rawQ)
		if err != nil {
			return Estimate{}, fmt.Errorf("jevtok: question %s: %w", id, err)
		}
		sum += cost
		if cost > longest {
			longest = cost
		}
	}
	return Estimate{Total: frameTokens + st + sum, StateAndLongestQuestion: st + longest}, nil
}

// frameTokens is the shared request frame measured on a minimal noul
// (empty state + "Yes?" = 268, of which 2 are the instruction and 7 the noul).
const frameTokens = 259

func questionCost(raw json.RawMessage) (int, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var q map[string]any
	if err := dec.Decode(&q); err != nil {
		return 0, err
	}
	typ, _ := q["type"].(string)
	instr := 0
	if v, ok := q["instructions"]; ok {
		instr = countValue(v)
	}
	crit, hasCrit := q["criteria"]
	switch typ {
	case "noul":
		extra := 7
		if hasCrit {
			if m, ok := crit.(map[string]any); ok {
				filtered := make(map[string]any, 2)
				if v, ok := m["true"]; ok {
					filtered["true"] = v
				}
				if v, ok := m["false"]; ok {
					filtered["false"] = v
				}
				instr += countValue(filtered)
				switch len(filtered) {
				case 1:
					extra += 5
				case 2:
					extra += 3
				}
			} else {
				instr += countValue(crit)
			}
		}
		return instr + extra, nil
	case "choice":
		n := 0
		cc := 0
		if hasCrit {
			cc = countValue(crit)
			if m, ok := crit.(map[string]any); ok {
				n = len(m)
			}
		}
		return instr + cc + 18 + 4*n, nil
	case "score":
		n := 0
		cc := 0
		if hasCrit {
			cc = countValue(crit)
			if a, ok := crit.([]any); ok {
				n = len(a)
			}
		}
		return instr + cc + 14 + 2*n, nil
	default:
		return 0, fmt.Errorf("unknown type %q", typ)
	}
}

func decodeValue(raw json.RawMessage) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("jevtok: state: %w", err)
	}
	return v, nil
}

func countValue(v any) int {
	switch x := v.(type) {
	case nil:
		return Count("null")
	case string:
		return Count(x)
	case json.Number:
		return Count(x.String())
	case bool:
		if x {
			return Count("true")
		}
		return Count("false")
	case []any:
		return countArray(x)
	case map[string]any:
		return countObject(x)
	default:
		return 0
	}
}

func countArray(a []any) int {
	if len(a) == 0 {
		return 0
	}
	n := len(a) - 1
	for _, el := range a {
		n += countElem(el)
	}
	return n
}

func countObject(m map[string]any) int {
	if len(m) == 0 {
		return 0
	}
	n := len(m) - 1
	for k, v := range m {
		n += countField(k, v)
	}
	return n
}

func countElem(v any) int {
	switch x := v.(type) {
	case string:
		return Count(x) + 2
	case json.Number:
		return Count(x.String())
	case bool:
		if x {
			return Count("true")
		}
		return Count("false")
	case nil:
		return Count("null")
	default:
		return countValue(v) + 2
	}
}

func countField(k string, v any) int {
	switch x := v.(type) {
	case string:
		return Count(k) + Count(x) + 5
	case json.Number:
		return Count(k) + Count(x.String()) + 3
	case bool:
		if x {
			return Count(k) + Count("true") + 3
		}
		return Count(k) + Count("false") + 3
	case nil:
		return Count(k) + Count("null") + 3
	default:
		return Count(k) + countValue(v) + 5
	}
}
