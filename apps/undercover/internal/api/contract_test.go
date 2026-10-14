package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"zazi.cool/undercover/internal/pairs"
)

// The contract file holds one example of every JSON message, with every field set.
// This test checks it matches the Go types; web/src/api/contract.test.ts checks it
// matches the TypeScript types. After changing a message, regenerate it with:
//
//	go test ./internal/api -run TestContract -update
const contractPath = "../../web/src/api/contract.json"

var update = flag.Bool("update", false, "rewrite the contract file")

func TestContract(t *testing.T) {
	yes := true
	view := PairView{ID: 1, WordA: "Chat", WordB: "Tigre", Extreme: false}
	contract := map[string]any{
		"pairView":           view,
		"drawResponse":       DrawResponse{ID: 1, CivilWord: "Tigre", UndercoverWord: "Chat", CivilSide: SideB, Extreme: false},
		"guessRequest":       GuessRequest{CivilSide: SideB, Guess: "tigre"},
		"guessResponse":      GuessResponse{Correct: true, Expected: "Tigre"},
		"checkResponse":      CheckResponse{Status: CheckDuplicate, Error: "message", Duplicate: &view, Similar: []PairView{view}, Related: []PairView{view}},
		"createPairRequest":  CreatePairRequest{WordA: "Chat", WordB: "Tigre", Extreme: &yes, ConfirmSimilar: true, Website: ""},
		"createPairResponse": CreatePairResponse{Pair: view},
		"reportRequest":      ReportRequest{Reason: pairs.ReasonExtreme},
		"errorResponse":      ErrorResponse{Error: "message"},
	}

	got, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	if *update {
		if err := os.WriteFile(contractPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract (run with -update to create it): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s is out of date with the Go types: run `go test ./internal/api -run TestContract -update`, then check the TypeScript types.\ngot:\n%s", contractPath, got)
	}
}
