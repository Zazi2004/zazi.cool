# Accueil — `zazi.cool`

Page d'accueil de `zazi.cool` : un hub vers les services du site, avec une bannière, un
titre et un paragraphe modifiables dans une mini administration (`/admin`).
Spécification : [`docs/specs/home.md`](../../docs/specs/home.md) ;
contexte : [`CLAUDE.md`](CLAUDE.md).

> **En construction.** Seul le squelette existe (T1) : santé, version, migrations. La page
> publique (T3) et l'admin (T4 à T7) arrivent dans les tickets suivants.

## Prérequis

Docker (Compose v2), `make` et Go 1.27+.

## Démarrage rapide

```sh
make up        # app + PostgreSQL dans Docker → http://localhost:8080
curl localhost:8080/healthz
curl localhost:8080/readyz    # 200 si la base répond, 503 sinon
curl localhost:8080/version   # « local » en Compose, le SHA git avec `make build`
```

`.env` est créé depuis `.env.example` s'il manque ; les migrations sont appliquées au
démarrage. En local, le mot de passe de l'admin est `home-dev-password` (hash fourni dans
`.env.example`).

## Développement

```sh
make dev            # PostgreSQL dans Docker, serveur Go sur la machine (relancer après une modification)
make test           # tests Go, dont les tests d'intégration PostgreSQL
make lint           # gofmt + go vet
make fmt            # formate le Go
make build          # image Docker taguée avec le SHA git (injecté dans /version)
make migrate        # applique les migrations sur DATABASE_URL
make hash-password  # génère un ADMIN_PASSWORD_HASH (mot de passe saisi sans écho)
make down           # arrête les conteneurs (le volume de la base est conservé)
```

La base locale écoute sur le port **5436** (Undercover, Pastebin et Linktree utilisent
5433 à 5435). Si le port 8080 est pris, changer `PORT` et `PUBLIC_BASE_URL` dans `.env`.
