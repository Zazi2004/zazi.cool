package content

// Service is one card of the homepage.
type Service struct {
	Name        string
	Description string
	URL         string
	Domain      string
	// Icon is the asset name of its shape, in internal/web/static/assets.
	Icon string
}

// Services is the list of the site's services (D3): it only changes when an app
// is added, which goes through a pull request anyway.
var Services = []Service{
	{
		Name:        "Undercover",
		Description: "Le jeu d'ambiance à jouer entre amis sur un seul téléphone.",
		URL:         "https://undercover.zazi.cool",
		Domain:      "undercover.zazi.cool",
		Icon:        "shape-undercover.svg",
	},
	{
		Name:        "Pastebin",
		Description: "Partager un texte ou du code avec un lien qui expire.",
		URL:         "https://pb.zazi.cool",
		Domain:      "pb.zazi.cool",
		Icon:        "shape-pastebin.svg",
	},
	{
		Name:        "Liens",
		Description: "Mes pages de liens : musique, jeux…",
		URL:         "https://link.zazi.cool",
		Domain:      "link.zazi.cool",
		Icon:        "shape-liens.svg",
	},
}
