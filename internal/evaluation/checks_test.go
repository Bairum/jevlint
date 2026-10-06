package evaluation

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

func TestDefaultScoreRequestOmitsExceptionsWhenChecksExist(t *testing.T) {
	rule := config.Rule{
		ID:          "joins",
		Description: "Join in the database.",
		Exceptions:  []string{"Different databases."},
		Guidance:    "Shared guidance.",
		Checks: &config.Checks{
			Subject: []config.CheckWording{{Question: "Is this a query?", Yes: "Yes.", No: "No."}},
			Violation: []config.ViolationCheck{{
				Question: "Does source join in memory?", Yes: "Yes.", No: "No.",
				Paraphrases: []config.CheckWording{{Question: "Is the join outside the query?", Yes: "Outside.", No: "Inside."}},
			}},
		},
	}
	body, err := marshalRequest("jev-test", stateFor(Batch{CodeUnit: parsing.CodeUnit{
		Kind: parsing.CodeKindFunction, Name: "Load", Language: parsing.SourceLanguageGo,
		Path: "db.go", Source: "func Load() {}",
	}}), mustQuestions(t, rule))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Questions map[string]struct {
			Type         string               `json:"type"`
			Instructions questionInstructions `json:"instructions"`
			Criteria     json.RawMessage      `json:"criteria"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	score := payload.Questions["joins.s0"]
	if score.Type != answerTypeScore || score.Instructions.Guidance != "Shared guidance." || score.Instructions.Exceptions != nil {
		t.Fatalf("default score = %#v", score)
	}
	if !strings.Contains(score.Instructions.Question, defaultScoreQuestion) || !strings.Contains(score.Instructions.Question, "Judge only `source`.") {
		t.Fatalf("question = %q", score.Instructions.Question)
	}
	if _, ok := payload.Questions["joins.sub0"]; !ok {
		t.Fatal("missing subject question")
	}
	if _, ok := payload.Questions["joins.v0.w0"]; !ok {
		t.Fatal("missing violation question")
	}
	if _, ok := payload.Questions["joins.v0.w1"]; !ok {
		t.Fatal("missing violation paraphrase")
	}
}

func TestDefaultScoreCarriesExceptionsWithoutChecks(t *testing.T) {
	rule := config.Rule{
		ID:          "joins",
		Description: "Join in the database.",
		Exceptions:  []string{"Different databases."},
	}
	questions := mustQuestions(t, rule)
	item := questions["joins.s0"]
	if len(item.Instructions.Exceptions) != 1 {
		t.Fatalf("exceptions = %#v", item.Instructions.Exceptions)
	}
	levels, _ := item.Criteria.([]string)
	if len(levels) != 3 || levels[2] != defaultLevelViolates {
		t.Fatalf("levels = %#v", item.Criteria)
	}
}

func TestExplicitScoreOmitsExceptions(t *testing.T) {
	rule := config.Rule{
		ID:          "joins",
		Description: "Join in the database.",
		Exceptions:  []string{"Different databases."},
		Score: &config.Score{
			Question: "How clearly does `source` join in memory?",
			Levels:   []string{"it does not", "unclear", "it does"},
			Paraphrases: []config.ScoreWording{{
				Question: "How obvious is an in-memory join?",
				Levels:   []string{"absent", "unclear", "obvious"},
			}},
		},
	}
	questions := mustQuestions(t, rule)
	if questions["joins.s0"].Instructions.Exceptions != nil || questions["joins.s1"].Instructions.Exceptions != nil {
		t.Fatalf("explicit score carried exceptions: %#v", questions)
	}
	if !strings.Contains(questions["joins.s0"].Instructions.Question, "How clearly does `source` join in memory?") {
		t.Fatalf("question = %q", questions["joins.s0"].Instructions.Question)
	}
}

func TestDecodeAgreesOnMinOfScoreAndChecks(t *testing.T) {
	rule := config.Rule{
		ID: "joins",
		Checks: &config.Checks{
			Subject: []config.CheckWording{{Question: "In scope?"}},
			Violation: []config.ViolationCheck{{
				Question:    "Broken?",
				Paraphrases: []config.CheckWording{{Question: "Still broken?"}},
			}},
		},
		Score: &config.Score{
			Question:    "How clear?",
			Levels:      []string{"no", "maybe", "yes"},
			Paraphrases: []config.ScoreWording{{Question: "How obvious?", Levels: []string{"no", "maybe", "yes"}}},
		},
	}
	result, err := decodeRule(map[string]QuestionAnswer{
		"joins.s0":    {Type: answerTypeScore, Score: new(2.0)},
		"joins.s1":    {Type: answerTypeScore, Score: new(1.0)},
		"joins.sub0":  {Type: answerTypeNoul, Noul: new(0.8)},
		"joins.v0.w0": {Type: answerTypeNoul, Noul: new(0.9)},
		"joins.v0.w1": {Type: answerTypeNoul, Noul: new(0.7)},
	}, rule, "jev-1.13.0")
	if err != nil {
		t.Fatal(err)
	}
	// S = mean(2/2, 1/2) = 0.75; C = 0.8 gate * mean(0.9, 0.7) = 0.8; min = 0.75
	if math.Abs(result.Score-0.75) > 1e-9 || math.Abs(result.Checks-0.8) > 1e-9 || !result.SubjectGate {
		t.Fatalf("components = %#v", result)
	}
	if result.Status != StatusFail || math.Abs(result.FailProbability-0.75) > 1e-9 {
		t.Fatalf("decision = %#v", result)
	}
}

func TestSubjectGateClosesChecks(t *testing.T) {
	rule := config.Rule{
		ID: "joins",
		Checks: &config.Checks{
			Subject:   []config.CheckWording{{Question: "In scope?"}},
			Violation: []config.ViolationCheck{{Question: "Broken?"}},
		},
	}
	result, err := decodeRule(map[string]QuestionAnswer{
		"joins.s0":    {Type: answerTypeScore, Score: new(2.0)},
		"joins.sub0":  {Type: answerTypeNoul, Noul: new(0.4)},
		"joins.v0.w0": {Type: answerTypeNoul, Noul: new(1.0)},
	}, rule, "jev-1.13.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.SubjectGate || result.Checks != 0 || result.FailProbability != 0 || result.Status != StatusPass {
		t.Fatalf("gated result = %#v", result)
	}
}

func TestChecksAbsentUsesScoreForBothSignals(t *testing.T) {
	rule := config.Rule{ID: "joins"}
	result, err := decodeRule(map[string]QuestionAnswer{
		"joins.s0": {Type: answerTypeScore, Score: new(1.0)},
	}, rule, "jev-1.13.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.Score != 0.5 || result.Checks != 0.5 || !result.SubjectGate || result.Status != StatusFail {
		t.Fatalf("default result = %#v", result)
	}
}

func mustQuestions(t *testing.T, rule config.Rule) map[string]question {
	t.Helper()
	questions, err := questionsFor(Batch{
		Rules:    []config.Rule{rule},
		CodeUnit: parsing.CodeUnit{Kind: parsing.CodeKindFunction, Name: "Load", Path: "db.go", Source: "func Load() {}"},
	}, requestState{Kind: parsing.CodeKindFunction, Name: "Load", Path: "db.go", Source: "func Load() {}"})
	if err != nil {
		t.Fatal(err)
	}
	return questions
}
