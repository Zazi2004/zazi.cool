# Undercover — `undercover.zazi.cool`

Jeu d'ambiance joué avec un seul téléphone qui circule : tout le monde a le même mot, sauf
les undercovers (un mot proche) et Mr. White (aucun mot). Spécifications complètes dans
[`CLAUDE.md`](CLAUDE.md).

## Prérequis

Docker (Compose v2), `make`, Go 1.27+ et Node.js 24+.

## Démarrage rapide

```sh
cp .env.example .env    # fait automatiquement par make si absent
make up                 # app + PostgreSQL dans Docker → http://127.0.0.1:8080
```

Au démarrage, l'application applique les migrations et insère les paires de départ.

## Développement

```sh
make dev     # PostgreSQL dans Docker, serveur Go + Vite (hot reload) → http://localhost:5173
make test    # tests Go (dont intégration PostgreSQL) + tests du front
make lint    # gofmt, go vet, tsc, prettier
make fmt     # formate le Go et le front
make build   # image Docker taguée avec le SHA git
make migrate # applique migrations et paires de départ sur DATABASE_URL
make down    # arrête les conteneurs (le volume de la base est conservé)
```

Après une modification Go, relancer `make dev` (seul le front est rechargé à chaud).

Si le port 8080 ou 5432/5433 est déjà pris sur la machine, changer `PORT` ou `DB_PORT`
(et `DATABASE_URL`) dans `.env`.

## Ajouter des paires de départ

Éditer [`internal/seed/pairs.json`](internal/seed/pairs.json), puis `make test` : le test
vérifie qu'aucune paire n'est invalide, en double ou trop proche d'une autre. Les paires
sont insérées au prochain démarrage.
