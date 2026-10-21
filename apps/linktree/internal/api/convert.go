package api

import (
	"html/template"

	"zazi.cool/linktree/internal/pages"
	"zazi.cool/linktree/internal/render"
)

// imageURL is where GET /images/{id} serves an image.
func imageURL(id string) string {
	return "/images/" + id
}

func toVisual(v pages.Visual) *Visual {
	switch {
	case v.ImageID != "":
		return &Visual{Kind: VisualImage, ID: v.ImageID, URL: imageURL(v.ImageID), Width: v.Width, Height: v.Height}
	case v.Shape != "":
		return &Visual{Kind: VisualShape, Shape: v.Shape}
	}
	return nil
}

// fromVisual converts a visual sent by the admin. ok is false for an unknown kind.
func fromVisual(v *Visual) (vis pages.Visual, ok bool) {
	if v == nil {
		return pages.Visual{}, true
	}
	switch v.Kind {
	case VisualImage:
		return pages.Visual{ImageID: v.ID}, true
	case VisualShape:
		return pages.Visual{Shape: v.Shape}, true
	}
	return pages.Visual{}, false
}

func toDocument(p pages.Page) Document {
	return Document{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Description: p.Description,
		Avatar:      toVisual(p.Avatar),
		Theme:       p.Theme,
		Accent:      p.Accent,
		Published:   p.Published,
		Listed:      p.Listed,
		Version:     p.Version,
		Items:       toDocItems(p.Items),
	}
}

func toDocItems(items []pages.Item) []DocItem {
	out := make([]DocItem, len(items)) // never nil: JSON [] rather than null
	for i, it := range items {
		out[i] = DocItem{
			ID:            it.ID,
			Kind:          string(it.Kind),
			Title:         it.Title,
			Subtitle:      it.Subtitle,
			URL:           it.URL,
			Visual:        toVisual(it.Visual),
			OpenByDefault: it.OpenByDefault,
			Published:     it.Published,
			Links:         toDocItems(it.Links),
		}
	}
	return out
}

// fromDocument converts a document sent by the admin. Visuals of an unknown kind
// are reported in errs, keyed like pages.Validate does.
func fromDocument(d Document) (p pages.Page, errs pages.Errors) {
	errs = pages.Errors{}
	avatar, ok := fromVisual(d.Avatar)
	if !ok {
		errs["avatar"] = "Visuel invalide."
	}
	p = pages.Page{
		ID:          d.ID,
		Slug:        d.Slug,
		Title:       d.Title,
		Description: d.Description,
		Avatar:      avatar,
		Theme:       d.Theme,
		Accent:      d.Accent,
		Published:   d.Published,
		Listed:      d.Listed,
		Version:     d.Version,
		Items:       fromDocItems(d.Items, errs),
	}
	return p, errs
}

func fromDocItems(items []DocItem, errs pages.Errors) []pages.Item {
	out := make([]pages.Item, len(items))
	for i, it := range items {
		vis, ok := fromVisual(it.Visual)
		if !ok {
			errs["items."+it.ID+".visual"] = "Visuel invalide."
		}
		out[i] = pages.Item{
			ID:            it.ID,
			Kind:          pages.Kind(it.Kind),
			Title:         it.Title,
			Subtitle:      it.Subtitle,
			URL:           it.URL,
			Visual:        vis,
			OpenByDefault: it.OpenByDefault,
			Published:     it.Published,
			Links:         fromDocItems(it.Links, errs),
		}
	}
	return out
}

func toSummary(s pages.Summary) PageSummary {
	return PageSummary{
		ID:          s.ID,
		Slug:        s.Slug,
		Title:       s.Title,
		Description: s.Description,
		Avatar:      toVisual(s.Avatar),
		Published:   s.Published,
		Listed:      s.Listed,
		LinkCount:   s.LinkCount,
	}
}

// toPublicPage converts a page for visitors. p must already be p.Visible().
func toPublicPage(p pages.Page) PublicPage {
	items := make([]any, len(p.Items))
	for i, it := range p.Items {
		if it.Kind == pages.KindGroup {
			links := make([]PublicLink, len(it.Links))
			for j, l := range it.Links {
				links[j] = toPublicLink(l)
			}
			items[i] = PublicGroup{
				Kind:          string(pages.KindGroup),
				ID:            it.ID,
				Title:         it.Title,
				Subtitle:      it.Subtitle,
				Visual:        toVisual(it.Visual),
				OpenByDefault: it.OpenByDefault,
				Links:         links,
			}
			continue
		}
		items[i] = toPublicLink(it)
	}
	return PublicPage{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Description: p.Description,
		Avatar:      toVisual(p.Avatar),
		Theme:       p.Theme,
		Accent:      p.Accent,
		Items:       items,
	}
}

func toPublicLink(l pages.Item) PublicLink {
	return PublicLink{Kind: string(pages.KindLink), ID: l.ID, Title: l.Title, URL: l.URL, Visual: toVisual(l.Visual)}
}

// pageFallback is the <noscript> version of a page. p must already be p.Visible().
func pageFallback(p pages.Page) *render.Fallback {
	f := &render.Fallback{Heading: p.Title, Text: p.Description}
	for _, it := range p.Items {
		item := render.FallbackItem{Title: it.Title, URL: template.URL(it.URL)}
		for _, l := range it.Links {
			item.Links = append(item.Links, render.FallbackItem{Title: l.Title, URL: template.URL(l.URL)})
		}
		f.Items = append(f.Items, item)
	}
	return f
}

// indexFallback is the <noscript> version of the index.
func indexFallback(list []pages.Summary) *render.Fallback {
	f := &render.Fallback{Heading: indexTitle}
	for _, s := range list {
		f.Items = append(f.Items, render.FallbackItem{Title: s.Title, URL: template.URL("/" + s.Slug)})
	}
	return f
}
