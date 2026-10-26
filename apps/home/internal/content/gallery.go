package content

// Illustration is a built-in banner (F16b). Its picture is
// internal/web/static/assets/banners/<ID>.jpg, drawn from web/banners/<ID>.svg.
type Illustration struct {
	ID   string
	Name string
	// Alt is the alternative text used when the admin leaves the field empty (F16c).
	Alt string
}

// Gallery lists the illustrations, in the order of the admin form. The first one is
// the default chosen by migration 002. An id is stored in content.banner_choice and
// appears in a file name: never rename one, add a new one instead.
var Gallery = []Illustration{
	{"aube", "Aube", "Dégradé violet et cyan avec une fleur lumineuse au centre."},
	{"lagon", "Lagon", "Vagues cyan et violettes autour d'un cercle clair."},
	{"nebuleuse", "Nébuleuse", "Nuage violet et rose sur fond nocturne, avec un anneau lumineux."},
	{"prisme", "Prisme", "Bandes de lumière cyan et violette traversées par un losange."},
	{"orchidee", "Orchidée", "Dégradé rose et violet doux avec une arche claire."},
	{"minuit", "Minuit", "Bleu nuit traversé d'une lueur cyan en forme de pilule."},
}

// Dimensions of every illustration (D6).
const (
	IllustrationWidth  = 1600
	IllustrationHeight = 533
)

// FindIllustration returns the illustration with the given id.
func FindIllustration(id string) (Illustration, bool) {
	for _, il := range Gallery {
		if il.ID == id {
			return il, true
		}
	}
	return Illustration{}, false
}
