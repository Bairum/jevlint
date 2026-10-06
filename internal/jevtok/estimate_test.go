package jevtok

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func TestEstimateRequestFixture(t *testing.T) {
	f, err := os.Open("testdata/requests.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	n, matched, maxAbs := 0, 0, 0
	for sc.Scan() {
		var rec struct {
			Name        string          `json:"name"`
			InputTokens int             `json:"input_tokens"`
			Body        json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatal(err)
		}
		est, err := EstimateRequest(rec.Body)
		if err != nil {
			t.Fatalf("%s: %v", rec.Name, err)
		}
		d := est.Total - rec.InputTokens
		if d < 0 {
			d = -d
		}
		if d > maxAbs {
			maxAbs = d
		}
		n++
		if est.Total == rec.InputTokens {
			matched++
		} else {
			t.Errorf("%s: Total %d, usage %d", rec.Name, est.Total, rec.InputTokens)
		}
		if est.StateAndLongestQuestion > est.Total {
			t.Errorf("%s: state+longest %d exceeds total %d", rec.Name, est.StateAndLongestQuestion, est.Total)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("exact %d/%d max abs %d", matched, n, maxAbs)
	if n < 100 {
		t.Fatalf("fixture too small: %d", n)
	}
	if matched != n {
		t.Fatalf("exact %d/%d", matched, n)
	}
}

func TestEstimateStateAndLongest(t *testing.T) {
	body := []byte(`{"model":"jev-1.13.0","state":"hello","questions":{"q1":{"type":"noul","instructions":"Yes?"},"q2":{"type":"noul","instructions":"No?"}}}`)
	est, err := EstimateRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	// frame 259 + state 1 + two noul questions of 9; longest question is 9.
	if est.Total != 278 || est.StateAndLongestQuestion != 10 {
		t.Fatalf("got %+v", est)
	}
}

func TestEstimateRejectsBadRequest(t *testing.T) {
	if _, err := EstimateRequest([]byte(`{`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := EstimateRequest([]byte(`{"questions":{}}`)); err == nil {
		t.Fatal("expected missing state")
	}
}

func TestEmptyChoiceCriteriaOverestimates(t *testing.T) {
	// Empty option text drops some structural tokens the non-empty fit counts.
	// Overestimate is the safe direction for a budget check.
	body := []byte(`{"model":"jev-1.13.0","state":"hello","questions":{"q":{"type":"choice","instructions":"Which?","criteria":{"a":"","b":""}}}}`)
	est, err := EstimateRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if est.Total-285 != 16 {
		t.Fatalf("overestimate %d, want 16 (Total %d)", est.Total-285, est.Total)
	}
}
