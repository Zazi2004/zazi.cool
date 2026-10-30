# Spec — Habit tracker (`habits.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **Brouillon** — en attente des réponses de Noah aux décisions ouvertes |
| Issue Multica  | ONBO-88 |
| Application    | `apps/habits/` |
| Auteur, valideur | Rédacteur de specs / Noah |

## 1. Contexte

Noah veut un suivi d'habitudes personnel : chaque jour, cocher les habitudes faites, puis
consulter le suivi par jour, semaine, mois et année. L'app est **privée** (un seul
utilisateur) et pensée d'abord pour le **téléphone**, où l'on coche en quelques secondes.
Rien n'existe encore dans le dépôt : pas de dossier `apps/habits/`, pas de manifests, pas
de base. Le DNS wildcard `*.zazi.cool` couvre déjà le sous-domaine.

L'authentification reprend le modèle déjà spécifié pour l'admin de l'accueil
(`docs/specs/home.md`, F17–F20) : mot de passe unique, session par cookie.

## 2. Objectifs

- **O1** — Cocher les habitudes du jour en moins de 10 secondes depuis un téléphone.
- **O2** — Voir, pour n'importe quelle période (jour, semaine, mois, année), ce qui a été
  fait et le taux de réussite.
- **O3** — Les données de Noah ne sont lisibles que par Noah (aucune page publique).
- **O4** — Le suivi est fiable : corriger un oubli sur un jour passé récent est possible,
  et les statistiques ne changent pas si on renomme ou archive une habitude.

## 3. Périmètre

