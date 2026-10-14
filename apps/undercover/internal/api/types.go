package api

// This file is the single definition of the JSON messages exchanged with the front.
// The TypeScript counterpart is web/src/api/types.ts; contract_test.go keeps both
// aligned through the web/src/api/contract.json fixture.

import "zazi.cool/undercover/internal/pairs"

// Side designates one word of a pair: "a" (WordA) or "b" (WordB).
type Side string

const (
	SideA Side = "a"
	SideB Side = "b"
)

// PairView is a pair as shown to players.
type PairView struct {
	ID      int64  `json:"id"`
	WordA   string `json:"wordA"`
	WordB   string `json:"wordB"`
	Extreme bool   `json:"extreme"`
}

// DrawResponse answers GET /api/pairs/random. CivilSide tells which word of the pair
// went to the civilians; it is sent back with Mr. White's guess.
type DrawResponse struct {
	ID             int64  `json:"id"`
	CivilWord      string `json:"civilWord"`
	UndercoverWord string `json:"undercoverWord"`
	CivilSide      Side   `json:"civilSide"`
	Extreme        bool   `json:"extreme"`
}

// GuessRequest is the body of POST /api/pairs/{id}/guess.
type GuessRequest struct {
	CivilSide Side   `json:"civilSide"`
	Guess     string `json:"guess"`
}

// GuessResponse answers POST /api/pairs/{id}/guess.
type GuessResponse struct {
	Correct  bool   `json:"correct"`
	Expected string `json:"expected"`
}

// CheckStatus is the outcome of a redundancy check.
type CheckStatus string

const (
	CheckOK        CheckStatus = "ok"
	CheckSimilar   CheckStatus = "similar"
	CheckDuplicate CheckStatus = "duplicate"
	CheckInvalid   CheckStatus = "invalid"
)

// CheckResponse answers GET /api/pairs/check, and POST /api/pairs when the pair
// is refused.
type CheckResponse struct {
	Status    CheckStatus `json:"status"`
	Error     string      `json:"error,omitempty"`
	Duplicate *PairView   `json:"duplicate,omitempty"`
	Similar   []PairView  `json:"similar"`
	Related   []PairView  `json:"related"`
}

// CreatePairRequest is the body of POST /api/pairs. Extreme is a pointer so that a
// missing choice can be told apart from "false": the category must be explicit.
// Website is a honeypot: hidden from humans, only bots fill it.
type CreatePairRequest struct {
	WordA          string `json:"wordA"`
	WordB          string `json:"wordB"`
	Extreme        *bool  `json:"extreme"`
	ConfirmSimilar bool   `json:"confirmSimilar"`
	Website        string `json:"website"`
}

// CreatePairResponse answers a successful POST /api/pairs.
type CreatePairResponse struct {
	Pair PairView `json:"pair"`
}

// ReportRequest is the body of POST /api/pairs/{id}/reports.
type ReportRequest struct {
	Reason pairs.Reason `json:"reason"`
}

// ErrorResponse is the body of every error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

func viewOf(p pairs.Pair) PairView {
	return PairView{ID: p.ID, WordA: p.WordA, WordB: p.WordB, Extreme: p.Extreme}
}

func viewsOf(ps []pairs.Pair) []PairView {
	// An empty, non-nil slice encodes as [] rather than null, which is simpler for
	// the front to handle.
	views := make([]PairView, 0, len(ps))
	for _, p := range ps {
		views = append(views, viewOf(p))
	}
	return views
}
