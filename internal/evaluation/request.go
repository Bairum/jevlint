package evaluation

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

// ErrOversized means the unit still exceeded the model input budget after
// splitting questions and dropping context. The run should continue.
var ErrOversized = errors.New("evaluation unit exceeds the model input budget")

const (
	answerTypeChoice = "choice"
	answerTypeNoul   = "noul"
	// probabilitySumTolerance accepts distributions that round to 1.
	probabilitySumTolerance = 0.02
	maxLocalizationRegions  = 24
)

// QuestionAnswer is one parsed per-question answer stored in the cache.
type QuestionAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
}

// Usage counts provider calls in one run. Cache hits add nothing.
type Usage struct {
	Requests     int `json:"requests"`
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

// Estimate is the token estimate of a serialized /v1/systemone body.
// Field names match jevtok.Estimate so the parent can wrap EstimateRequest
// without this package importing internal/jevtok.
type Estimate struct {
	Total                   int
	StateAndLongestQuestion int
}

// Oversized is a unit skipped because it still exceeded the input budget
// after splitting and dropping context.
type Oversized struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Tokens int    `json:"tokens"`
	Limit  int    `json:"limit"`
}

// ContextDrop records context removed so a unit fit the input budget.
type ContextDrop struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Dropped string `json:"dropped"`
}

// RunMeta is the model, usage, and budget fallout of a run.
type RunMeta struct {
	Models         []string
	Usage          Usage
	Oversized      []Oversized
	ContextDropped []ContextDrop
}

type question struct {
	Type         string               `json:"type"`
	Instructions questionInstructions `json:"instructions"`
	Criteria     any                  `json:"criteria,omitempty"`
}

type questionInstructions struct {
	Rule       string   `json:"rule"`
	Exceptions []string `json:"exceptions,omitempty"`
	Guidance   string   `json:"guidance,omitempty"`
	Question   string   `json:"question"`
}

type choiceCriteria struct {
	Pass    string `json:"pass"`
	Fail    string `json:"fail"`
	Skip    string `json:"skip,omitempty"`
	Abstain string `json:"abstain,omitempty"`
}

type noulCriteria struct {
	True  string `json:"true,omitempty"`
	False string `json:"false,omitempty"`
}

type requestState struct {
	Kind         parsing.CodeKind       `json:"kind"`
	Name         string                 `json:"name"`
	Language     parsing.SourceLanguage `json:"language"`
	Path         string                 `json:"path"`
	Source       string                 `json:"source"`
	ParentSource string                 `json:"parentSource,omitempty"`
	RegionKind   parsing.NodeKind       `json:"regionKind,omitempty"`
	RelatedTypes []requestType          `json:"types,omitempty"`
	Callees      []requestCallee        `json:"callees,omitempty"`
	Regions      map[string]string      `json:"regions,omitempty"`
}

type requestType struct {
	Name   string `json:"name"`
	Path   string `json:"path,omitempty"`
	Source string `json:"source"`
}

type requestCallee struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

type systemOneRequest struct {
	Model     string              `json:"model"`
	State     requestState        `json:"state"`
	Questions map[string]question `json:"questions"`
}

type serviceResponse struct {
	Model   string
	Answers map[string]QuestionAnswer
	Usage   Usage
}

