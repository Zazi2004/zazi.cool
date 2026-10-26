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

**Statut : en développement, application dynamique.** Le squelette (T1) et
l'authentification de l'admin (T4) sont en place : configuration, `/healthz`, `/readyz`,
`/version`, migrations, connexion et déconnexion de `/admin`. Démarrage local :
[`README.md`](README.md).

## Architecture

- **Back-end Go** (`net/http`) : `cmd/home` (point d'entrée, configuration, arrêt
  propre), `cmd/hashpw` (`make hash-password`), `internal/server` (routes HTTP),
  `internal/migrate` (exécuteur de migrations), `internal/auth` (hash du mot de passe,
  sessions en base), `internal/ratelimit` (limitation par IP en mémoire),
  `internal/dbtest` (base PostgreSQL de test dans un schéma temporaire).
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
  la limitation (`internal/ratelimit`) et `clientIP` viennent de `apps/linktree/` ; le
  traitement d'image le sera aussi (T6). L'origine est notée en tête des fichiers.
  Chaque app garde son module Go : **un correctif de sécurité dans l'une doit être reporté dans l'autre.**
- **Le contenu est une seule ligne** (`content.id = 1`) ; `banner_choice = 'upload'`
  exige `upload_image_id` (contrainte en base). Les illustrations par défaut ne sont pas
  en base : leur liste est une constante du code.
- **`/healthz` ne consulte jamais la base** (une panne de base ne doit pas faire
  redémarrer le pod) ; `/readyz` la consulte.
- **Configuration** : `PORT`, `DATABASE_URL`, `ADMIN_PASSWORD_HASH`, `PUBLIC_BASE_URL`,
  `TRUST_PROXY`, `RATE_LIMIT_LOGIN`, `MAX_UPLOAD_MB` (voir `.env.example`), toutes
  vérifiées au démarrage. `TRUST_PROXY=true` seulement derrière Traefik.
- **Admin** : toutes ses réponses portent `Cache-Control: no-store`, `noindex` et la CSP
  de NF5, sans aucun script. Chaque `POST` d'admin passe par `sameOrigin` (403 si
  `Sec-Fetch-Site: cross-site` ou `Origin` ≠ `PUBLIC_BASE_URL`, NF6) ; `POST /admin/login`
  passe en plus par la limitation, **avant** la vérification du mot de passe (F20).
- **Sessions** : jeton de 256 bits, seule son empreinte SHA-256 en base ; chaque session
  garde l'empreinte de `ADMIN_PASSWORD_HASH` (F19). Cookie `__Host-session` en HTTPS,
  `session` en HTTP local. Purge des sessions expirées toutes les heures (`cmd/home`).
- Base locale sur le port **5436**.

## Décisions ouvertes

Voir la spec, §7 :

- [ ] **D7** — Fonds proposés (`aurore`, `fils`, `grain`, `statique`) et couleurs.
- [ ] **D8** — Illustrations par défaut de la bannière.
