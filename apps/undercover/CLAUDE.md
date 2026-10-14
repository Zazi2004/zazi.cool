# Undercover — `undercover.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au jeu.

## Contexte

Jeu d'ambiance inspiré d'**Undercover**, joué autour d'une table avec **un seul téléphone
qui circule** entre les joueurs. Tous les joueurs reçoivent le même mot, sauf un ou
plusieurs **undercovers** (mot proche mais différent) et éventuellement un ou plusieurs
**Mr. White** (aucun mot). Les indices et le vote se font **à l'oral** ; l'application
distribue les mots, indique l'ordre de passage, enregistre les éliminations, révèle les
rôles et détermine la fin de partie.

Le second pilier du projet est une **banque de paires de mots enrichie par les joueurs**,
avec une catégorie « extrême » séparée pour que le jeu reste tout public par défaut.

**Statut : en développement** (spécifications validées le 2026-10-14).

---

## 1. Règles du jeu

### Rôles

| Rôle        | Mot reçu               | Connaît son rôle ?                           |
|-------------|------------------------|----------------------------------------------|
| Civil       | mot A                  | Non (voit seulement son mot)                 |
| Undercover  | mot B, proche de A     | **Non** : il le découvre à son élimination   |
| Mr. White   | aucun                  | Oui (« Tu es Mr. White »)                    |

- Pour chaque partie, une paire est tirée au hasard ; **le mot attribué aux civils est
  tiré au sort** entre les deux mots de la paire (sinon on finit par reconnaître le « mot
  undercover » d'une paire).
- Undercovers et Mr. White forment le camp des **imposteurs**.

### Déroulé d'une partie

1. **Réglages** (voir §2), puis saisie des prénoms **dans le sens de la table**.
2. **Distribution** : le téléphone passe de main en main. Chaque joueur touche son nom,
   voit son mot (ou son rôle de Mr. White), puis le masque avant de passer le téléphone.
   Une fois tous les mots vus, la partie commence.
3. **Tour** : l'application affiche l'ordre de passage, **entièrement tiré au sort à chaque
   tour** parmi les joueurs en vie. Le premier de la liste respecte les réglages « ne peut
   pas commencer ». Chacun donne un indice à l'oral.
4. **Vote** à l'oral ; on désigne dans l'application le joueur éliminé et on confirme.
   Un bouton **« Personne n'est éliminé »** (égalité, indécision) passe directement au tour
   suivant, sans révélation.
5. **Révélation** : l'application affiche le **rôle** du joueur éliminé (civil, undercover
   ou Mr. White), pas son mot.
6. Si c'est un **Mr. White**, il saisit le mot qu'il pense être celui des civils (voir
   ci-dessous).
