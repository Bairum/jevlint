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
	answerTypeNoul         = "noul"
	answerTypeScore        = "score"
	maxLocalizationRegions = 24
	defaultScoreQuestion   = "How clearly does `source` violate `rule`?"
	defaultLevelFollows    = "`source` follows `rule`, `rule` does not apply to it, or one of `exceptions` applies."
	defaultLevelUnclear    = "It is unclear from `source` whether `rule` is violated."
	defaultLevelViolates   = "`source` clearly violates `rule` and none of `exceptions` applies."
)

// QuestionAnswer is one parsed per-question answer stored in the cache.
type QuestionAnswer struct {
	Type  string   `json:"type"`
	Noul  *float64 `json:"noul,omitempty"`
	Score *float64 `json:"score,omitempty"`
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
	Type  string   `json:"type"`
	Noul  *float64 `json:"noul"`
	Score *float64 `json:"score"`
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
		for id, item := range ruleQuestions(rule, state) {
			if _, exists := questions[id]; exists {
				return nil, fmt.Errorf("duplicate question id %q", id)
			}
			questions[id] = item
		}
	}
	return questions, nil
}

func ruleQuestions(rule config.Rule, state requestState) map[string]question {
	questions := make(map[string]question)
	for index, wording := range scoreWordings(rule) {
		id := fmt.Sprintf("%s.s%d", rule.ID, index)
		questions[id] = scoreQuestion(rule, wording.Question, wording.Levels, state, rule.Score == nil && rule.Checks == nil)
	}
	if rule.Checks == nil {
		return questions
	}
	for index, subject := range rule.Checks.Subject {
		questions[fmt.Sprintf("%s.sub%d", rule.ID, index)] = noulQuestion(rule, subject, state)
	}
	for index, violation := range rule.Checks.Violation {
		questions[fmt.Sprintf("%s.v%d.w0", rule.ID, index)] = noulQuestion(rule, config.CheckWording{
			Question: violation.Question, Yes: violation.Yes, No: violation.No,
		}, state)
		for wording, paraphrase := range violation.Paraphrases {
			questions[fmt.Sprintf("%s.v%d.w%d", rule.ID, index, wording+1)] = noulQuestion(rule, paraphrase, state)
		}
	}
	return questions
}

func scoreWordings(rule config.Rule) []config.ScoreWording {
	if rule.Score == nil {
		return []config.ScoreWording{{
			Question: defaultScoreQuestion,
			Levels:   []string{defaultLevelFollows, defaultLevelUnclear, defaultLevelViolates},
		}}
	}
	wordings := make([]config.ScoreWording, 0, 1+len(rule.Score.Paraphrases))
	wordings = append(wordings, config.ScoreWording{Question: rule.Score.Question, Levels: rule.Score.Levels})
	wordings = append(wordings, rule.Score.Paraphrases...)
	return wordings
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
		text := fmt.Sprintf("Does `regions.r%d` itself violate `rule`?", index)
		if len(rule.Exceptions) > 0 {
			text = fmt.Sprintf("Does `regions.r%d` itself violate `rule`, with none of `exceptions` applying?", index)
		}
		text += " `source` is surrounding context only."
		questions[id] = question{
			Type: answerTypeNoul,
			Instructions: questionInstructions{
				Rule:       rule.Description,
				Exceptions: rule.Exceptions,
				Guidance:   rule.Guidance,
				Question:   text,
			},
		}
	}
	if len(state.Regions) != len(batch.Regions) {
		return nil, errors.New("localization state is missing regions")
	}
	return questions, nil
}

func scoreQuestion(rule config.Rule, text string, levels []string, state requestState, exceptions bool) question {
	item := question{
		Type:         answerTypeScore,
		Instructions: explicitInstructions(rule, strings.TrimRight(text, " \t\r\n")+" Judge only `source`."+scopeSentences(state)),
		Criteria:     append([]string(nil), levels...),
	}
	if exceptions && len(rule.Exceptions) > 0 {
		item.Instructions.Exceptions = append([]string(nil), rule.Exceptions...)
	}
	return item
}

func noulQuestion(rule config.Rule, wording config.CheckWording, state requestState) question {
	item := question{
		Type:         answerTypeNoul,
		Instructions: explicitInstructions(rule, strings.TrimRight(wording.Question, " \t\r\n")+" Judge only `source`."+scopeSentences(state)),
	}
	if wording.Yes != "" || wording.No != "" {
		item.Criteria = noulCriteria{True: wording.Yes, False: wording.No}
	}
	return item
}

