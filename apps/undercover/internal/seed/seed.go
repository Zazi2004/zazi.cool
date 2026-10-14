// Package seed provides the starter word pairs, embedded in the binary.
package seed

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed pairs.json
var pairsJSON []byte

// Pair is one starter pair as written in pairs.json.
type Pair struct {
	A       string `json:"a"`
	B       string `json:"b"`
	Extreme bool   `json:"extreme"`
}

// Pairs decodes the embedded starter pairs.
func Pairs() ([]Pair, error) {
	var ps []Pair
	if err := json.Unmarshal(pairsJSON, &ps); err != nil {
		return nil, fmt.Errorf("decode seed pairs: %w", err)
	}
	return ps, nil
}