7. Vérification des conditions de victoire ; sinon, nouveau tour (retour à l'étape 3).
8. **Fin de partie** : camp gagnant, rôle et mot de chaque joueur, les deux mots de la
   paire, bouton « Rejouer » (mêmes joueurs et réglages, nouvelle paire) et boutons de
   signalement de la paire (voir §3).

**Revoir son mot** : pendant la partie, un joueur en vie peut revoir son mot (ou son rôle
de Mr. White) en touchant son nom, après un écran de confirmation « Seul ⟨prénom⟩ doit
regarder » pour éviter un affichage accidentel.

### Deviner le mot (Mr. White éliminé)

- Champ texte libre ; la réponse est comparée au mot des civils **côté serveur**.
- Comparaison tolérante : insensible à la casse, aux accents et aux espaces superflus,
  articles initiaux ignorés (le, la, les, l', un, une, des, du), pluriel simple (`s`/`x`
  final) ignoré, et **une faute de frappe tolérée** (distance de Levenshtein ≤ 1) pour les
  mots de 5 lettres ou plus.
- La réponse et le mot attendu sont affichés à tous. Si l'application refuse une réponse
  que le groupe juge correcte, un bouton **« Valider quand même »** permet de corriger.
- Réponse correcte → **ce Mr. White gagne seul**, la partie s'arrête immédiatement. Réponse
  fausse → la partie continue. Avec plusieurs Mr. White, c'est donc **le premier qui devine**
  qui l'emporte.

### Conditions de victoire

Vérifiées après chaque élimination (et après la réponse d'un Mr. White éliminé) :

- **Mr. White** : un Mr. White éliminé devine le mot des civils → il gagne seul.
- **Civils** : tous les undercovers et Mr. White ont été éliminés.
- **Imposteurs** : il ne reste plus qu'**un seul civil** en vie.

---

## 2. Réglages d'une partie

| Réglage                                  | Valeurs                 | Défaut              |
|------------------------------------------|-------------------------|---------------------|
| Nombre de joueurs                        | 3 à 20                  | 5                   |
| Nombre d'undercovers                     | 0 à …                   | 1                   |
| Nombre de Mr. White                      | 0 à …                   | 0                   |
| Mr. White ne peut pas commencer un tour  | case à cocher           | cochée              |
| Undercover ne peut pas commencer un tour | case à cocher           | décochée            |
| Mode extrême                             | case à cocher           | **décochée**        |
| Uniquement les paires proposées par les joueurs | case à cocher    | décochée            |

Contraintes (validées à la saisie) :
- au moins **1 imposteur** (undercover ou Mr. White) ;
- les **civils sont strictement plus nombreux** que les imposteurs, et au moins 2 ;
- prénoms : 1 à 20 caractères, non vides, uniques (sans tenir compte de la casse).

Avec ces contraintes, il existe toujours un civil en vie pour commencer un tour tant que la
partie n'est pas terminée : les deux cases « ne peut pas commencer » sont donc compatibles.

Tirage de la paire (les paires masquées ne sont jamais tirées) :
- **Mode extrême décoché** : seules les paires tout public sont tirées.
- **Mode extrême coché** : les paires tout public **et** extrêmes sont tirées.
- **« Uniquement les paires proposées par les joueurs » coché** : le tirage se limite aux
  suggestions (`source = 'suggestion'`), en respectant toujours le mode extrême.
- Si aucune paire ne correspond aux réglages, la partie ne démarre pas et un message
  invite à changer les réglages ou à proposer des paires.
- Pour limiter les répétitions, le navigateur mémorise les paires déjà jouées et le serveur
  les exclut du tirage. Quand toutes les paires possibles ont été jouées, l'exclusion est
  ignorée (les répétitions recommencent plutôt que de bloquer la partie).

---

## 3. Banque de paires de mots

### Suggestion d'une paire (page « Proposer une paire »)

- Deux champs (mot 1, mot 2) et un choix **obligatoire, sans valeur par défaut** :
  « Tout public » ou « Extrême ». La personne doit choisir explicitement.
- Validation côté serveur : 1 à 40 caractères par mot ; lettres (accentuées comprises),
  chiffres, espaces, tirets et apostrophes ; les deux mots doivent être différents une fois
  normalisés.
- **Vérification de redondance** en direct pendant la saisie, puis à nouveau à l'envoi
  (voir ci-dessous).
- Une suggestion acceptée est **publiée immédiatement** et entre dans le tirage.
- Anti-abus : rate limiting par IP (voir §4), champ piège (honeypot) contre les robots.

### Détection de redondance

Chaque mot est **normalisé** (minuscules, sans accents, espaces réduits, article initial
retiré, pluriel simple retiré). Une paire est **non ordonnée** : « chat / tigre » et
« tigre / chat » sont la même paire.

| Niveau      | Cas                                                                  | Effet                                   |
|-------------|----------------------------------------------------------------------|-----------------------------------------|
| Doublon     | Mêmes mots normalisés, dans n'importe quel ordre                     | **Refus**, la paire existante est affichée |
| Similaire   | Un mot identique et l'autre à une faute près, ou les deux mots à une faute près | **Avertissement** avec les paires proches ; envoi possible après confirmation |
| Information | Un des mots apparaît déjà dans d'autres paires                       | Affichage des paires concernées         |

- La comparaison se fait **en Go, en mémoire** : la banque compte au plus quelques milliers
  de paires, la parcourir entièrement prend une fraction de milliseconde. Cela évite
  d'activer les extensions PostgreSQL `unaccent`/`pg_trgm`, qui demandent des droits
  superutilisateur sur l'instance partagée (`infra/data`).

### Signalement

En fin de partie (et sur une paire affichée lors d'une vérification de redondance), deux
boutons. **Un seul signalement suffit**, avec effet immédiat :

| Signalement              | Effet                                                        |
|--------------------------|--------------------------------------------------------------|
| « Devrait être extrême » | La paire passe en extrême (sort du mode tout public)         |
| « À supprimer »          | La paire est masquée : elle n'est plus jamais tirée (contenu inacceptable, paire absurde, faute…) |

- Un signalement par IP, par paire et par type.
- Une paire masquée reste en base (traçabilité) et **compte toujours pour la détection de
  doublons** : on ne peut pas la reproposer telle quelle.
- **Pas d'annulation** de signalement pour l'instant (pas d'interface d'admin).

### Base de départ

- **100 paires** : **75 tout public et 25 extrêmes**, en français, dans le fichier versionné
  `internal/seed/pairs.json`. Il est embarqué dans le binaire et inséré **à chaque démarrage,
  de façon idempotente** (`ON CONFLICT DO NOTHING`) : ajouter une paire au fichier suffit à
  l'ajouter en base, et une paire masquée par signalement n'est jamais réinsérée. Une SQL
  écrite à la main aurait demandé de calculer les mots normalisés hors de Go.
- La banque grandit ensuite par les suggestions des joueurs.
- Une bonne paire : deux mots **proches mais distincts** (même univers, assez de points
  communs pour que les indices se recoupent), connus du grand public. Exemples :
  « chat / tigre », « plage / piscine », « Mario / Luigi ».
- Extrême : sujets sensibles, politiques, religieux, sous-entendus, humour noir,
  personnalités publiques (ex. « Israël / Palestine », « Netanyahou / Superman »).
  Exclus même en extrême : contenus haineux visant un groupe, personnes privées.
- La base de départ passe par la même validation et la même détection de redondance
  (`internal/seed/seed_test.go` : au moins 100 paires dont 25 extrêmes, aucun doublon ni
  paire similaire).

---

## 4. Architecture

### Répartition client / serveur

Le jeu se joue sur **un seul appareil** : il n'y a pas d'adversaire entre le navigateur et
le serveur (le joueur qui tient le téléphone pourrait de toute façon regarder l'écran).

- **Front TypeScript + Preact (`web/`)** : moteur de partie (réglages, distribution, ordre de
  passage, éliminations, conditions de victoire) sous forme de **fonctions pures testées**
  (Vitest). L'état de la partie est conservé dans le navigateur (`localStorage`) pour
  survivre à un rechargement ou à la mise en veille du téléphone.
- **Serveur Go** : banque de paires, tirage, vérification de la réponse de Mr. White,
  suggestions, redondance, signalements. Il **valide tout ce qui touche aux données
  partagées**. Il ne voit jamais les prénoms des joueurs.

> **Exception à la convention racine** « toute règle métier est validée côté Go »
> (validée le 2026-10-14) : le moteur de partie tourne côté client, faute d'adversaire à
> arbitrer en mode « un téléphone ». Les règles qui touchent aux données partagées
> (suggestions, signalements, redondance) et la vérification de Mr. White restent en Go.
> Si un mode « un téléphone par joueur » est ajouté un jour, le moteur devra passer côté
> serveur.

Conséquences : **pas de temps réel** (ni WebSocket ni SSE), **pas de Redis**, serveur
**sans état** (plusieurs répliques possibles, une seule suffit).

### API HTTP (JSON)

| Méthode et chemin                      | Rôle                                                             |
|----------------------------------------|------------------------------------------------------------------|
| `GET  /api/pairs/random`               | Tire une paire (`extreme`, `suggestionsOnly`, `exclude=<ids>`) → `id`, `civilWord`, `undercoverWord` ; 404 si aucune paire ne correspond |
| `POST /api/pairs/{id}/guess`           | Compare la réponse de Mr. White au mot des civils → `correct`    |
| `GET  /api/pairs/check`                | Vérifie la redondance d'une paire (`a`, `b`) sans l'enregistrer  |
| `POST /api/pairs`                      | Propose une paire (201, 409 si doublon, 422 si invalide)         |
| `POST /api/pairs/{id}/reports`         | Signale une paire (`reason: "extreme" \| "remove"`)              |
| `GET  /healthz`, `GET /readyz`         | Santé (liveness / readiness, base joignable)                     |

Les messages sont définis dans `internal/api/types.go`, et leur miroir TypeScript dans
`web/src/api/types.ts`. Pour les garder alignés, un test Go génère et compare
`web/src/api/contract.json` (un exemple de chaque message, tous les champs remplis), et un
test Vitest vérifie que les types TS produisent exactement la même structure. Après une
modification d'un message : `go test ./internal/api -run TestContract -update`, puis
adapter `types.ts` jusqu'à ce que `npm test` passe.

`GET /api/pairs/random` indique quel mot de la paire est celui des civils (`civilSide:
"a" | "b"`, tiré au sort par le serveur). `POST /api/pairs/{id}/guess` le renvoie, et le
serveur retrouve le mot lui-même.

**Rate limiting** applicatif par IP (fenêtre glissante d'une heure, en mémoire : une
réplique suffit), limites configurables par variables d'environnement :

| Endpoint                         | Limite par défaut | Variable                     |
|----------------------------------|-------------------|------------------------------|
| `POST /api/pairs`                | 20 / heure        | `RATE_LIMIT_SUGGESTIONS`     |
| `POST /api/pairs/{id}/reports`   | 30 / heure        | `RATE_LIMIT_REPORTS`         |
| `GET /api/pairs/check`           | 300 / heure       | `RATE_LIMIT_CHECKS`          |

Dépassement → `429 Too Many Requests`. L'IP réelle est lue dans `X-Forwarded-For` posé
par Traefik (seulement derrière le proxy de confiance).

### Pages

- `/` : accueil (Nouvelle partie, Proposer une paire, Règles).
- `/partie` : déroulé complet d'une partie.
- `/proposer` : suggestion de paire avec vérification en direct.
- `/regles` : règles du jeu.

Interface en français, **mobile-first**, utilisable d'une main, mode sombre. Les mots
proposés par les joueurs sont du contenu non fiable : affichage par `textContent`
uniquement, CSP stricte.

### Données (PostgreSQL)

```sql
CREATE TABLE word_pairs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    word_a      text        NOT NULL,           -- tel que saisi
    word_b      text        NOT NULL,
    norm_a      text        NOT NULL,           -- normalisé, norm_a < norm_b
    norm_b      text        NOT NULL,
    extreme     boolean     NOT NULL DEFAULT false,
    hidden      boolean     NOT NULL DEFAULT false, -- signalée « à supprimer »
    source      text        NOT NULL CHECK (source IN ('seed', 'suggestion')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (norm_a < norm_b),
    UNIQUE (norm_a, norm_b)                     -- doublons impossibles, même en concurrence
);

CREATE TABLE pair_reports (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pair_id        bigint      NOT NULL REFERENCES word_pairs (id) ON DELETE CASCADE,
    reason         text        NOT NULL CHECK (reason IN ('extreme', 'remove')),
    reporter_hash  text        NOT NULL,        -- hash salé de l'IP, jamais l'IP en clair
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (pair_id, reason, reporter_hash)
);
```

Le tri `norm_a < norm_b` rend la paire non ordonnée : la contrainte `UNIQUE` suffit à
garantir l'absence de doublon exact. Un signalement insère la ligne dans `pair_reports` et
met à jour `extreme` ou `hidden` **dans la même transaction**. Le sel du hash d'IP est un
secret (variable d'environnement).

Migrations : exécuteur maison commun aux apps (voir `CLAUDE.md` racine §5), appliqué au
démarrage du serveur et par `make migrate`.

### Structure

```
apps/undercover/
├── cmd/undercover/main.go   # configuration, migrations + seed, serveur, arrêt propre
├── internal/
│   ├── api/          # handlers HTTP, middlewares, types des messages, test de contrat
│   ├── migrate/      # exécuteur de migrations (verrou consultatif PostgreSQL)
│   ├── pairs/        # normalisation, redondance, comparaison Mr. White (logique pure)
│   ├── ratelimit/    # limitation par IP, fenêtre glissante en mémoire
│   ├── seed/         # base de départ (pairs.json, embarqué)
│   └── store/        # accès PostgreSQL (pgx) + tests d'intégration
├── migrations/       # schéma SQL, embarqué
└── web/              # Vite + TypeScript + Preact
    ├── embed.go      # embarque web/dist dans le binaire Go
    └── src/
        ├── api/      # types des messages, client HTTP, contract.json
        ├── game/     # moteur de partie (fonctions pures) + persistance localStorage
        └── ui/       # pages et écrans
```

### Dépendances

- Go : **`github.com/jackc/pgx/v5`** (pilote PostgreSQL, aucun pilote dans la bibliothèque
  standard) et `golang.org/x/text` (retrait des accents, maintenu par l'équipe Go et déjà
  requis par pgx).
- Front : **`preact`** ; outillage de dev : `vite`, `@preact/preset-vite`, `typescript`,
  `vitest`, `prettier`.
- `web/.npmrc` force le registre public npm : sans lui, une configuration npm globale vers
  un registre privé finirait dans `package-lock.json` et casserait le build Docker et la CI.

### Développement local : particularités

- `make dev` : rechargement à chaud du **front** (Vite, port 5173, qui relaie `/api` vers
  le serveur Go). Le serveur Go n'est **pas** rechargé automatiquement : relancer
  `make dev` après une modification Go (pas d'outil type `air`, pour éviter une dépendance).
- La base locale écoute sur le port **5433** de l'hôte (`DB_PORT`), pour ne pas entrer en
  conflit avec un autre PostgreSQL local.
- Les tests d'intégration (`internal/store`) créent un schéma PostgreSQL temporaire par
  test et le suppriment ensuite : ils ne touchent pas aux données de développement. Sans
  `TEST_DATABASE_URL`, ils sont ignorés (`make test` la renseigne).

---

## 5. Hors périmètre du MVP (pistes ultérieures)

- Mode « un téléphone par joueur » (salon, code, vote dans l'application).
- Scores cumulés sur plusieurs parties.
- Thèmes / catégories de mots (animaux, cuisine…) et filtrage par thème.
- Interface d'administration (corriger, fusionner ou supprimer une paire).
- Jeu hors ligne (banque mise en cache par le navigateur).

---

## 6. Décisions

### Prises (2026-10-14)

- [x] Sous-domaine : **`undercover.zazi.cool`**.
- [x] Stack : **Go** côté serveur, **TypeScript + Vite** côté front.
- [x] Mode de jeu : **un seul téléphone qui circule** ; indices et vote à l'oral.
- [x] Les undercovers **ne connaissent pas** leur rôle ; Mr. White sait qu'il n'a pas de mot.
- [x] Réponse de Mr. White : **validation automatique tolérante**, correction possible par
      les joueurs.
- [x] Suggestions **publiées immédiatement**, avec catégorie (tout public / extrême)
      obligatoire.
- [x] Signalements « devrait être extrême » et « à supprimer », **effet immédiat dès un
      signalement**, pas d'annulation pour l'instant.
- [x] Victoire des imposteurs quand il ne reste qu'**un seul civil** ; avec plusieurs
      Mr. White, **le premier qui devine gagne seul**.
- [x] Bouton **« Personne n'est éliminé »** au moment du vote.
- [x] Ordre de passage **entièrement aléatoire** à chaque tour.
- [x] Un joueur peut **revoir son mot** (avec confirmation).
- [x] Front en **Preact** (dépendance validée).
- [x] Migrations par un **exécuteur maison** (`embed` + table `schema_migrations`).
- [x] Rate limiting applicatif par IP, limites configurables (voir §4).

- [x] Moteur de partie **côté client** (exception à la convention racine, voir §4).
- [x] Base de départ : **100 paires, dont 25 extrêmes**.
- [x] Réglage « **Uniquement les paires proposées par les joueurs** ».

### Ouvertes

Aucune pour le MVP.
