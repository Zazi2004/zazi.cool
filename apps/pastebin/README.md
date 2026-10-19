# Pastebin — `pb.zazi.cool`

Service de partage de texte et de code : formulaire de création sur `/`, vue formatée
`/<id>`, vue brute `/raw/<id>`, expiration de 10 minutes à 1 mois. Spécification : [`docs/specs/pastebin.md`](../../docs/specs/pastebin.md) ;
contexte : [`CLAUDE.md`](CLAUDE.md).

## Prérequis

Docker (Compose v2), `make` et Go 1.27+.

## Démarrage rapide

```sh
make up        # app + PostgreSQL dans Docker → http://127.0.0.1:8080
curl -i --data-urlencode content='bonjour' localhost:8080/   # 303 → /<id>
curl localhost:8080/healthz
curl localhost:8080/readyz
curl localhost:8080/version   # « local » en Compose, le SHA git avec `make build`
```

`.env` est créé depuis `.env.example` si absent. Les migrations sont appliquées au démarrage ;
les pastes expirés sont purgés au démarrage puis toutes les `PURGE_INTERVAL` (15 min).

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

Si le port 8080 ou 5434 est déjà pris (Undercover utilise aussi 8080), changer `PORT` ou
`DB_PORT` (et `DATABASE_URL`) dans `.env`.
