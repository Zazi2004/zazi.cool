# Spec — Pastebin (`pb.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **En revue** — décisions D1 à D5 tranchées le 2026-10-16 (spec pilote du workflow) |
| Issue Multica  | ONBO-1 (conception du workflow) ; une issue de cadrage dédiée sera créée à la validation |
| Application    | `apps/pastebin/` |
| Auteur, valideur | Mika / Noah |

> **Pourquoi le pastebin comme pilote ?** C'est le projet qui exerce **tout** le
> workflow : il part de zéro (cadrage → premier déploiement), il a une base de données
> (migrations, création de la base dans `infra/data`), un endpoint public d'écriture
> (rate limiting, abus), du contenu non fiable (XSS, CSP), une tâche de fond (purge) et
> des décisions à trancher. Undercover est déjà en production, l'accueil et le
> linktree sont statiques : ils n'auraient testé qu'une partie du cycle.

## 1. Contexte

Service personnel de partage de texte et de code : on colle un contenu, on obtient une
URL courte à partager. Le dossier `apps/pastebin/` ne contient aujourd'hui que son
`CLAUDE.md` (périmètre envisagé et points d'attention), dont cette spec reprend et
précise le MVP. Le DNS wildcard `*.zazi.cool` couvre déjà `pb.zazi.cool` ; l'app n'a ni
code, ni CI, ni manifests, ni base.

## 2. Objectifs

- **O1** — Partager un texte en moins de 10 secondes : coller, choisir une expiration,
  obtenir un lien.
- **O2** — Un lien reste illisible pour qui ne l'a pas (identifiant non devinable) et
  plus personne ne peut le lire une fois expiré.
- **O3** — Aucun contenu collé ne peut exécuter de script dans le navigateur d'un lecteur.
- **O4** — Le service tient sur le VPS partagé sans dégrader les autres apps, même sous
  un usage abusif modéré.

## 3. Périmètre

### Inclus (MVP)
- Création d'un paste (texte brut) depuis un formulaire web.
- Lecture : vue formatée (avec numéros de ligne) et vue brute.
- Coloration syntaxique avec choix du langage (`chroma`, D4).
- Expiration choisie à la création (**10 minutes par défaut**, D3) ; refus de servir un
  paste expiré ; purge.
- Taille maximale, rate limiting, champ piège (honeypot).

### Hors périmètre
- « Burn after reading », mot de passe, chiffrement côté client (D2) (pistes ultérieures
  du `CLAUDE.md` de l'app).
- Expiration « jamais » (D3) et création authentifiée (D1).
- Création via `curl` / API (piste ultérieure ; le modèle de données la permet déjà).
- Lien de suppression pour l'auteur, comptes utilisateurs, liste des pastes.
- Détection automatique du langage (choix manuel, « Texte brut » par défaut).

## 4. Exigences

### Fonctionnelles

- **F1** — La page `/` affiche un formulaire : zone de texte, langage (liste, défaut
  « Texte brut »), expiration (10 min, 1 h, 1 jour, 1 semaine, 1 mois ; **10 min
  présélectionné**, voir D3), bouton « Créer ».
- **F2** — À la création, le serveur génère un identifiant de **8 caractères** base62
  (`crypto/rand`, ~47 bits), enregistre le paste et redirige (`303`) vers `/<id>`.
  Une collision d'identifiant est réessayée côté serveur.
- **F3** — `GET /<id>` affiche le paste formaté (numéros de ligne, coloration `chroma`),
  la date d'expiration et un lien vers la vue brute et un bouton « Copier le lien ».
- **F4** — `GET /raw/<id>` renvoie le contenu tel quel en
  `Content-Type: text/plain; charset=utf-8`.
- **F5** — Un paste dont `expires_at` est passé renvoie **404** (vue formatée et brute),
  même s'il est encore en base. Un identifiant inconnu renvoie la même 404 (on ne
  distingue pas « expiré » de « inexistant »).
- **F6** — Les pastes expirés sont **supprimés** de la base au plus tard une heure après
  leur expiration.
- **F7** — Validation côté serveur : contenu non vide, UTF-8 valide, au plus
  `MAX_PASTE_BYTES` octets ; langage et expiration dans les listes autorisées. Erreur →
  formulaire réaffiché avec le contenu saisi et un message en français. Une expiration
  absente du formulaire vaut la valeur par défaut (10 minutes).
- **F8** — `GET /version` renvoie le SHA git de l'image en cours d'exécution (injecté au
  build), pour la vérification automatique des déploiements
  (`docs/workflow/cycle-de-vie.md`).

### Non fonctionnelles

- **NF1** — Sécurité du contenu : rendu uniquement via `html/template` (échappement
  automatique) ; CSP stricte (`default-src 'none'; style-src 'self'; img-src 'self';
  form-action 'self'; frame-ancestors 'none'` + `script-src 'self'` seulement si le
  bouton « Copier » demande du JS) ; vue brute avec `X-Content-Type-Options: nosniff`
  et `Content-Security-Policy: sandbox`.
- **NF2** — Anti-abus : rate limiting applicatif par IP sur `POST /` (défaut **10 / heure**),
  `429` au-delà ; champ honeypot (création ignorée silencieusement s'il est rempli).
  Les pastes ne sont pas indexés (`X-Robots-Tag: noindex`).
- **NF3** — Ressources : requests `50m` CPU / `32Mi` mémoire, limits `200m` / `128Mi`
  (à ajuster après mesure) ; corps de requête plafonné à `MAX_PASTE_BYTES` + marge
  (`http.MaxBytesReader`).
- **NF4** — Vie privée : aucun cookie, aucun tracker ; l'IP n'est jamais stockée en base
  (seulement en mémoire pour le rate limiting).

### Interfaces

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `GET /` | Formulaire de création | 200 |
| `POST /` | Crée un paste (formulaire `content`, `language`, `expires_in`, `website` = honeypot) | 303 → `/<id>` ; 400 / 413 / 422 formulaire avec erreur ; 429 |
| `GET /{id}` | Vue formatée | 200 ; 404 |
| `GET /raw/{id}` | Vue brute (`text/plain`) | 200 ; 404 |
| `GET /healthz`, `GET /readyz` | Santé (readyz : base joignable) | 200 ; 503 |
| `GET /version` | SHA de build (`text/plain`) | 200 |

Pages en **rendu serveur** (`html/template`), sans front TypeScript, conformément à la
décision commune (`CLAUDE.md` racine §9).

### Données

Table `pastes` (migration `001_create_pastes.sql`) :

| Colonne | Type | Note |
|---------|------|------|
| `id` | `text` primary key | 8 caractères base62 |
| `content` | `text` not null | ≤ `MAX_PASTE_BYTES` |
| `language` | `text` not null | clé de la liste autorisée, `plaintext` par défaut |
| `created_at` | `timestamptz` not null default `now()` | |
| `expires_at` | `timestamptz` not null | pas d'expiration « jamais » (D3) |

Index sur `expires_at`. Base `pastebin` et utilisateur dédié, créés par le Job
`create-databases` (`infra/data`). La purge (F6) est faite par l'application : une
goroutine exécute `DELETE FROM pastes WHERE expires_at < now()` toutes les 15 minutes —
plus simple qu'un `CronJob` (pas de second manifest, pas d'accès base supplémentaire) et
sans risque avec une seule réplique ; la requête est idempotente si on en ajoute.

### Configuration

| Variable | Rôle | Défaut |
|----------|------|--------|
| `PORT` | Port d'écoute | `8080` |
| `DATABASE_URL` | Connexion PostgreSQL | — (obligatoire) |
| `MAX_PASTE_BYTES` | Taille maximale d'un paste | `524288` (512 Kio) |
| `RATE_LIMIT_CREATE` | Créations par IP et par heure | `10` |
| `PURGE_INTERVAL` | Période de la purge | `15m` |
| `BASE_URL` | URL publique, pour les liens affichés | `https://pb.zazi.cool` |

Les durées d'expiration proposées et la valeur par défaut (10 minutes) sont des
constantes du code, pas des variables d'environnement : elles font partie de la spec.

## 5. Critères d'acceptation

- **CA1** (F1, F2) — Étant donné un texte valide et l'expiration laissée à sa valeur
  par défaut, quand je soumets le formulaire, alors je suis redirigé vers `/<id>` où `id`
  fait 8 caractères base62, le texte s'affiche à l'identique et `expires_at` vaut
  `created_at` + 10 minutes.
- **CA2** (F4) — Quand j'appelle `/raw/<id>`, alors je reçois exactement les octets
  envoyés, en `text/plain; charset=utf-8` avec `nosniff`.
- **CA3** (F5) — Étant donné un paste dont `expires_at` est dans le passé et pas encore
  purgé, quand je l'ouvre (formaté ou brut), alors je reçois une 404 identique à celle
  d'un identifiant inconnu.
- **CA4** (F6) — Étant donné des pastes expirés, après une exécution de la purge, alors
  ils ne sont plus en base et les pastes non expirés sont intacts (test d'intégration).
- **CA5** (F7) — Un contenu vide, de plus de `MAX_PASTE_BYTES` ou non UTF-8, ou une
  expiration hors liste, est refusé avec un message et rien n'est enregistré.
- **CA6** (NF1) — Un paste contenant `<script>alert(1)</script>` s'affiche comme du
  texte dans la vue formatée et aucun script ne s'exécute ; les en-têtes CSP attendus
  sont présents sur chaque page (test HTTP).
- **CA7** (NF2) — La 11ᵉ création depuis une même IP dans l'heure renvoie `429` ; une
  soumission avec le honeypot rempli ne crée rien.
- **CA8** (production) — `https://pb.zazi.cool/healthz` et `/readyz` répondent `200`,
  `/version` renvoie le SHA déployé, `http://` redirige vers `https://`, le certificat est
  valide.
- **CA9** (F1, D3) — Le formulaire propose exactement 10 min, 1 h, 1 jour, 1 semaine et
  1 mois, avec 10 min présélectionné ; une expiration hors liste (dont « jamais ») est
  refusée.

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Spam / contenus illicites hébergés sous `zazi.cool` | Élevé (réputation, signalements) | Moyenne si public | Rate limiting, taille max, pas d'expiration « jamais » et défaut à 10 min (D1, D3), `noindex` |
| XSS via contenu collé | Élevé | Faible avec `html/template` | NF1, CA6, review sécurité |
| Remplissage de la base / du disque du VPS | Moyen (toutes les apps partagent PostgreSQL) | Faible | Taille max, rate limiting, purge ; surveiller la taille de la base |
| Liens partagés expirés trop tôt (défaut 10 min) | Faible | Moyenne | Durée affichée sur le formulaire et la page du paste ; choix jusqu'à 1 mois |
| Énumération des identifiants | Moyen | Faible (~47 bits) | `crypto/rand`, 404 indistinctes, rate limiting global possible plus tard |
| Dépendance de coloration syntaxique (`chroma`) | Faible | — | Isolée dans T6 : le MVP peut être livré sans |

## 7. Décisions

Toutes tranchées par Noah le 2026-10-16.

- [x] **D1** — Création **publique**, avec les parades de NF2 ; un jeton (`CREATE_TOKEN`)
  pourra être ajouté plus tard si l'abus apparaît.
- [x] **D2** — **Pas de chiffrement côté client** pour le MVP (hors périmètre).
- [x] **D3** — Durées : 10 min, 1 h, 1 jour, 1 semaine, 1 mois ; **défaut 10 minutes** ;
  pas d'option « jamais ».
- [x] **D4** — Coloration syntaxique côté serveur avec
  [`chroma`](https://github.com/alecthomas/chroma) (dépendance approuvée), dans un ticket
  séparé (T6).
- [x] **D5** — Base `pastebin` dans `infra/data` : l'agent prépare la PR (mot de passe
  chiffré par Noah avec `sops edit`, ligne `create_app_db`), Noah l'applique.

## 8. Découpage en tickets

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | Squelette de l'app : `cmd/pastebin`, `/healthz` `/readyz` `/version` (SHA injecté au build), arrêt propre, exécuteur de migrations, `Makefile` (cibles standard), `compose.yaml`, `Dockerfile`, `.env.example`, `README.md` ; workflow CI `.github/workflows/pastebin.yml` calqué sur celui d'Undercover | F8, NF3 | — | dev + infra |
| T2 | 1 | Manifests `infra/apps/pastebin/{base,overlays/prod}` (Deployment, Service, Ingress `pb.zazi.cool`, Middlewares, secret SOPS) et base `pastebin` dans `infra/data` | NF3, D5 | — | infra |
| T3 | 2 | Création et lecture : formulaire, `POST /`, vue formatée, vue brute, validation, en-têtes de sécurité, migration `001_create_pastes.sql` | F1–F5, F7, NF1, NF4 ; CA1–CA3, CA5, CA6, CA9 | T1 | dev |
| T4 | 2 | Expiration (durées et défaut 10 min de D3) et purge périodique | F5, F6 ; CA3, CA4 | T3 (même étape, après T3) | dev |
| T5 | 2 | Anti-abus : rate limiting par IP et honeypot | NF2 ; CA7 | T1 | dev |
| T6 | 3 | Coloration syntaxique avec `chroma` | F3 | T3 | dev |
| T7 | 4 | **Déployer pastebin `<sha7>`** : premier déploiement (base dans `infra/data`, namespace, secret, `newTag`) selon l'étape de la trajectoire de déploiement en vigueur (`docs/workflow/cycle-de-vie.md`), recette §9, ajout au service d'uptime quand il sera choisi | CA8 | T1–T5 | déploiement |

T4 et T5 peuvent avancer en parallèle de T3 sur des branches distinctes ; T4 se rebase
sur T3. Le premier déploiement (T7) peut aussi se faire dès la fin de l'étape 2, T6
suivant alors avec son propre ticket de déploiement.

## 9. Recette après déploiement

- [ ] `curl -sI https://pb.zazi.cool/healthz` → `200` ; `/readyz` → `200` ;
      `curl -s https://pb.zazi.cool/version` → SHA déployé (vérifié par le job `verify`
      dès l'étape 2 de la trajectoire).
- [ ] `curl -sI http://pb.zazi.cool/` → redirection vers `https://`.
- [ ] Certificat Let's Encrypt valide pour `pb.zazi.cool`.
- [ ] Créer un paste depuis un téléphone, l'ouvrir depuis un autre appareil, ouvrir
      `/raw/<id>`.
- [ ] Coller `<script>alert(1)</script>` : affiché comme texte, en-tête CSP présent.
- [ ] Créer un paste avec l'expiration par défaut (10 min), vérifier la 404 après expiration.
- [ ] Mettre à jour le tableau des projets du `CLAUDE.md` racine (statut « Production »).
