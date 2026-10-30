# Habits — `habits.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre à l'app.

## Contexte

Suivi d'habitudes personnel : cocher chaque jour les habitudes faites, consulter le suivi
par jour, semaine, mois et année. Privé (un seul utilisateur), mobile d'abord. La spec
`docs/specs/habits.md` fait foi (brouillon tant que Noah n'a pas validé les décisions D1–D6 ;
les propositions de la spec sont appliquées).

**Statut : en développement.** Le squelette (ONBO-90, T1) est en place : `GET /healthz`
(liveness, sans base), `GET /readyz` (base jointe), `GET /version` (SHA git), arrêt propre,
exécuteur de migrations. L'authentification (ONBO-92, T3) l'est aussi : `/login`, `/logout`,
sessions en base, rate limiting, CSRF, en-têtes de sécurité ; `/` est pour l'instant une page
« Connecté » que T4 remplace par la vue du jour. Rien n'est déployé : les manifests (`infra/apps/habits/`, T2) et le
premier déploiement (T7) viennent après.

## Reste à faire (spec §8)

T4 habitudes et suivi du jour → T5 statistiques / T6 finitions mobile → T7 déploiement.

## Configuration

Variables d'environnement (voir `.env.example`) : `DATABASE_URL` et `ADMIN_PASSWORD_HASH`
(obligatoires ; hash bcrypt produit par `make hash-password`), `PORT` (`8080`), `BASE_URL`
(`https://habits.zazi.cool` : contrôle d'`Origin`, cookie `Secure`/`__Host-` et HSTS si
`https://`), `TRUST_PROXY` (`false` ; `true` derrière Traefik pour lire l'IP dans
`X-Forwarded-For`), `RATE_LIMIT_LOGIN` (`10` par IP et par heure). `TZ_NAME` et
`EDIT_WINDOW_DAYS` sont ajoutées par le ticket qui les utilise (T4).

## Commandes

Depuis `apps/habits/` (détails dans le `README.md`) : `make up` / `make down`, `make dev`,
`make test` (avec intégration PostgreSQL), `make lint` (gofmt + `go vet`), `make fmt`,
`make build` (image taguée avec le SHA git), `make migrate`, `make hash-password`.

## Conventions de l'app

- Paquets : `cmd/habits` (point d'entrée, configuration, arrêt propre), `internal/server`
  (handlers HTTP et middlewares ; dépend des interfaces `Store` et `Authenticator`, pas des
  paquets concrets), `internal/auth` (bcrypt, sessions ; dépend de l'interface `SessionStore`),
  `internal/ratelimit` (fenêtre glissante en mémoire), `internal/store` (accès PostgreSQL),
  `internal/dbtest` (base jetable pour les tests), `internal/migrate` (exécuteur de migrations), `migrations/` (SQL
  embarqué, appliqué au démarrage et par `habits migrate`).
- Pas de framework ni de TypeScript : rendu `html/template` et un petit script embarqué
  (spec NF1). Aucune dépendance au-delà de `pgx` sans approbation de la spec (le hash bcrypt
  de T3 demandera `golang.org/x/crypto`, prévu par F13).
- Port local de la base Compose : `5437` (5433 à 5436 sont pris par les autres apps).
- Tests d'intégration : `TEST_DATABASE_URL` (posé par `make test`) ; chaque test de
  migration travaille dans un schéma jetable.

## Décisions

- [x] D1–D6 de la spec : propositions appliquées (ticket ONBO-90).
