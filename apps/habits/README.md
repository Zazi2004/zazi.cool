# Habits — `habits.zazi.cool`

Suivi d'habitudes personnel (un seul utilisateur, pensé pour le téléphone). Spécification :
[`docs/specs/habits.md`](../../docs/specs/habits.md) ; contexte : [`CLAUDE.md`](CLAUDE.md).

Pour l'instant (refonte v2, branche `feat/habits-redesign`) : `/healthz`, `/readyz`, `/version`,
la connexion en JSON (`/api/login`, `/api/logout`, `/api/session`) et le front React embarqué
(écran de connexion, écran « Aujourd'hui » vide, onglets) ; le reste arrive avec les tickets
suivants de la spec (§13).

## Prérequis

Docker (Compose v2), `make`, Go 1.27+ et Node 24+ (front dans `web/`).

## Démarrage rapide

```sh
make up        # app + PostgreSQL dans Docker → http://127.0.0.1:8080
curl localhost:8080/healthz
curl localhost:8080/readyz
curl localhost:8080/version   # « local » en Compose, le SHA git avec `make build`
```

`.env` est créé depuis `.env.example` si absent. En local, le mot de passe est
`habits-dev-password` ; `make hash-password` produit le hash d'un autre mot de passe
(`ADMIN_PASSWORD_HASH`). Les migrations sont appliquées au démarrage.

## Développement

```sh
make dev       # PostgreSQL dans Docker, serveur Go sur la machine, Vite sur http://localhost:5173
make test      # tests Go (dont intégration PostgreSQL) et tests du front (Vitest)
make lint      # gofmt + go vet, puis tsc + Prettier
make fmt       # formate le code Go et le front
make web-build # construit le front dans web/dist (embarqué dans le binaire)
make build     # image Docker taguée avec le SHA git (injecté dans /version)
make migrate   # applique les migrations sur DATABASE_URL
make down      # arrête les conteneurs (le volume de la base est conservé)
```

Si le port 8080 ou 5437 est déjà pris (Undercover et Pastebin utilisent aussi 8080), changer
`PORT` ou `DB_PORT` (et `DATABASE_URL`) dans `.env`.