type wireResponse struct {
	Model   string          `json:"model"`
	Answers json.RawMessage `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Result json.RawMessage `json:"result"`
}

type wireAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Noul          *float64           `json:"noul"`
}

// questionsFor builds the questions for a batch against the state that will
// actually be sent. Scope sentences follow that state, so dropping context
// rebuilds them.
func questionsFor(batch Batch, state requestState) (map[string]question, error) {
	if len(batch.Regions) > 0 {
		return localizationQuestions(batch, state)
	}
	if len(batch.Rules) < 1 {
		return nil, errors.New("at least one rule is required")
	}
	questions := make(map[string]question)
	for _, rule := range batch.Rules {
		if err := config.ValidateChecks(rule, rule.ID); err != nil {
			return nil, err
		}
		if len(rule.Checks) > 0 {
			for _, check := range rule.Checks {
				id := rule.ID + "." + check.ID
				if _, exists := questions[id]; exists {
					return nil, fmt.Errorf("duplicate question id %q", id)
				}
				questions[id] = noulCheckQuestion(rule, check, state)
			}
			continue
		}
		if _, exists := questions[rule.ID]; exists {
			return nil, fmt.Errorf("duplicate rule id %q in evaluation batch", rule.ID)
		}
		questions[rule.ID] = choiceQuestion(rule, state)
	}
	return questions, nil
}

func localizationQuestions(batch Batch, state requestState) (map[string]question, error) {
	if len(batch.Rules) != 1 {
		return nil, errors.New("localization requires exactly one rule")
	}
	if len(batch.Regions) > maxLocalizationRegions {
		return nil, fmt.Errorf("localization accepts at most %d regions", maxLocalizationRegions)
	}
	rule := batch.Rules[0]
	questions := make(map[string]question, len(batch.Regions))
	for index := range batch.Regions {
		id := fmt.Sprintf("%s.r%d", rule.ID, index)
		questions[id] = question{
			Type: answerTypeNoul,
			Instructions: questionInstructions{
				Rule:       rule.Description,
				Exceptions: rule.Exceptions,
				Guidance:   rule.Guidance,
				Question: fmt.Sprintf(
					"Does `regions.r%d` itself violate `rule`? `source` is surrounding context only.",
					index,
				),
			},
		}
	}
	if len(state.Regions) != len(batch.Regions) {
		return nil, errors.New("localization state is missing regions")
	}
	return questions, nil
}

func choiceQuestion(rule config.Rule, state requestState) question {
	questionText := "Does `source` violate `rule`"
	pass := "`source` follows `rule`"
	fail := "`source` violates `rule`"
	if len(rule.Exceptions) > 0 {
		questionText += ", with none of `exceptions` applying"
		pass += ", or one of `exceptions` applies"
		fail += " and none of `exceptions` applies"
	}
	questionText += "? Judge only `source`." + scopeSentences(state)
	pass += "."
	fail += "."
	criteria := choiceCriteria{Pass: pass, Fail: fail}
	if rule.AllowSkip {
		criteria.Skip = "`rule`'s subject is not present in `source`; the rule does not apply."
	}
	if rule.AllowAbstain {
		criteria.Abstain = "`rule` is relevant, but the supplied state does not contain enough context to decide pass or fail."
	}
	return question{
		Type: answerTypeChoice,
		Instructions: questionInstructions{
			Rule:       rule.Description,
			Exceptions: rule.Exceptions,
			Guidance:   rule.Guidance,
			Question:   questionText,
		},
		Criteria: criteria,
	}
}

func noulCheckQuestion(rule config.Rule, check config.Check, state requestState) question {
	item := question{
		Type: answerTypeNoul,
		Instructions: questionInstructions{
			Rule:     rule.Description,
			Guidance: rule.Guidance,
			Question: check.Question + " Judge only `source`." + scopeSentences(state),
		},
	}
	if check.Yes != "" || check.No != "" {
		item.Criteria = noulCriteria{True: check.Yes, False: check.No}
	}
	return item
}

func scopeSentences(state requestState) string {
	var builder strings.Builder
	if state.ParentSource != "" {
		builder.WriteString(" `parentSource` is surrounding context only.")
	}
	if len(state.Callees) > 0 || len(state.RelatedTypes) > 0 {
		builder.WriteString(" `types` and `callees` are context only; do not fail `source` for problems that exist only in them.")
	}
	return builder.String()
}

func stateFor(batch Batch) requestState {
	unit := batch.CodeUnit
	state := requestState{
		Kind:         unit.Kind,
		Name:         unit.Name,
		Language:     unit.Language,
		Path:         unit.Path,
		Source:       unit.Source,
		ParentSource: unit.ParentSource,
		RegionKind:   unit.RegionKind,
	}
	if len(unit.RelatedTypes) > 0 {
		state.RelatedTypes = make([]requestType, 0, len(unit.RelatedTypes))
		for _, declaration := range unit.RelatedTypes {
			requested := requestType{Name: declaration.Name, Source: declaration.Source}
			if declaration.Path != unit.Path {
				requested.Path = declaration.Path
			}
			state.RelatedTypes = append(state.RelatedTypes, requested)
		}
	}
	if len(unit.Callees) > 0 {
		state.Callees = make([]requestCallee, 0, len(unit.Callees))
		for _, callee := range unit.Callees {
			state.Callees = append(state.Callees, requestCallee{
				Name:   callee.Name,
				Path:   callee.Path,
				Source: callee.Source,
			})
		}
	}
	if len(batch.Regions) > 0 {
		state.Regions = make(map[string]string, len(batch.Regions))
		for index, region := range batch.Regions {
			state.Regions[fmt.Sprintf("r%d", index)] = region.Source
		}
	}
	return state
}

func marshalRequest(model string, state requestState, questions map[string]question) ([]byte, error) {
	body, err := json.Marshal(systemOneRequest{
		Model:     model,
		State:     state,
		Questions: questions,
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	return body, nil
}

func subsetQuestions(ids []string, questions map[string]question) map[string]question {
	subset := make(map[string]question, len(ids))
	for _, id := range ids {
		subset[id] = questions[id]
	}
	return subset
}

func sortedQuestionIDs(questions map[string]question) []string {
	ids := make([]string, 0, len(questions))
	for id := range questions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func parseServiceResponse(body []byte) (serviceResponse, error) {
	var wire wireResponse
	if err := json.Unmarshal(body, &wire); err != nil {
		return serviceResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if len(wire.Answers) == 0 && len(wire.Result) > 0 {
		if err := json.Unmarshal(wire.Result, &wire); err != nil {
			return serviceResponse{}, fmt.Errorf("decode response: %w", err)
		}
	}
	if len(wire.Answers) == 0 {
		return serviceResponse{}, errors.New("decode response: answers are missing")
	}
	var raw map[string]wireAnswer
	if err := json.Unmarshal(wire.Answers, &raw); err != nil {
		return serviceResponse{}, fmt.Errorf("decode response: %w", err)
	}
	answers := make(map[string]QuestionAnswer, len(raw))
	for id, answer := range raw {
		parsed := QuestionAnswer{
			Type:          answer.Type,
			Choice:        answer.Choice,
			Probabilities: answer.Probabilities,
			Noul:          answer.Noul,
		}
		answers[id] = parsed
	}
	return serviceResponse{
		Model:   wire.Model,
		Answers: answers,
		Usage: Usage{
			InputTokens:  wire.Usage.InputTokens,
			OutputTokens: wire.Usage.OutputTokens,
		},
	}, nil
}

func decodeAnswers(answers map[string]QuestionAnswer, batch Batch, model string) (map[string]Result, error) {
	if len(batch.Regions) > 0 {
		return decodeLocalization(answers, batch.Rules[0], len(batch.Regions), model)
	}
	results := make(map[string]Result, len(batch.Rules))
	for _, rule := range batch.Rules {
		result, err := decodeRule(answers, rule, model)
		if err != nil {
			return nil, err
		}
		results[rule.ID] = result
	}
	return results, nil
}

func decodeRule(answers map[string]QuestionAnswer, rule config.Rule, model string) (Result, error) {
	if len(rule.Checks) > 0 {
		return decodeChecks(answers, rule, model)
	}
	answer, ok := answers[rule.ID]
	if !ok {
		return Result{}, fmt.Errorf("decode response: answer for rule %q is missing", rule.ID)
	}
	if answer.Type != answerTypeChoice {
		return Result{}, fmt.Errorf("decode response: answer for rule %q has unsupported type", rule.ID)
	}
	failProbability, err := failProbabilityFromChoice(answer.Probabilities)
	if err != nil {
		return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
	}
	status, err := argmaxStatus(answer.Probabilities)
	if err != nil {
		return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
	}
	if status == StatusSkip && !rule.AllowSkip {
		return Result{}, fmt.Errorf("decode response: rule %q returned skip without allowSkip", rule.ID)
	}
	if status == StatusAbstain && !rule.AllowAbstain {
		return Result{}, fmt.Errorf("decode response: rule %q returned abstain without allowAbstain", rule.ID)
	}
	result := Result{
		Status:          status,
		FailProbability: failProbability,
		Probabilities:   copyProbabilities(answer.Probabilities),
		Model:           model,
	}
	if err := result.Validate(); err != nil {
		return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
	}
	return result, nil
}

func decodeChecks(answers map[string]QuestionAnswer, rule config.Rule, model string) (Result, error) {
	nouls := make([]float64, len(rule.Checks))
	for index, check := range rule.Checks {
		id := rule.ID + "." + check.ID
		answer, ok := answers[id]
		if !ok {
			return Result{}, fmt.Errorf("decode response: answer for check %q is missing", id)
		}
		if answer.Type != answerTypeNoul || answer.Noul == nil {
			return Result{}, fmt.Errorf("decode response: noul for check %q is missing", id)
		}
		if !validProbability(*answer.Noul) {
			return Result{}, fmt.Errorf("decode response: noul for check %q is invalid", id)
		}
		nouls[index] = *answer.Noul
	}
	failProbability := combineChecks(rule.Checks, nouls)
	status := StatusPass
	if failProbability >= 0.5 {
		status = StatusFail
	}
	result := Result{
		Status:          status,
		FailProbability: failProbability,
		Model:           model,
	}
	if err := result.Validate(); err != nil {
		return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
	}
	return result, nil
}

func decodeLocalization(
	answers map[string]QuestionAnswer,
	rule config.Rule,
	count int,
	model string,
) (map[string]Result, error) {
	results := make(map[string]Result, count)
	for index := range count {
		id := fmt.Sprintf("%s.r%d", rule.ID, index)
		answer, ok := answers[id]
		if !ok {
			return nil, fmt.Errorf("decode response: answer for region %q is missing", id)
		}
		if answer.Type != answerTypeNoul || answer.Noul == nil || !validProbability(*answer.Noul) {
			return nil, fmt.Errorf("decode response: noul for region %q is missing or invalid", id)
		}
		status := StatusPass
		if *answer.Noul >= 0.5 {
			status = StatusFail
		}
		results[fmt.Sprintf("r%d", index)] = Result{
			Status:          status,
			FailProbability: *answer.Noul,
			Model:           model,
		}
	}
	return results, nil
}

// combineChecks is the failure probability of a checks rule.
// ponytail: product assumes independent checks; fitted weights if calibration shows dependence.
func combineChecks(checks []config.Check, nouls []float64) float64 {
	probability := 1.0
	for index, check := range checks {
		if check.FailWhen {
			probability *= nouls[index]
		} else {
			probability *= 1 - nouls[index]
		}
	}
	return probability
}

func failProbabilityFromChoice(probabilities map[string]float64) (float64, error) {
	if len(probabilities) == 0 {
		return 0, errors.New("probabilities are missing")
	}
	fail, ok := probabilities["fail"]
	if !ok {
		return 0, errors.New("probabilities missing fail")
	}
	sum := 0.0
	for key, value := range probabilities {
		if !validProbability(value) {
			return 0, fmt.Errorf("invalid probability %q", key)
		}
		sum += value
	}
	if math.Abs(sum-1) > probabilitySumTolerance {
		return 0, fmt.Errorf("probabilities sum to %v", sum)
	}
	return fail, nil
}

func argmaxStatus(probabilities map[string]float64) (Status, error) {
	order := []string{"fail", "pass", "skip", "abstain"}
	known := make(map[string]struct{}, len(order))
	for _, key := range order {
		known[key] = struct{}{}
	}
	extras := make([]string, 0)
	for key := range probabilities {
		if _, ok := known[key]; !ok {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	best := ""
	bestValue := 0.0
	seen := false
	for _, key := range append(order, extras...) {
		value, ok := probabilities[key]
		if !ok {
			continue
		}
		if !seen || value > bestValue {
			best = key
			bestValue = value
			seen = true
		}
	}
	status, err := ParseStatus(best)
	if err != nil {
		return StatusUnknown, fmt.Errorf("invalid evaluation status %q", best)
	}
	return status, nil
}

func validProbability(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func copyProbabilities(probabilities map[string]float64) map[string]float64 {
	if len(probabilities) == 0 {
		return nil
	}
	copied := make(map[string]float64, len(probabilities))
	for key, value := range probabilities {
		copied[key] = value
	}
	return copied
}

func (answer QuestionAnswer) validate() error {
	switch answer.Type {
	case answerTypeChoice:
		_, err := failProbabilityFromChoice(answer.Probabilities)
		return err
	case answerTypeNoul:
		if answer.Noul == nil || !validProbability(*answer.Noul) {
			return errors.New("invalid noul")
		}
		return nil
	default:
		return fmt.Errorf("unsupported answer type %q", answer.Type)
	}
}