### Inclus (MVP)
- Habitudes quotidiennes : création, renommage, archivage.
- Cases à cocher par jour (aujourd'hui et jours passés), annulables.
- Habitudes **à jours choisis** (ex. lundi, mercredi, vendredi) en plus du « tous les jours » (D2).
- Vues **jour**, **semaine**, **mois**, **année** avec taux de réussite.
- Série en cours (« streak ») par habitude.
- Connexion par mot de passe unique.

### Hors périmètre
- Plusieurs utilisateurs, inscription, réinitialisation de mot de passe par l'interface.
- Rappels / notifications, export, import de données.
- Habitudes quantitatives (« 8 verres d'eau ») : seule la case faite / pas faite existe.
- Notes, catégories, graphiques complexes, application mobile native.
- Mode hors ligne (PWA) : piste ultérieure.

## 4. Exigences

### Fonctionnelles

**Habitudes**

- **F1** — Créer une habitude : nom (1 à 60 caractères, unique parmi les habitudes
  actives) et jours concernés (par défaut tous les jours, D2).
- **F2** — Renommer une habitude ; changer ses jours concernés n'affecte que les jours
  **à partir d'aujourd'hui** (l'historique garde ses règles passées, voir NF5).
- **F3** — Archiver une habitude : elle disparaît de la vue du jour et des jours futurs
  mais reste dans l'historique et les statistiques passées. Une habitude archivée peut
  être restaurée. Pas de suppression définitive en MVP.

**Suivi**

- **F4** — La page d'accueil `/` affiche **aujourd'hui** : la liste des habitudes
  concernées par ce jour, chacune avec une case. Cocher ou décocher enregistre
  immédiatement (requête serveur, sans rechargement de page si JS disponible, sinon
  formulaire).
- **F5** — On peut ouvrir un **jour passé** et cocher / décocher ses habitudes, jusqu'à
  `EDIT_WINDOW_DAYS` jours en arrière (défaut 14, D3). Les jours futurs ne sont pas
  modifiables.
- **F6** — Le « jour » est celui du fuseau `Europe/Paris` (variable `TZ_NAME`), pas celui
  du serveur.

**Statistiques**

- **F7** — Vue **semaine** (lundi à dimanche) : grille habitudes × 7 jours (fait, pas
  fait, non concerné) et taux de réussite par habitude et global.
- **F8** — Vue **mois** : calendrier du mois, chaque jour coloré selon la part d'habitudes
  faites ; taux par habitude sur le mois.
- **F9** — Vue **année** : 12 mois, chacun avec son taux, plus une grille de
  365 petits carrés (style « heatmap ») ; taux par habitude sur l'année.
- **F10** — Taux de réussite = jours faits ÷ jours où l'habitude était **concernée et
  existante** sur la période, jusqu'à aujourd'hui inclus (les jours futurs et ceux
  d'avant la création de l'habitude ne comptent pas).
- **F11** — Série en cours par habitude : nombre de jours concernés consécutifs faits,
  en finissant à aujourd'hui (si aujourd'hui n'est pas encore coché, la série d'hier est
  conservée). Les jours « non concernés » ne cassent pas la série.

**Authentification**

- **F12** — Toutes les pages (sauf `/login`, `/healthz`, `/readyz`, `/version`) exigent
  une session ; sinon redirection `303` vers `/login`.
- **F13** — Mot de passe unique : l'app ne connaît que son hash bcrypt
  (`ADMIN_PASSWORD_HASH`). `POST /login` ouvre une session de 30 jours (jeton aléatoire
  de 256 bits, seule l'empreinte SHA-256 est en base), cookie `HttpOnly`,
  `SameSite=Strict`, `Secure` + préfixe `__Host-` en HTTPS. `POST /logout` la ferme.
- **F14** — Tentatives de connexion limitées par IP (`RATE_LIMIT_LOGIN` par heure,
  défaut 10, en mémoire) : au-delà, `429`, même avec le bon mot de passe.
- **F15** — `GET /version` renvoie le SHA git de l'image (vérification des déploiements,
  `docs/workflow/cycle-de-vie.md`).

### Non fonctionnelles

- **NF1** — Pages en rendu serveur (`html/template`), CSP stricte
  (`default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self';
  form-action 'self'; base-uri 'none'; frame-ancestors 'none'`), `nosniff`,
  `X-Robots-Tag: noindex`. Un petit script embarqué (pas de TypeScript/Vite) envoie les
  cases cochées sans recharger ; sans JS, tout fonctionne par formulaires.
- **NF2** — **CSRF** : les `POST` sont refusés (`403`) si `Sec-Fetch-Site` vaut
  `cross-site` ou si `Origin` ne correspond pas à `BASE_URL`.
- **NF3** — **Mobile d'abord** : utilisable à partir de 360 px de large, cases de 44 px
  au moins, contraste AA, mode clair / sombre selon le système, navigation clavier.
- **NF4** — **Vie privée** : aucun pisteur ni ressource tierce ; le seul cookie est celui
  de session ; l'IP n'est gardée qu'en mémoire.
- **NF5** — **Historique stable** : les statistiques passées ne changent pas quand on
  modifie une habitude (les jours concernés sont versionnés, voir Données).
- **NF6** — Ressources : une seule réplique ; requests `10m` CPU / `32Mi`, limits
  `200m` / `128Mi` (à ajuster après mesure). Les vues année exigent au plus 2 requêtes SQL.

### Interfaces

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `GET /` | Aujourd'hui (équivaut à `/day/<date du jour>`) | 200 ; 303 → `/login` |
| `GET /day/{date}` | Un jour (`AAAA-MM-JJ`) | 200 ; 303 ; 400 date invalide |
| `POST /day/{date}/toggle` | Formulaire `habit_id`, `done` (`0`/`1`) ; JSON si `Accept: application/json` | 200 / 303 ; 403 ; 404 ; 422 jour hors fenêtre ou futur |
| `GET /week/{date}` | Semaine contenant la date | 200 ; 400 |
| `GET /month/{AAAA-MM}` | Mois | 200 ; 400 |
| `GET /year/{AAAA}` | Année | 200 ; 400 |
| `GET /habits` | Gestion des habitudes (actives et archivées) | 200 |
| `POST /habits` | Crée (`name`, `weekdays`) | 303 ; 403 ; 409 nom pris ; 422 |
| `POST /habits/{id}` | Renomme / change les jours | 303 ; 403 ; 404 ; 409 ; 422 |
| `POST /habits/{id}/archive`, `/restore` | Archive / restaure | 303 ; 403 ; 404 |
| `GET /login`, `POST /login`, `POST /logout` | Session | 200 ; 303 ; 401 ; 403 ; 429 |
| `GET /healthz`, `GET /readyz`, `GET /version` | Santé (readyz : base joignable), SHA | 200 ; 503 |

### Données

PostgreSQL, base `habits` et utilisateur dédié (Job `create-databases`, `infra/data`).

| Table | Colonnes principales | Note |
|-------|----------------------|------|
| `habits` | `id` bigserial pk, `name` text, `created_on` date, `archived_on` date null | nom unique parmi `archived_on is null` (index unique partiel) |
| `habit_schedules` | `habit_id`, `valid_from` date, `weekdays` smallint (masque 7 bits) | une ligne par changement de jours ; pk (`habit_id`, `valid_from`). Résout NF5 |
| `completions` | `habit_id`, `day` date, `created_at` timestamptz | pk (`habit_id`, `day`) ; présence = fait, décocher = suppression de la ligne |
| `sessions` | `token_hash` bytea pk, `password_hash_fp` bytea, `expires_at` timestamptz | comme `home` : changer le mot de passe invalide les sessions ; purge horaire |

Volume : quelques dizaines d'habitudes × 365 jours × des années = quelques milliers de
lignes ; aucune purge de `completions`. Dates en `date` (pas d'heure) ; calcul du jour
courant côté Go avec `TZ_NAME`.

### Configuration

| Variable | Rôle | Défaut |
|----------|------|--------|
| `PORT` | Port d'écoute | `8080` |
| `DATABASE_URL` | Connexion PostgreSQL | — (obligatoire) |
| `BASE_URL` | URL publique (CSRF, cookie) | `https://habits.zazi.cool` |
| `ADMIN_PASSWORD_HASH` | Hash bcrypt du mot de passe (`make hash-password`) | — (obligatoire) |
| `RATE_LIMIT_LOGIN` | Tentatives de connexion par IP et par heure | `10` |
| `TZ_NAME` | Fuseau qui définit le « jour » | `Europe/Paris` |
| `EDIT_WINDOW_DAYS` | Nombre de jours passés modifiables | `14` |

## 5. Critères d'acceptation

- **CA1** (F1) — Étant donné une habitude « Lire » créée, quand j'ouvre `/`, alors elle
  apparaît non cochée ; une seconde « Lire » active est refusée (`409`).
- **CA2** (F4) — Quand je coche « Lire » puis recharge la page, alors elle reste cochée ;
  quand je la décoche, la ligne `completions` disparaît.
- **CA3** (F5) — Cocher un jour d'il y a 3 jours fonctionne ; cocher un jour d'il y a
  `EDIT_WINDOW_DAYS` + 1 jours ou un jour futur renvoie `422` et ne modifie rien.
- **CA4** (F6) — Étant donné l'heure 23:30 UTC un 30 juin, quand j'ouvre `/`, alors le jour
  affiché est le 1ᵉʳ juillet (Europe/Paris, test avec horloge injectée).
- **CA5** (F2, NF5) — Étant donné une habitude faite tous les jours puis restreinte au
  lundi à partir d'aujourd'hui, alors le taux et la grille de la semaine passée sont
  inchangés.
- **CA6** (F3) — Une habitude archivée n'apparaît plus sur `/` mais ses jours cochés
  restent comptés dans les vues passées ; restaurée, elle réapparaît.
- **CA7** (F7) — La vue semaine affiche, pour une habitude « lun / mer / ven », les autres
  jours en « non concerné » et ne les compte pas dans le taux.
- **CA8** (F8, F9) — Étant donné 3 habitudes dont 2 faites un jour donné, alors ce jour
  est rendu à 67 % dans les vues mois et année.
- **CA9** (F10) — Une habitude créée le 10 du mois n'a aucun jour compté avant le 10 ; les
  jours futurs ne baissent pas le taux.
- **CA10** (F11) — Étant donné une habitude faite les 3 derniers jours concernés, alors la
  série vaut 3 ; si aujourd'hui n'est pas encore coché, elle vaut encore 3 ; un jour
  concerné manqué la remet à 0.
- **CA11** (F12, F13) — Sans session, `GET /` redirige vers `/login` ; avec un mauvais mot
  de passe, `401` ; avec le bon, la session dure 30 jours et le cookie a les attributs
  attendus. Changer `ADMIN_PASSWORD_HASH` invalide les sessions.
- **CA12** (F14) — La 11ᵉ tentative depuis une même IP dans l'heure renvoie `429`, même
  avec le bon mot de passe.
- **CA13** (NF1, NF2) — Les en-têtes CSP / `nosniff` / `noindex` sont présents ; un `POST`
  avec `Origin` étranger renvoie `403` ; un nom d'habitude contenant `<script>` s'affiche
  comme du texte.
- **CA14** (NF3) — À 360 px de large, la page du jour ne défile pas horizontalement et les
  cases font au moins 44 px (revue manuelle sur téléphone).
- **CA15** (F15, production) — `https://habits.zazi.cool/healthz` et `/readyz` répondent
  `200`, `/version` renvoie le SHA déployé, `http://` redirige vers `https://`,
  certificat valide.

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Erreurs de fuseau (jour décalé autour de minuit) | Moyen (cases au mauvais jour) | Moyenne | F6, horloge injectée dans les tests, CA4 |
| Règles de calcul (jours concernés, série) mal comprises | Moyen | Moyenne | Fonctions pures testées en table (CA5–CA10), ticket dédié |
| Données personnelles exposées | Moyen | Faible | Tout derrière la session, `noindex`, pas de page publique, sauvegarde `pg_dump` existante |
| Perte de motivation si l'écran est lourd | Faible | — | MVP minimal : une page, des cases |
| Réplique unique (limitation en mémoire) | Faible | — | Même compromis que `home` |

## 7. Décisions ouvertes

- [ ] **D1** — Sous-domaine : `habits.zazi.cool`. **Proposition :** `habits.zazi.cool`
  (bloque : T2, T7)
- [ ] **D2** — Habitudes à jours choisis (ex. lun / mer / ven) dès le MVP, ou seulement
  « tous les jours » ? **Proposition :** jours choisis inclus (déjà dans la spec) : sans
  cela, les taux hebdo / mensuels sont faussés pour les habitudes non quotidiennes
  (bloque : T3, T5)
- [ ] **D3** — Combien de jours en arrière peut-on corriger ? **Proposition :** 14 jours,
  réglable (`EDIT_WINDOW_DAYS`) (bloque : T4)
- [ ] **D4** — Authentification : mot de passe unique comme l'admin de l'accueil, ou autre
  (lien magique, passkey) ? **Proposition :** mot de passe unique, réutilise le modèle
  existant (bloque : T2)
- [ ] **D5** — Base `habits` dans `infra/data` : l'agent prépare la PR (mot de passe
  chiffré par Noah avec `sops edit`), Noah l'applique. **Proposition :** même procédure
  que pastebin (bloque : T7)
- [ ] **D6** — Faut-il une vue « tous les jours en un coup d'œil » sur la page d'accueil
  (mini-semaine) ? **Proposition :** non, vues séparées en MVP (bloque : rien)

## 8. Découpage en tickets

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | Squelette `apps/habits/` : `cmd/habits`, `/healthz` `/readyz` `/version`, arrêt propre, exécuteur de migrations, `Makefile`, `compose.yaml`, `Dockerfile`, `.env.example`, `README.md`, `CLAUDE.md` de l'app, workflow CI | F15, NF6 | — | dev + infra |
| T2 | 1 | Manifests `infra/apps/habits/{base,overlays/prod}` (Deployment, Service, Ingress, Middlewares, secret SOPS) et base `habits` dans `infra/data` | NF6, D1, D5 | — | infra |
| T3 | 2 | Authentification : sessions, `/login`, `/logout`, hash bcrypt, rate limiting, CSRF, en-têtes de sécurité, migration `sessions` | F12–F14, NF1, NF2, NF4 ; CA11–CA13 | T1 | dev |
| T4 | 3 | Habitudes et suivi du jour : migrations `habits`, `habit_schedules`, `completions` ; CRUD / archivage ; vue jour, cases, jours passés, fuseau | F1–F6, NF5 ; CA1–CA6 | T3 | dev |
| T5 | 4 | Statistiques : taux, série, vues semaine / mois / année | F7–F11 ; CA7–CA10 | T4 | dev |
| T6 | 4 | Finitions mobile : cases 44 px, clair / sombre, script de cochage sans rechargement, accessibilité | NF1, NF3 ; CA14 | T4 | dev |
| T7 | 5 | **Déployer habits `<sha7>`** : première mise en production (base, namespace, secret, `newTag`) selon la trajectoire de `docs/workflow/cycle-de-vie.md`, recette §9, ajout au service d'uptime, mise à jour du tableau du `CLAUDE.md` racine | CA15 | T1–T6 | déploiement |

T5 et T6 peuvent avancer en parallèle sur des branches distinctes. T1 et T2 sont
indépendants et peuvent démarrer ensemble. Un premier déploiement dès la fin de l'étape 3
est possible (suivi du jour seul), les statistiques suivant avec leur propre déploiement.

## 9. Recette après déploiement

- [ ] `curl -sI https://habits.zazi.cool/healthz` → `200` ; `/readyz` → `200` ;
      `curl -s https://habits.zazi.cool/version` → SHA déployé.
- [ ] `curl -sI http://habits.zazi.cool/` → redirection vers `https://`.
- [ ] Certificat Let's Encrypt valide pour `habits.zazi.cool`.
- [ ] Sans session, `/` redirige vers `/login` ; mauvais mot de passe refusé.
- [ ] Depuis un téléphone : se connecter, créer deux habitudes, cocher l'une, recharger,
      vérifier qu'elle reste cochée.
- [ ] Corriger un jour d'hier ; consulter les vues semaine, mois et année.
- [ ] Créer une habitude « lun / mer / ven » et vérifier les jours « non concernés ».
- [ ] Mettre à jour le tableau des projets du `CLAUDE.md` racine (statut « Production »).
