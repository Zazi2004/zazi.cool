# Accueil — `zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre à la page d'accueil.
> Spécification de référence : [`docs/specs/home.md`](../../docs/specs/home.md)
> (exigences F/NF, critères d'acceptation, découpage, recette).

## Contexte

Page d'accueil servie sur l'apex `zazi.cool` : un **hub** vers les services du site
(Undercover, Pastebin, Liens), avec une bannière, un titre et un paragraphe que Noah
modifie dans une **mini administration** (`/admin`), sans redéploiement, sur un fond animé.
`www.zazi.cool` redirige en 301 vers `zazi.cool` (middleware Traefik `redirectRegex`, sans
logique applicative).

**Statut : en développement, application dynamique.** Le squelette (T1), la page
publique (T3 : `/`, 404, fichiers statiques, Open Graph) et l'authentification de l'admin
(T4 : connexion et déconnexion de `/admin`), le formulaire d'édition (T5 : titre,
paragraphe, texte alternatif, fond) et la bannière téléversée (T6 : `/images/{id}`,
purge) sont en place. Démarrage local :
[`README.md`](README.md).

## Architecture

- **Back-end Go** (`net/http`) : `cmd/home` (point d'entrée, configuration, arrêt
  propre), `cmd/hashpw` (`make hash-password`), `internal/server` (routes HTTP, en-têtes
  de sécurité), `internal/content` (le document de la page, les règles d'affichage sans
  I/O et sa lecture en base ; liste des services), `internal/web` (gabarits, CSS, police
  et images embarqués, noms avec empreinte), `internal/migrate` (exécuteur de
  migrations), `internal/auth` (hash du mot de passe, sessions en base),
  `internal/ratelimit` (limitation par IP en mémoire), `internal/images` (vérification,
  redimensionnement et ré-encodage des bannières), `internal/dbtest` (base PostgreSQL
  de test dans un schéma temporaire).
- **Rendu serveur** (`html/template` + CSS), pas de React (D1). Seul le fond animé sera
  en TypeScript + Vite sans framework, dans `web/`, embarqué dans le binaire (T8).
- **PostgreSQL**, base `home` (`migrations/001_init.sql`) : `content` (une seule ligne :
  titre, paragraphe, choix de bannière, image téléversée, texte alternatif, fond,
  version), `images` (bannières téléversées), `sessions`.
- Un seul réplica (la limitation des tentatives de connexion est en mémoire).

## Décisions closes

- [x] **Dynamique** (base PostgreSQL + admin protégée par mot de passe), et non plus une
  page statique sans base (spec, 2026-10-26).
- [x] Rendu serveur, thème `zazi` du linktree, fonds animés (D1, D2).
- [x] Liste des services **en dur dans le code** (D3).
- [x] Mot de passe d'admin propre à l'accueil, sessions séparées du linktree (D5).
- [x] Bannière au ratio 3:1 (2:1 sous 640 px), stockée à 1600 px de large au plus (D6).

## Règles durables (à respecter pour tout changement)

- **Code recopié du linktree** : l'exécuteur de migrations, `cmd/hashpw`, le hash du
  mot de passe, les sessions (`internal/auth/session.go`, sans interface de stockage),
  la limitation (`internal/ratelimit`), `clientIP` et le traitement d'image
  (`internal/images`, simplifié : 1600 px de large, 4000 px en entrée) viennent de `apps/linktree/`. L'origine est notée en tête des fichiers.
  Chaque app garde son module Go : **un correctif de sécurité dans l'une doit être reporté dans l'autre.**
- **Le contenu est une seule ligne** (`content.id = 1`) ; `banner_choice = 'upload'`
  exige `upload_image_id` (contrainte en base). Les illustrations par défaut ne sont pas
  en base : leur liste est une constante du code.
- **`/healthz` ne consulte jamais la base** (une panne de base ne doit pas faire
  redémarrer le pod) ; `/readyz` la consulte.