func explicitInstructions(rule config.Rule, questionText string) questionInstructions {
	return questionInstructions{
		Rule:     rule.Description,
		Guidance: rule.Guidance,
		Question: questionText,
	}
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
			Type:  answer.Type,
			Noul:  answer.Noul,
			Score: answer.Score,
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
	wordings := scoreWordings(rule)
	scoreValues := make([]float64, len(wordings))
	used := make(map[string]QuestionAnswer, len(wordings))
	for index := range wordings {
		id := fmt.Sprintf("%s.s%d", rule.ID, index)
		value, answer, err := scoreAnswer(answers, id)
		if err != nil {
			return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
		}
		scoreValues[index] = value / 2
		used[id] = answer
	}
	scoreMean := mean(scoreValues)
	checksMean := scoreMean
	subjectGate := true
	if rule.Checks != nil {
		gate, err := subjectGateValue(answers, rule, used)
		if err != nil {
			return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
		}
		subjectGate = gate
		violation, err := maxViolation(answers, rule, used)
		if err != nil {
			return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
		}
		checksMean = 0
		if subjectGate {
			checksMean = violation
		}
	}
	failProbability := math.Min(scoreMean, checksMean)
	status := StatusPass
	if failProbability >= 0.5 {
		status = StatusFail
	}
	result := Result{
		Status:          status,
		FailProbability: failProbability,
		Score:           scoreMean,
		Checks:          checksMean,
		SubjectGate:     subjectGate,
		Answers:         used,
		Model:           model,
	}
	if err := result.Validate(); err != nil {
		return Result{}, fmt.Errorf("decode response: rule %q: %w", rule.ID, err)
	}
	return result, nil
}

func scoreAnswer(answers map[string]QuestionAnswer, id string) (float64, QuestionAnswer, error) {
	answer, ok := answers[id]
	if !ok {
		return 0, QuestionAnswer{}, fmt.Errorf("answer for %q is missing", id)
	}
	if answer.Type != answerTypeScore || answer.Score == nil {
		return 0, QuestionAnswer{}, fmt.Errorf("score for %q is missing", id)
	}
	value := *answer.Score
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 2 {
		return 0, QuestionAnswer{}, fmt.Errorf("score for %q must be between 0 and 2", id)
	}
	return value, answer, nil
}

func subjectGateValue(answers map[string]QuestionAnswer, rule config.Rule, used map[string]QuestionAnswer) (bool, error) {
	if len(rule.Checks.Subject) == 0 {
		return true, nil
	}
	minimum := 1.0
	for index := range rule.Checks.Subject {
		id := fmt.Sprintf("%s.sub%d", rule.ID, index)
		value, answer, err := noulAnswer(answers, id)
		if err != nil {
			return false, err
		}
		used[id] = answer
		if value < minimum {
			minimum = value
		}
	}
	return minimum >= 0.5, nil
}

func maxViolation(answers map[string]QuestionAnswer, rule config.Rule, used map[string]QuestionAnswer) (float64, error) {
	maximum := 0.0
	for index, violation := range rule.Checks.Violation {
		ids := []string{fmt.Sprintf("%s.v%d.w0", rule.ID, index)}
		for wording := range violation.Paraphrases {
			ids = append(ids, fmt.Sprintf("%s.v%d.w%d", rule.ID, index, wording+1))
		}
		values := make([]float64, len(ids))
		for wording, id := range ids {
			value, answer, err := noulAnswer(answers, id)
			if err != nil {
				return 0, err
			}
			used[id] = answer
			values[wording] = value
		}
		if averaged := mean(values); averaged > maximum {
			maximum = averaged
		}
	}
	return maximum, nil
}

func noulAnswer(answers map[string]QuestionAnswer, id string) (float64, QuestionAnswer, error) {
	answer, ok := answers[id]
	if !ok {
		return 0, QuestionAnswer{}, fmt.Errorf("answer for %q is missing", id)
	}
	if answer.Type != answerTypeNoul || answer.Noul == nil || !validProbability(*answer.Noul) {
		return 0, QuestionAnswer{}, fmt.Errorf("noul for %q is missing or invalid", id)
	}
	return *answer.Noul, answer, nil
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
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

func validProbability(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func (answer QuestionAnswer) validate() error {
	switch answer.Type {
	case answerTypeScore:
		if answer.Score == nil {
			return errors.New("score is missing")
		}
		value := *answer.Score
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 2 {
			return errors.New("invalid score")
		}
		return nil
	case answerTypeNoul:
		if answer.Noul == nil || !validProbability(*answer.Noul) {
			return errors.New("invalid noul")
		}
		return nil
	default:
		return fmt.Errorf("unsupported answer type %q", answer.Type)
	}
}
