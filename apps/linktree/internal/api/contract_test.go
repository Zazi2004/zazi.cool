package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"zazi.cool/linktree/internal/pages"
)

// The contract file holds one example of every JSON message, with every field set.
// This test checks it matches the Go types; web/src/api/contract.test.ts checks it
// matches the TypeScript types. After changing a message, regenerate it with:
//
//	go test ./internal/api -run TestContract -update
const contractPath = "../../web/src/api/contract.json"

var update = flag.Bool("update", false, "rewrite the contract file")

func TestContract(t *testing.T) {
	const id = "0199e6a8-0000-7000-8000-000000000001"
	image := Visual{Kind: VisualImage, ID: id, URL: "/images/" + id, Width: 512, Height: 384}
	shape := Visual{Kind: VisualShape, Shape: "star-1"}
	link := PublicLink{Kind: "link", ID: id, Title: "Spotify", URL: "https://open.spotify.com/track/1", Visual: &image}
	group := PublicGroup{Kind: "group", ID: id, Title: "Titre 1", Subtitle: "Artiste", Visual: &shape,
		OpenByDefault: true, Links: []PublicLink{link}}
	docLink := DocItem{ID: id, Kind: "link", Title: "Spotify", URL: "https://open.spotify.com/track/1",
		Visual: &image, Published: true, Links: []DocItem{}}
	docGroup := DocItem{ID: id, Kind: "group", Title: "Titre 1", Subtitle: "Artiste", Visual: &shape,
		OpenByDefault: true, Published: true, Links: []DocItem{docLink}}
	summary := PageSummary{ID: id, Slug: "musique", Title: "Musique", Description: "Mes sons", Avatar: &shape,
		Published: true, Listed: true, LinkCount: 3}

	contract := map[string]any{
		"themes":       pages.Themes,
		"accents":      pages.Accents,
		"imageVisual":  image,
		"shapeVisual":  shape,
		"indexView":    IndexView{View: ViewIndex, Pages: []IndexEntry{{Slug: "musique", Title: "Musique", Description: "Mes sons", Avatar: &image}}},
		"notFoundView": NotFoundView{View: ViewNotFound},
		"pageView": PageView{View: ViewPage, Page: PublicPage{ID: id, Slug: "musique", Title: "Musique",
			Description: "Mes sons", Avatar: &image, Theme: "nuit", Accent: "violet", Items: []any{link, group}}},
		"loginRequest":      LoginRequest{Password: "secret"},
		"sessionResponse":   SessionResponse{Authenticated: true},
		"pagesResponse":     PagesResponse{Pages: []PageSummary{summary}},
		"createPageRequest": CreatePageRequest{Title: "Musique", Slug: "musique"},
		"document": Document{ID: id, Slug: "musique", Title: "Musique", Description: "Mes sons", Avatar: &image,
			Theme: "nuit", Accent: "violet", Published: true, Listed: true, Version: 2, Items: []DocItem{docLink, docGroup}},
		"saveResponse":      SaveResponse{Version: 3},
		"reorderRequest":    ReorderRequest{IDs: []string{id}},
		"slugCheckResponse": SlugCheckResponse{Slug: "musique", Available: false, Error: "Ce slug est déjà utilisé."},
		"imageResponse":     ImageResponse{Visual: image},
		"errorResponse": ErrorResponse{Error: ErrorBody{Code: CodeValidation, Message: "Certains champs sont invalides.",
			Fields: map[string]string{"slug": "Ce slug est déjà utilisé."}}},
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
