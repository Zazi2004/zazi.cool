# Linktree — `link.zazi.cool`

Pages de liens thématiques (`/musique`, `/jeux`…) avec liens, groupes en accordéon,
thèmes et fonds animés, gérées dans une administration en ligne (`/admin`).
Spécification : [`docs/specs/linktree.md`](../../docs/specs/linktree.md) ;
contexte : [`CLAUDE.md`](CLAUDE.md).

## Prérequis

Docker (Compose v2), `make`, Go 1.27+ et Node.js 24+.

## Démarrage rapide

```sh
make up        # app + PostgreSQL dans Docker → http://localhost:8080
curl localhost:8080/healthz
curl localhost:8080/version   # « local » en Compose, le SHA git avec `make build`
```

L'admin est sur `/admin`. En local, le mot de passe est `linktree-dev` (hash fourni dans
`.env.example`). `.env` est créé depuis `.env.example` s'il manque ; les migrations sont
appliquées au démarrage.

## Développement

```sh
make dev            # PostgreSQL dans Docker, serveur Go + Vite (rechargement à chaud du front)
make test           # tests Go (dont intégration PostgreSQL) et tests du front (Vitest)
make lint           # gofmt, go vet, vérification des types TypeScript, Prettier
make fmt            # formate le Go et le front
make build          # image Docker taguée avec le SHA git (injecté dans /version)
make migrate        # applique les migrations sur DATABASE_URL
make hash-password  # génère un ADMIN_PASSWORD_HASH (mot de passe saisi sans écho)
make down           # arrête les conteneurs (le volume de la base est conservé)
```

Avec `make dev`, ouvrir le serveur Go (`http://localhost:8080`), pas Vite : Go rend les
pages et y charge les modules du serveur Vite. Relancer `make dev` après une modification
Go.

La base locale écoute sur le port **5435** (Undercover et Pastebin utilisent 5433 et
5434). Si le port 8080 est pris, changer `PORT` et `PUBLIC_BASE_URL` dans `.env`.

## Formes coolshapes

Les SVG de `web/public/shapes/` sont générés par `npm run export-shapes` (dans `web/`)
depuis le paquet `coolshapes-react`, puis versionnés. L'application n'importe jamais ce
paquet.
