# Habits — `habits.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre à l'app.

## Contexte

Suivi d'habitudes personnel : cocher chaque jour les habitudes faites, consulter le suivi
par jour, semaine, mois et année. Privé (un seul utilisateur), mobile d'abord. La spec
`docs/specs/habits.md` (v2) fait foi ; la v1 est dans son historique git.

**Statut : premier déploiement (2026-10-30), recette en cours.** Le squelette (ONBO-90, T1) est en place : `GET /healthz`
(liveness, sans base), `GET /readyz` (base jointe), `GET /version` (SHA git), arrêt propre,
exécuteur de migrations. L'authentification (ONBO-92, T3) l'est aussi : `/login`, `/logout`,
sessions en base, rate limiting, CSRF, en-têtes de sécurité. Les habitudes et le suivi du jour
(ONBO-93, T4) le sont aussi : `/` et `/day/{date}` (cases à cocher, jours passés dans la
fenêtre `EDIT_WINDOW_DAYS`), `POST /day/{date}/toggle`, `/habits` (création, renommage, jours,
archivage, restauration). Les statistiques (ONBO-94, T5) le sont aussi : `/week/{date}`,
`/month/{AAAA-MM}`, `/year/{AAAA}` (et `/week`, `/month`, `/year` qui redirigent vers la période
courante). Les finitions mobile (ONBO-95, T6) le sont aussi : feuille de style et script de cochage. Les manifests (`infra/apps/habits/`, T2) sont en place et le
premier déploiement (T7) est en cours.

## Refonte v2 (cadrée le 2026-11-06)

La v1 décrite ci-dessous est en production mais va être remplacée. **La spec v2
(`docs/specs/habits.md`) fait foi** pour tout nouveau travail ; elle prime sur les règles v1 de
ce fichier en cas de conflit. En résumé :

- Modèle : période (jour / semaine / mois / année) + cible + jours autorisés, oui/non ou
  quantité ; période, cible et jours **versionnés** à partir de la période en cours ; emoji,
  couleur, moment (Matin / Journée / Soir), ordre manuel ; note journalière. Données v1
  abandonnées (migration `003_v2.sql`).
- Front **React + TypeScript + Vite + Motion** dans `web/`, embarqué ; Go devient une API JSON
  sous `/api/`. Ces dépendances sont approuvées ; les règles « pas de TypeScript » et « aucune
  dépendance au-delà de `pgx` » de la v1 ne s'appliquent plus au front.
- Toute règle métier reste calculée en Go (`internal/habit`) ; formats d'échange définis au
  §7 de la spec, alignés par des golden files.
- Direction visuelle : variante A « Anneaux » du prototype (branche `proto/habits-redesign`,
  jamais fusionnée).
- `EDIT_WINDOW_DAYS` passe à 7.
- Issue parente ONBO-109, sous-issues ONBO-110 → ONBO-118 (spec §13), mergées dans la branche
  d'intégration `feat/habits-redesign` ; toutes les décisions sont prises (spec §12), dont un pas
  d'incrément par habitude (`step`) et une bascule en un bloc à la fin (V7).

La recette v1 (T7) est abandonnée au profit de la recette v2 (spec §14).

## Configuration

Variables d'environnement (voir `.env.example`) : `DATABASE_URL` et `ADMIN_PASSWORD_HASH`
(obligatoires ; hash bcrypt produit par `make hash-password`), `PORT` (`8080`), `BASE_URL`
(`https://habits.zazi.cool` : contrôle d'`Origin`, cookie `Secure`/`__Host-` et HSTS si
`https://`), `TRUST_PROXY` (`false` ; `true` derrière Traefik pour lire l'IP dans
`X-Forwarded-For`), `RATE_LIMIT_LOGIN` (`10` par IP et par heure). `TZ_NAME`
(`Europe/Paris` : fuseau qui définit le « jour ») et `EDIT_WINDOW_DAYS` (`14` : jours passés
modifiables, aujourd'hui inclus dans la fenêtre, le futur jamais).

## Commandes

Depuis `apps/habits/` (détails dans le `README.md`) : `make up` / `make down`, `make dev`
(Go sur la machine + Vite sur `:5173`), `make test` (Go avec intégration PostgreSQL + Vitest),
`make lint` (gofmt + `go vet` + `tsc` + Prettier), `make fmt`, `make web-build`,
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
- Côté Go, pas de framework : bibliothèque standard, `pgx` et `golang.org/x/crypto`. Le front
  (`web/`, React + TypeScript + Vite) a ses propres dépendances, approuvées par la spec v2 :
  `react`, `react-dom`, `motion` et l'outillage ; toute autre demande l'accord de Noah.
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

## Socle SPA (v2, ONBO-110)

- `web/` : front React 19 + TypeScript `strict` + Vite, Prettier, Vitest (`package-lock.json`
  versionné). `web/embed.go` embarque `web/dist` ; `make` crée un `web/dist/.keep` pour que
  `go build` / `go vet` compilent avant le premier `npm run build`. L'image Docker construit le
  front dans une étape Node, l'exécution reste distroless sans Node.
- Routage côté client (`web/src/router.ts`, sans bibliothèque) : toute URL hors `/api/`,
  `/healthz`, `/readyz`, `/version` et les fichiers du build sert `index.html` ; un fichier
  manquant sous `/assets/` donne `404` (jamais du HTML à un `<script>`).
- API JSON sous `/api/` : `POST /api/login`, `POST /api/logout`, `GET /api/session`. Sans session,
  tout `/api/*` répond `401 {"error":"unauthorized","message":"…"}` ; avec session, une URL
  inconnue répond `404` JSON. Erreurs : `{"error": code, "message": texte FR}`. Les requêtes non
  `GET` sont contrôlées par `Sec-Fetch-Site` / `Origin` (`403`) et un corps non JSON donne `415`.
- CSP v2 (spec NF2) définie dans `internal/server/auth.go` (`pageCSP`) : aucune balise `<style>`
  ni attribut `style` dans le HTML, polices et icônes servies par l'app (Inter latin en variable,
  `web/src/fonts/`, licence OFL jointe), `assetsInlineLimit: 0` pour ne produire aucune URL `data:`
  de police.
- `EDIT_WINDOW_DAYS` et `TZ_NAME` restent lus par `cmd/habits`, mais rien ne les utilise tant que
  l'API des jours (ONBO-112) n'est pas là. Les paquets `internal/habit` et `internal/store` (modèle
  v1) sont remplacés par ONBO-111.
- Ce qui suit (statistiques, interface T6) décrit la v1 : les pages HTML sont supprimées de cette
  branche et ces sections seront réécrites avec les tickets qui les remplacent.

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

- Refonte esthétique (ONBO-106) : effets inspirés de React Bits (aurore, projecteur au survol,
  titre dégradé, pop de la coche) en CSS et JS maison, **sans React ni dépendance** (NF1). Toute
  animation doit être neutralisée par `prefers-reduced-motion` ; le script n'écrit que des
  variables CSS via `style.setProperty` (compatible avec la CSP).

## Décisions

- [x] D1–D6 de la spec : propositions appliquées (ticket ONBO-90).