- **Configuration** : `PORT`, `DATABASE_URL`, `ADMIN_PASSWORD_HASH`, `PUBLIC_BASE_URL`,
  `TRUST_PROXY`, `RATE_LIMIT_LOGIN`, `MAX_UPLOAD_MB` (voir `.env.example`), toutes
  vérifiées au démarrage. `TRUST_PROXY=true` seulement derrière Traefik.
- **Page publique sans JavaScript ni style en ligne** : la CSP de NF5
  (`internal/server/middleware.go`) interdit tout `<script>` en ligne et tout attribut
  `style=""` ; la mise en forme passe par `home.css`. Le paragraphe est du texte brut
  (`content.Paragraphs`) : ne jamais le marquer `template.HTML`.
- **Le contenu est lu en base à chaque requête** (NF9), sans cache applicatif ; la page
  HTML est servie en `Cache-Control: no-cache`.
- **Thème** : les couleurs du thème `zazi` du linktree (accent violet) sont recopiées dans
  `internal/web/static/assets/home.css` (variables `:root`, clair puis sombre) ;
  `contrast_test.go` vérifie l'AA de chaque couleur de texte sur le fond et sur les cartes.
  `--accent-text` (sombre : `#c4b5fd`) remplace l'accent pour le texte, trop peu contrasté
  sur une carte survolée.
- **Fichiers statiques** : ceux de `static/assets/` ont une empreinte dans leur nom et un
  cache d'un an ; `favicon.svg`, `og-default.png` et `robots.txt` gardent leur nom (cache
  d'un jour). Les SVG ont leur propre CSP (`svgCSP`, styles en ligne des Coolshapes).
- **Poids** : HTML + CSS + police ≤ 80 Kio (`TestPageWeight`).
- **Corps de requête bornés** (NF8) : 16 Kio, sauf `POST /admin` (`MAX_UPLOAD_MB` + 64 Kio).
- **Admin** : toutes ses réponses portent `Cache-Control: no-store`, `noindex` et la CSP
  de NF5, sans aucun script. Chaque `POST` d'admin passe par `sameOrigin` (403 si
  `Sec-Fetch-Site: cross-site` ou `Origin` ≠ `PUBLIC_BASE_URL`, NF6) ; `POST /admin/login`
  passe en plus par la limitation, **avant** la vérification du mot de passe (F20).
- **Sessions** : jeton de 256 bits, seule son empreinte SHA-256 en base ; chaque session
  garde l'empreinte de `ADMIN_PASSWORD_HASH` (F19). Cookie `__Host-session` en HTTPS,
  `session` en HTTP local. Purge des sessions expirées toutes les heures (`cmd/home`).
- **Édition** (`POST /admin`) : les règles de F13 sont dans `content.Edit.Validate` (sans
  I/O, toutes les erreurs d'un coup, 422) ; `Store.Save` est un seul `UPDATE … WHERE
  version = $n` (verrou optimiste, `ErrConflict` → 409). Le message de succès passe par
  `/admin?saved=1`. Le formulaire est en `multipart/form-data` et garde la `version` lue.
- **Bannière téléversée** (F14–F16) : `images.Process` lit les dimensions avant de
  décoder (4000 × 4000 au plus → 422), redimensionne à 1600 px et ré-encode (plus aucune
  métadonnée). Le fichier est mesuré à part (`MAX_UPLOAD_MB` → 413), car la limite du
  corps laisse 64 Kio de marge ; le parseur multipart garde tout en mémoire (pas de
  disque inscriptible). `Store.Save` insère l'image et met à jour `content` dans **une
  transaction**. Un fichier envoyé exige le choix `upload` (422 sinon). Passer à « Aucune »
  garde `upload_image_id` : « Mon image » revient sans nouvel envoi. `/images/{id}` est
  `immutable` (un envoi = un nouvel identifiant, jamais de réécriture) ; la purge horaire
  (`cmd/home`) supprime les images de plus de 24 h que `content` ne référence pas.
- Base locale sur le port **5436**.

## Décisions ouvertes

Aucune : D7 (fonds `aurore`, `fils`, `grain`, `statique`) et D8 (6 illustrations
abstraites) ont été tranchées par Noah le 2026-10-26 (spec, §7).
