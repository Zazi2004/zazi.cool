package pairs

import (
	"errors"
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Chat", "chat"},
		{"  Éléphant  ", "elephant"},
		{"Le Chat", "chat"},
		{"l’Avion", "avion"},
		{"de la Crème", "creme"},
		{"Les Pâtes", "pate"},
		{"Chevaux", "chevau"},
		{"Porte-monnaie", "porte monnaie"},
		{"porte   monnaie", "porte monnaie"},
		{"Œuf", "oeuf"},
		{"os", "os"},
		{"Le", "le"}, // an article alone is kept rather than emptied
		{"Pain au chocolat", "pain au chocolat"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"chat", "chat", 0},
		{"chat", "chats", 1},
		{"chat", "chap", 1},
		{"chien", "niche", 4},
		{"", "abc", 3},
		{"été", "ete", 2}, // runes, not bytes
	}
	for _, tt := range tests {
		if got := Distance(tt.a, tt.b); got != tt.want {
			t.Errorf("Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestGuessMatches(t *testing.T) {
	tests := []struct {
		guess, word string
		want        bool
	}{
		{"chat", "Chat", true},
		{"le chat", "chat", true},
		{"elephant", "Éléphant", true},
		{"éléphants", "éléphant", true},
		{"elefant", "éléphant", false}, // two edits away
		{"elephnt", "éléphant", true},  // one typo on a long word
		{"chot", "chat", false},        // no typo tolerance on short words
		{"", "chat", false},
		{"   ", "chat", false},
		{"tigre", "chat", false},
	}
	for _, tt := range tests {
		if got := GuessMatches(tt.guess, tt.word); got != tt.want {
			t.Errorf("GuessMatches(%q, %q) = %v, want %v", tt.guess, tt.word, got, tt.want)
		}
	}
}

func TestNewCandidate(t *testing.T) {
	c, err := NewCandidate(" Tigre ", "chat")
	if err != nil {
		t.Fatalf("NewCandidate: unexpected error: %v", err)
	}
	want := Candidate{WordA: "chat", WordB: "Tigre", NormA: "chat", NormB: "tigre"}
	if c != want {
		t.Errorf("NewCandidate = %+v, want %+v (canonical order)", c, want)
	}
}

func TestNewCandidateInvalid(t *testing.T) {
	tests := []struct {
		name, a, b string
	}{
		{"empty word", "", "chat"},
		{"blank word", "   ", "chat"},
		{"same word", "Chat", "le chat"},
		{"too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmno", "chat"},
		{"forbidden character", "chat<script>", "tigre"},
		{"punctuation only", "-'-", "tigre"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCandidate(tt.a, tt.b)
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("NewCandidate(%q, %q) error = %v, want ErrInvalid", tt.a, tt.b, err)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	existing := []Pair{
		{ID: 1, WordA: "Chat", WordB: "Tigre"},
		{ID: 2, WordA: "Plage", WordB: "Piscine"},
		{ID: 3, WordA: "Chat", WordB: "Lion"},
		{ID: 4, WordA: "Café", WordB: "Thé", Hidden: true},
		{ID: 5, WordA: "Tigre", WordB: "Panthère", Hidden: true},
	}

	tests := []struct {
		name        string
		a, b        string
		wantStatus  Status
		wantDup     int64
		wantSimilar []int64
		wantRelated []int64
	}{
		{"new pair", "Pomme", "Poire", StatusOK, 0, nil, nil},
		{"exact duplicate", "chat", "tigre", StatusDuplicate, 1, nil, nil},
		{"reversed duplicate", "Les tigres", "le chat", StatusDuplicate, 1, nil, nil},
		{"hidden pair is still a duplicate", "thé", "café", StatusDuplicate, 4, nil, nil},
		{"plural", "Chat", "Tigres", StatusDuplicate, 1, nil, nil},
		{"similar", "Chats", "Tigrr", StatusSimilar, 0, []int64{1}, []int64{3}},
		{"related only", "Chat", "Chien", StatusOK, 0, nil, []int64{1, 3}},
		{"hidden pair is not related", "Panthère", "Puma", StatusOK, 0, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCandidate(tt.a, tt.b)
			if err != nil {
				t.Fatalf("NewCandidate: %v", err)
			}
			r := Check(c, existing)
			if r.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", r.Status, tt.wantStatus)
			}
			if tt.wantDup != 0 && (r.Duplicate == nil || r.Duplicate.ID != tt.wantDup) {
				t.Errorf("duplicate = %+v, want pair %d", r.Duplicate, tt.wantDup)
			}
			if got := ids(r.Similar); !slices.Equal(got, tt.wantSimilar) {
				t.Errorf("similar = %v, want %v", got, tt.wantSimilar)
			}
			if got := ids(r.Related); !slices.Equal(got, tt.wantRelated) {
				t.Errorf("related = %v, want %v", got, tt.wantRelated)
			}
		})
	}
}

func ids(ps []Pair) []int64 {
	var out []int64
	for _, p := range ps {
		out = append(out, p.ID)
	}
	return out
}
