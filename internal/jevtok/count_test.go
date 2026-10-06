package jevtok

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCountFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/jev.json")
	if err != nil {
		t.Fatal(err)
	}
	var fix struct {
		Cases []struct {
			Text  string `json:"text"`
			Count int    `json:"count"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fix); err != nil {
		t.Fatal(err)
	}
	if len(fix.Cases) != 400 {
		t.Fatalf("fixture cases = %d, want 400", len(fix.Cases))
	}
	matched := 0
	for _, c := range fix.Cases {
		got := Count(c.Text)
		if got == c.Count {
			matched++
			continue
		}
		t.Errorf("Count(%q) = %d, want %d", trunc(c.Text), got, c.Count)
	}
	t.Logf("exact match %d/%d", matched, len(fix.Cases))
	if matched != len(fix.Cases) {
		t.Fatalf("exact match %d/%d", matched, len(fix.Cases))
	}
}

func TestWholeWordsOnlyWholePieces(t *testing.T) {
	if got := Count("token"); got != 1 {
		t.Fatalf("token = %d", got)
	}
	if got := Count("tokenize"); got != 3 {
		t.Fatalf("tokenize = %d", got)
	}
	if got := Count(" information"); got != 1 {
		t.Fatalf(" information = %d", got)
	}
	if got := Count("xinformation"); got != 3 {
		t.Fatalf("xinformation = %d", got)
	}
}

func trunc(s string) string {
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
