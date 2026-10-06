package evaluation

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/codegirl-007/jevlint/internal/config"
	"github.com/codegirl-007/jevlint/internal/parsing"
)

func TestCombineChecksIsProduct(t *testing.T) {
	checks := []config.Check{
		{ID: "present", FailWhen: true},
		{ID: "allowed", FailWhen: false},
	}
	got := combineChecks(checks, []float64{0.8, 0.25})
	want := 0.8 * (1 - 0.25)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("combineChecks() = %v, want %v", got, want)
	}
}

func TestChoiceRequestShape(t *testing.T) {
	batch := Batch{
		Rules: []config.Rule{{
			ID:          "joins",
			Description: "Join related records in the database.",
			Exceptions:  []string{"a documented exception"},
			Guidance:    "Shared guidance.",
			AllowSkip:   true,
		}},
		CodeUnit: parsing.CodeUnit{
			Kind:         parsing.CodeKindFunction,
			Name:         "Load",
			Language:     parsing.SourceLanguageGo,
			Path:         "db.go",
			Source:       "func Load() {}",
			ParentSource: "package db",
			Callees:      []parsing.CalleeContext{{Name: "query", Path: "q.go", Source: "func query() {}"}},
		},
	}
	state := stateFor(batch)
	questions, err := questionsFor(batch, state)
	if err != nil {
		t.Fatal(err)
	}
	body, err := marshalRequest("jev-1.13.0", state, questions)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		State struct {
			Guidance []string `json:"guidance"`
		} `json:"state"`
		Questions map[string]struct {
			Type         string `json:"type"`
			Instructions struct {
				Rule       string   `json:"rule"`
				Exceptions []string `json:"exceptions"`
				Guidance   string   `json:"guidance"`
				Question   string   `json:"question"`
			} `json:"instructions"`
			Criteria map[string]string `json:"criteria"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.State.Guidance != nil {
		t.Fatalf("state guidance = %#v, want omitted", payload.State.Guidance)
	}
	question := payload.Questions["joins"]
	if question.Type != "choice" || question.Instructions.Guidance != "Shared guidance." {
		t.Fatalf("question = %#v", question)
	}
	if !strings.Contains(question.Instructions.Question, "Does `source` violate `rule`, with none of `exceptions` applying?") {
		t.Fatalf("question text = %q", question.Instructions.Question)
	}
	if !strings.Contains(question.Instructions.Question, "`parentSource` is surrounding context only.") {
		t.Fatalf("question text = %q", question.Instructions.Question)
	}
	if question.Criteria["skip"] == "" || question.Criteria["abstain"] != "" {
		t.Fatalf("criteria = %#v", question.Criteria)
	}
}

func TestChecksRequestOmitsEmptyCriteria(t *testing.T) {
	batch := Batch{
		Rules: []config.Rule{{
			ID:          "split",
			Description: "A decomposed rule.",
			Checks: []config.Check{
				{ID: "present", Question: "Is the subject present?", FailWhen: true},
				{ID: "excused", Question: "Is it excused?", Yes: "yes", No: "no", FailWhen: false},
			},
		}},
		CodeUnit: parsing.CodeUnit{
			Kind: parsing.CodeKindFunction, Name: "Load", Language: parsing.SourceLanguageGo,
			Path: "db.go", Source: "func Load() {}",
		},
	}
	questions, err := questionsFor(batch, stateFor(batch))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := questions["split.present"].Criteria.(noulCriteria); ok || questions["split.present"].Criteria != nil {
		t.Fatalf("empty criteria = %#v", questions["split.present"].Criteria)
	}
	criteria, ok := questions["split.excused"].Criteria.(noulCriteria)
	if !ok || criteria.True != "yes" || criteria.False != "no" {
		t.Fatalf("criteria = %#v", questions["split.excused"].Criteria)
	}
}

func TestDecodeChecksProduct(t *testing.T) {
	rule := config.Rule{
		ID: "split",
		Checks: []config.Check{
			{ID: "present", FailWhen: true},
			{ID: "excused", FailWhen: false},
		},
	}
	yes := 0.9
	no := 0.2
	result, err := decodeRule(map[string]QuestionAnswer{
		"split.present": {Type: answerTypeNoul, Noul: &yes},
		"split.excused": {Type: answerTypeNoul, Noul: &no},
	}, rule, "jev-1.13.0")
	if err != nil {
		t.Fatal(err)
	}
	want := 0.9 * 0.8
	if math.Abs(result.FailProbability-want) > 1e-9 || result.Status != StatusFail {
		t.Fatalf("result = %#v, want fail %v", result, want)
	}
}

func TestDecodeChoiceRequiresFailProbability(t *testing.T) {
	rule := config.Rule{ID: "joins"}
	_, err := decodeRule(map[string]QuestionAnswer{
		"joins": {Type: answerTypeChoice, Choice: "fail"},
	}, rule, "jev-1.13.0")
	if err == nil {
		t.Fatal("missing probabilities decoded")
	}
	result, err := decodeRule(map[string]QuestionAnswer{
		"joins": {Type: answerTypeChoice, Probabilities: map[string]float64{"fail": 0.7, "pass": 0.3}},
	}, rule, "jev-1.13.0")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusFail || result.FailProbability != 0.7 {
		t.Fatalf("result = %#v", result)
	}
}

func TestLocalizationRequestIsOneNoulPerRegion(t *testing.T) {
	batch := Batch{
		Rules: []config.Rule{{
			ID:          "joins",
			Description: "Join related records.",
			Guidance:    "Shared guidance.",
		}},
		CodeUnit: parsing.CodeUnit{
			Kind: parsing.CodeKindFunction, Name: "Load", Language: parsing.SourceLanguageGo,
			Path: "db.go", Source: "func Load() {}",
		},
		Regions: []parsing.Region{
			{Source: "query()"},
			{Source: "join()"},
		},
	}
	state := stateFor(batch)
	questions, err := questionsFor(batch, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 || questions["joins.r0"].Type != answerTypeNoul || questions["joins.r1"].Type != answerTypeNoul {
		t.Fatalf("questions = %#v", questions)
	}
	if state.Regions["r0"] != "query()" || state.Regions["r1"] != "join()" {
		t.Fatalf("regions = %#v", state.Regions)
	}
	if !strings.Contains(questions["joins.r1"].Instructions.Question, "Does `regions.r1` itself violate `rule`?") {
		t.Fatalf("question = %q", questions["joins.r1"].Instructions.Question)
	}
	if questions["joins.r0"].Instructions.Guidance != "Shared guidance." {
		t.Fatalf("guidance = %q", questions["joins.r0"].Instructions.Guidance)
	}
}
