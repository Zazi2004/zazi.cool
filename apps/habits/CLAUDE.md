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
(`Europe/Paris` : fuseau qui définit le « jour ») et `EDIT_WINDOW_DAYS` (`7` : nombre de jours
modifiables, aujourd'hui inclus donc aujourd'hui − 6 au plus loin ; le futur jamais).

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
- `EDIT_WINDOW_DAYS` (défaut 7) alimente `server.Config.EditWindowDays` ; `TZ_NAME` définit le jour de
  `server.Config.Location`.
- La section « Interface (T6) » ci-dessous décrit la v1 : les pages HTML sont supprimées de cette
  branche et elle sera réécrite avec ONBO-117.

## Habitudes (v2, ONBO-111)

- Migration `003_v2.sql` : supprime les tables v1 (`completions`, `habit_schedules`, `habits`) et crée
  tout le schéma v2 (`habits`, `habit_rules`, `entries`, `day_notes`) ; `sessions` est intacte. Elle
  est testée sur une base vide et sur une base v1 avec données (`migrations/migrations_test.go`).
- `internal/habit` : types du domaine (`Habit`, `Input`, `Moment`, `Period`, `Color`, `Weekdays`) et
  `Input.Validate` (erreurs `*ValidationError` au message français, affiché tel quel par le front).
  Un emoji = 1 à 10 points de code sans lettre, espace ni caractère de contrôle (la bibliothèque
  standard ne sait pas compter les graphèmes).
- `internal/store/habits.go` : `CreateHabit` écrit l'habitude et sa première règle
  (`valid_from` = jour de création) dans une transaction, en dernière position de son moment ;
  `ListHabits` lit la règle en vigueur ce jour-là. Un nom pris (index unique partiel, erreur
  PostgreSQL `23505`) devient `habit.ErrNameTaken` → `409`.
- API : `GET /api/habits`, `POST /api/habits` (`201`, `409` nom pris, `422` validation, code
  `invalid_habit`). Tous les champs de `HabitInput` sont obligatoires (pas de valeur par défaut côté
  serveur : le front envoie tout). Les DTO JSON (`internal/server/habits.go`) sont distincts des
  types du domaine : `archivedOn` y est une date `AAAA-MM-JJ`.
- **Golden files (NF9)** : `TestGoldenFiles` (`internal/server/habits_test.go`) écrit, à chaque
  `go test`, la réponse réelle des handlers dans `web/src/golden/*.json` (à committer). Côté front,
  `web/src/golden.test.ts` les vérifie contre `web/src/types.ts` (`satisfies` + mêmes clés) ;
  `npm test` lance `tsc --noEmit` avant Vitest, donc un champ renommé d'un seul côté fait échouer
  `make test`. Chaque nouveau type de réponse ajoute son golden et sa vérification.
- Front : `web/src/ui/Habits.tsx` (liste groupée par moment) et `HabitSheet.tsx` (panneau du bas,
  `<dialog>` natif) ; `web/src/habits.ts` ne contient que des fonctions d'affichage / de mise en
  forme du formulaire, aucune règle métier. Les couleurs de la palette sont des classes
  `.color-<clé>` qui posent `--tone`.

## Aujourd'hui (v2, ONBO-112)

- `GET /api/days/{date}` et `PUT /api/days/{date}/entries/{habitId}` (`{value}`, `0` supprime la
  saisie) répondent tous deux par le `Day`. Pour l'instant **seul le jour courant** est accepté
  (`422` sinon) ; série, bande de la semaine et note sont neutres (`0`, `[]`, `""`) jusqu'aux
  tickets suivants. `Day` porte en plus
  `done` / `total` (« N / M du jour »), ajoutés au §7 de la spec.
- Règles pures dans `internal/habit/day.go` (`Progress`, `Complete`, `DayProgress`, `DayCounts`,
  `GroupByMoment` : réussies grisées en bas, ordre manuel conservé), testées en table. Le store
  (`internal/store/days.go`) lit le jour en une requête (`DayHabits`) et `SetEntry` refuse une
  habitude absente ce jour-là (`ErrNotFound`, `404`) ou un jour exclu par `weekdays`
  (`ErrDayNotAllowed`, `422`).
