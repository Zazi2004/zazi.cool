# Accueil — `zazi.cool`

Page d'accueil de `zazi.cool` : un hub vers les services du site, avec une bannière, un
titre et un paragraphe modifiables dans une mini administration (`/admin`).
Spécification : [`docs/specs/home.md`](../../docs/specs/home.md) ;
contexte : [`CLAUDE.md`](CLAUDE.md).

> **En construction.** Le squelette (T1), la page publique (T3) et la connexion à l'admin
> (T4), l'édition du texte et du fond (T5) et les fonds animés (T8) existent. La bannière
> téléversée et la galerie (T6, T7) arrivent dans les tickets suivants.

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

Licences : police Geist (SIL OFL 1.1) et formes Coolshapes (MIT), dans
`internal/web/licenses/`.
