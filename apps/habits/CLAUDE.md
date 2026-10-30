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
sessions en base, rate limiting, CSRF, en-têtes de sécurité. Les habitudes et le suivi du jour
(ONBO-93, T4) le sont aussi : `/` et `/day/{date}` (cases à cocher, jours passés dans la
fenêtre `EDIT_WINDOW_DAYS`), `POST /day/{date}/toggle`, `/habits` (création, renommage, jours,
archivage, restauration). Les statistiques (ONBO-94, T5) le sont aussi : `/week/{date}`,
`/month/{AAAA-MM}`, `/year/{AAAA}` (et `/week`, `/month`, `/year` qui redirigent vers la période
courante). Les finitions mobile (ONBO-95, T6) le sont aussi : feuille de style et script de cochage. Rien n'est déployé : les manifests (`infra/apps/habits/`, T2) et le
premier déploiement (T7) viennent après.

## Reste à faire (spec §8)

T7 déploiement.

## Configuration

Variables d'environnement (voir `.env.example`) : `DATABASE_URL` et `ADMIN_PASSWORD_HASH`
(obligatoires ; hash bcrypt produit par `make hash-password`), `PORT` (`8080`), `BASE_URL`
(`https://habits.zazi.cool` : contrôle d'`Origin`, cookie `Secure`/`__Host-` et HSTS si
`https://`), `TRUST_PROXY` (`false` ; `true` derrière Traefik pour lire l'IP dans
`X-Forwarded-For`), `RATE_LIMIT_LOGIN` (`10` par IP et par heure). `TZ_NAME`
(`Europe/Paris` : fuseau qui définit le « jour ») et `EDIT_WINDOW_DAYS` (`14` : jours passés
modifiables, aujourd'hui inclus dans la fenêtre, le futur jamais).

## Commandes

Depuis `apps/habits/` (détails dans le `README.md`) : `make up` / `make down`, `make dev`,
`make test` (avec intégration PostgreSQL), `make lint` (gofmt + `go vet`), `make fmt`,
`make build` (image taguée avec le SHA git), `make migrate`, `make hash-password`.

## Conventions de l'app

- Paquets : `cmd/habits` (point d'entrée, configuration, arrêt propre), `internal/server`
  (handlers HTTP et middlewares ; dépend des interfaces `Store` et `Authenticator`, pas des
  paquets concrets), `internal/auth` (bcrypt, sessions ; dépend de l'interface `SessionStore`),
  `internal/habit` (types du domaine et règles pures : masque de jours, jour civil, fenêtre
  d'édition, validation du nom ; ni HTTP ni SQL),
  `internal/ratelimit` (fenêtre glissante en mémoire), `internal/store` (accès PostgreSQL),
  `internal/dbtest` (base jetable pour les tests), `internal/migrate` (exécuteur de migrations), `migrations/` (SQL
  embarqué, appliqué au démarrage et par `habits migrate`).
- Pas de framework ni de TypeScript : rendu `html/template` et un petit script embarqué
  (spec NF1). Aucune dépendance au-delà de `pgx` sans approbation de la spec (le hash bcrypt
  de T3 demandera `golang.org/x/crypto`, prévu par F13).
- Jours : toujours des **jours civils** (`time.Time` à minuit UTC, colonnes `date`), calculés
  avec `habit.Today(now, loc)` ; l'horloge (`Config.Now`) et le fuseau sont injectés, jamais
  `time.Now()` direct dans un handler.
- Jours concernés : masque 7 bits, bit 0 = lundi ; `habit_schedules` est versionnée (une ligne
  par changement, effective dès `valid_from`) : ne jamais réécrire une ligne passée (NF5).
  Modifier les jours un même jour remplace la version de ce jour.
- Archiver = `archived_on` (l'habitude disparaît ce jour-là et après) ; une habitude cochée
  n'est enregistrée que pour un jour où elle est concernée (`SetDone` refuse sinon).
- Port local de la base Compose : `5437` (5433 à 5436 sont pris par les autres apps).
- Tests d'intégration : `TEST_DATABASE_URL` (posé par `make test`) ; chaque test de
  migration travaille dans un schéma jetable.

## Statistiques

- Les règles de calcul sont pures et testées en table dans `internal/habit/stats.go`
  (`History.Applies`, `Tally`, `Streak`) ; le store ne fait que lire (`Store.History`, 2 requêtes
  SQL quelle que soit la période, NF6) et les handlers (`internal/server/stats.go`) mettent en page.
- Taux = jours faits ÷ jours concernés **et** existants jusqu'à aujourd'hui inclus ; le rendu est
  « 67 % » (espace insécable), « — » si rien n'était dû. Série : voir `Streak` (aujourd'hui non
  coché ne casse pas la série ; un jour non concerné la saute ; une habitude archivée n'en a pas).
- Les cellules portent des classes `lvl-none`, `lvl-0` à `lvl-4` (par quarts) et `done` / `missed` /
  `future` / `na` pour que T6 les colore ; il n'y a aucun style en ligne (CSP).

## Interface (T6)

- Assets embarqués dans `internal/server/static/` et servis publiquement sous `/static/` (la page de
  connexion en a besoin avant toute session) : `style.css` et `app.js`, tous deux en `'self'`
  (CSP stricte) ; jamais de `style=`, `onclick=` ni de script en ligne.
- Mobile d'abord : cibles tactiles d'au moins 44 px (`min-height: 44px` sur liens, boutons, champs ;
  `.check .box` fait 44×44), pas de défilement horizontal de la page (les tableaux défilent dans
  `.scroll`). Clair / sombre via variables CSS et `prefers-color-scheme` ; ne coder une couleur
  qu'en variable, en gardant le contraste AA dans les deux modes.
- Le script intercepte les formulaires `data-toggle`, poste avec `Accept: application/json` et met à
  jour la case ; sans JS ou en cas d'échec réseau côté `fetch` indisponible, le formulaire part
  normalement. Le message d'erreur va dans `#status` (`role="status"`).

## Décisions

- [x] D1–D6 de la spec : propositions appliquées (ticket ONBO-90).
