# Accueil — `zazi.cool`

Page d'accueil de `zazi.cool` : un hub vers les services du site, avec une bannière, un
titre et un paragraphe modifiables dans une mini administration (`/admin`).
Spécification : [`docs/specs/home.md`](../../docs/specs/home.md) ;
contexte : [`CLAUDE.md`](CLAUDE.md).

> **En construction.** Toutes les étapes de la spec sont en place : squelette (T1), page
> publique (T3), connexion à l'admin (T4), édition (T5), bannière téléversée (T6),
> illustrations par défaut (T7) et fonds animés (T8).

## Prérequis

Docker (Compose v2), `make`, Go 1.27+ et Node 24+ (pour les fonds animés, `web/`).

## Démarrage rapide

```sh
make up        # app + PostgreSQL dans Docker → http://localhost:8080 (page d'accueil)
curl localhost:8080/healthz
curl localhost:8080/readyz    # 200 si la base répond, 503 sinon
curl localhost:8080/version   # « local » en Compose, le SHA git avec `make build`
```

`.env` est créé depuis `.env.example` s'il manque ; les migrations sont appliquées au
démarrage. En local, le mot de passe de l'admin est `home-dev-password` (hash fourni dans
`.env.example`).

## Développement

```sh
make dev            # PostgreSQL dans Docker, serveur Go sur la machine (relancer après une modification du Go ou de web/)
make test           # tests du front (Vitest), puis tests Go, dont les tests d'intégration PostgreSQL
make lint           # gofmt + go vet, type check TypeScript + Prettier
make fmt            # formate le Go et le front
make build          # image Docker taguée avec le SHA git (injecté dans /version) ; échoue si le front dépasse son budget
make migrate        # applique les migrations sur DATABASE_URL
make export-banners # régénère les JPEG des illustrations depuis web/banners/*.svg (Chrome, Chromium ou Brave)
make hash-password  # génère un ADMIN_PASSWORD_HASH (mot de passe saisi sans écho)
make down           # arrête les conteneurs (le volume de la base est conservé)
```

La base locale écoute sur le port **5436** (Undercover, Pastebin et Linktree utilisent
5433 à 5435). Si le port 8080 est pris, changer `PORT` et `PUBLIC_BASE_URL` dans `.env`.

## Fichiers statiques

Tout est embarqué dans le binaire (`internal/web`) : gabarits dans `templates/`, fichiers
dans `static/`. Ceux de `static/assets/` sont servis sous un nom avec empreinte
(`/assets/home.<empreinte>.css`) et mis en cache un an ; une nouvelle extension doit être
ajoutée à `contentTypes` (`internal/web/web.go`). Les couleurs du thème sont dans
`static/assets/home.css`, vérifiées par le test de contraste.

Image de partage : `static/og-default.png` est une capture à 1200 × 630 de
`internal/web/source/og-default.html` (ouvrir le fichier dans un navigateur, ou avec
Playwright : `page.setViewportSize({ width: 1200, height: 630 })` puis
`page.screenshot()`).

Illustrations par défaut : les sources sont les SVG de `web/banners/` (1600 × 533,
formes Coolshapes et dégradés violet et cyan). `make export-banners` les exporte en JPEG
dans `internal/web/static/assets/banners/` (250 Kio au plus), servis sous
`/assets/banners/<id>.<empreinte>.jpg`. Les JPEG sont versionnés : après un changement de
SVG, relancer l'export et commiter les deux. La liste (identifiant, nom, texte alternatif)
est `content.Gallery` ; `TestGalleryPictures` vérifie qu'à chaque entrée correspond une
image au bon format.

Licences : police Geist (SIL OFL 1.1) et formes Coolshapes (MIT), dans
`internal/web/licenses/`.