- Front : `web/src/ui/Today.tsx` ; `web/src/today.ts` décide ce qu'un toucher envoie (`tapValue` :
  oui/non bascule, quantité + `step`, cible dépassable) et applique la mise à jour optimiste d'une
  ligne ; la réponse du serveur remplace tout le jour, une erreur restaure l'état et affiche le
  message. Seule la réponse du dernier toucher est appliquée. Le jour demandé est la date locale
  de l'appareil.
- Golden `day.json` ajouté (`TestGoldenFiles`, `golden.test.ts`).

## Jours, saisie exacte, série et note (v2, ONBO-113)

- `GET /api/days/{date}` accepte tout jour jusqu'à aujourd'hui (le futur donne `422`
  `day_in_future`) ; `editable` vient de `habit.Editable(day, today, EditWindowDays)`. Les écritures
  (`PUT …/entries/{id}`, `PUT …/note`) refusent un jour hors fenêtre ou futur en `422`
  `day_not_editable` **avant** tout accès au store : rien n'est modifié.
- Note : `PUT /api/days/{date}/note` `{text}` → `204` ; vide = supprimée, plus de 1 000 caractères
  (points de code) → `422` `invalid_note`. `Day.note` la renvoie.
- Série globale et bande de la semaine viennent d'**une seule** requête, `Store.DayStats` (somme par
  jour des habitudes `day` : dues, faites, progression), depuis la plus ancienne habitude jusqu'à
  aujourd'hui ou au dimanche affiché. `habit.Streak` (pure, testée en table) compte les journées
  complètes jusqu'à aujourd'hui : aujourd'hui incomplet ne casse pas la série, un jour sans habitude
  `day` est sauté. `GET /api/days/{date}` fait 3 requêtes (NF7) : `DayHabits`, `DayStats`, `DayNote`.
  La série est toujours celle d'aujourd'hui, même sur un jour passé.
