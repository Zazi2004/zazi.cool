# Habits — `habits.zazi.cool`

Suivi d'habitudes personnel (un seul utilisateur, pensé pour le téléphone). Spécification :
[`docs/specs/habits.md`](../../docs/specs/habits.md) ; contexte : [`CLAUDE.md`](CLAUDE.md).

Le squelette (ONBO-90) ne sert encore que `/healthz`, `/readyz` et `/version` ; les
fonctionnalités arrivent avec les tickets suivants de la spec (§8).

## Prérequis

Docker (Compose v2), `make` et Go 1.27+.

## Démarrage rapide

```sh
make up        # app + PostgreSQL dans Docker → http://127.0.0.1:8080
curl localhost:8080/healthz
curl localhost:8080/readyz
curl localhost:8080/version   # « local » en Compose, le SHA git avec `make build`
```

`.env` est créé depuis `.env.example` si absent. Les migrations sont appliquées au démarrage.

## Développement

```sh
make dev       # PostgreSQL dans Docker, serveur Go sur la machine
make test      # tests Go (dont intégration PostgreSQL)
make lint      # gofmt + go vet
make fmt       # formate le code
make build     # image Docker taguée avec le SHA git (injecté dans /version)
make migrate   # applique les migrations sur DATABASE_URL
make down      # arrête les conteneurs (le volume de la base est conservé)
```

Si le port 8080 ou 5437 est déjà pris (Undercover et Pastebin utilisent aussi 8080), changer
`PORT` ou `DB_PORT` (et `DATABASE_URL`) dans `.env`.