- Front : `/day/{date}` (`/` = aujourd'hui) ; `Today` est monté avec `key` = date. Gestes dans
  `web/src/today.ts` (pur : `swipeTarget`, `cancelsHold`, seuils 60 px / 10 px / 450 ms) et
  `ui/useLongPress.ts`. Un mouvement de plus de 10 px annule l'appui long ; `touch-action: pan-y` sur
  la page laisse le défilement vertical au navigateur. Le bouton « Saisir une valeur » est l'alternative
  clavier à l'appui long.
- Confettis (`ui/Confetti.tsx`, CSS seul, sans dépendance) : seulement quand la réponse du serveur à
  une action passe `complete` de faux à vrai (`shouldCelebrate`) ; jamais à l'ouverture d'un jour ni
  avec `prefers-reduced-motion`.
- Note : `ui/DayNote.tsx`, enregistrement 800 ms après la dernière frappe, envoi immédiat du texte non
  enregistré en quittant le jour.

## Objectifs sur période (v2, ONBO-114)

- `internal/habit/period.go` : `PeriodStart` (lundi de la semaine, 1ᵉʳ du mois, 1ᵉʳ janvier),
  `PeriodDays` et `NewPace` (rythme : attendu = cible × jours écoulés ÷ jours de la période, écart
  arrondi, libellé « N d'avance » / « N de retard » / « dans le rythme » / « objectif atteint »),
  testés en table (années bissextiles, semaines à cheval sur deux mois / deux années). `Pace.Expected`
  est un flottant (127,4 en CA2) ; `pace` vaut `null` pour `day`.
- `Store.DayHabits` calcule `Logged.Total` (somme des saisies du début de la période au jour demandé)
  dans la même requête ; les débuts de période sont calculés en Go et passés en paramètres.
- Les habitudes semaine / mois / année apparaissent chaque jour autorisé, même une fois atteintes
  (grisées en bas du groupe, toujours incrémentables) ; elles n'entrent ni dans le grand anneau ni
  dans `done` / `total`. Un jour non autorisé : absente de `Day`, saisie refusée (`422`).
- Front : `subtitle` ajoute la période (« cette année »), `paceTone` donne la teinte (`.pace-ahead`
  vert, `.pace-behind` rouge) ; la mise à jour optimiste déplace le total de la période du même écart
  que la valeur du jour.

## Modifier, réordonner, archiver (v2, ONBO-115)

- `PATCH /api/habits/{id}` (champs partiels), `POST /api/habits/{id}/archive` et `/restore` répondent
  par l'`Habit` ; `PUT /api/habits/order` `{moment, ids}` → `204`, `ids` = toutes les habitudes actives
  du moment, une fois chacune (sinon `422` `invalid_order`, rien ne change). Une habitude inconnue ou
  déjà archivée donne `404` à l'édition ; restaurer un nom repris par une active donne `409`.
- Versionnement : `habit.PlanRuleChange` (pure, testée en table) décide quoi écrire ; `writeRule`
  (`internal/store/habits.go`) l'applique dans la transaction de `UpdateHabit` : `valid_from` = début
  de la période en cours **selon la nouvelle période**, les versions de `valid_from` ≥ celui-ci sont
  remplacées, aucune version passée n'est touchée ; règle inchangée = aucune écriture ; modification
  annulée dans la même période = version supprimée. Le statut « interrompue » (D8) n'est pas stocké :
  les statistiques (ONBO-116, voir plus bas) le déduisent du `valid_from` de la version suivante.
- Nom, emoji, couleur, unité, pas : non versionnés. Changer de moment place l'habitude en dernier du
  nouveau moment ; restaurer la place en dernier de son moment ; l'ordre posé par `order` est celui
  de l'écran Aujourd'hui (avant le tri « réussies en bas »).
- `GET /api/habits` renvoie aussi `ruleStarts` (par période) : le panneau l'affiche avant
  l'enregistrement (« s'applique à partir du 2 novembre ») sans recalculer de règle côté front.
- Front : `HabitSheet` sert à créer et modifier (le PATCH n'envoie que les champs changés, `toPatch`) ;
  `Habits.tsx` réordonne par poignée (événements pointeur, aperçu local puis `PUT` au relâchement,
  retour à l'ordre précédent en cas d'erreur) ou par les flèches ↑ ↓ ; section repliée « Archivées ».

## Statistiques (v2, ONBO-116)

- `GET /api/stats` (`{habits: HabitSummary[]}`, habitudes actives) et `GET /api/stats/habits/{id}?year=`
  (`HabitStats` ; année par défaut = courante, hors 2000…année courante → `422` `invalid_year`,
  habitude inconnue → `404`, une habitude archivée reste lisible). Goldens `stats.json` et
  `habit-stats.json`.
- Règles pures dans `internal/habit/stats.go`, testées en table : `History.Periods` (toutes les
  périodes, la plus ancienne d'abord), `Streaks`, `Rate`, `History.Summary`, `History.Heatmap`.
  Le store (`internal/store/stats.go`) ne fait que lire : `ActiveHistories` et `HabitHistory`, **3
  requêtes** (habitudes, règles, saisies) quel que soit l'âge de l'habitude ou le nombre d'habitudes
  (le test les compte avec un traceur pgx).
- Une période est évaluée avec la règle de son premier jour ; la première période d'une habitude
  commence au début de la période qui contient sa création. `day` : une « période » = un jour autorisé.
  La dernière période d'une règle remplacée par un **changement de période** est coupée la veille de
  la version suivante et vaut `interrupted` (D8) : ni série, ni taux. Statuts : `success` (cible
  atteinte, y compris pour la période en cours), `current` (en cours, pas atteinte), `missed`
  (close, pas atteinte).
- Série : périodes `success` consécutives en remontant ; `current` et `interrupted` sont sautées ;
  `missed` l'arrête. `null` si la règle en vigueur est `year`. Habitude archivée : calculs arrêtés
  la veille de l'archivage.
- Taux (fraction 0..1, `null` = « — ») : réussies ÷ (réussies + manquées) parmi les périodes qui
  **chevauchent** l'intervalle (30 derniers jours ; année affichée).
- Heatmap : une case par jour de l'année, `level` `null` hors vie de l'habitude ou jour non autorisé ;
  `day` = progression du jour en quarts, autres périodes = valeur ÷ plus forte valeur de l'année.
  `periods` : plus récente d'abord, vide pour `day`.
- Front : `ui/Stats.tsx` (cartes), `ui/StatsHabit.tsx` (page, année en état local ; « année
  précédente » désactivée si l'année affichée ne contient aucun jour de l'habitude), `stats.ts` (mise
  en forme seulement). La heatmap est une grille CSS (semaines en colonnes, `row-N` place le 1ᵉʳ janvier),
  sans style en ligne.

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
