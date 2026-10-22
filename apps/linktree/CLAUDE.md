# Linktree — `link.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au linktree.
> Spécification de référence : [`docs/specs/linktree.md`](../../docs/specs/linktree.md)
> (exigences F/NF, critères d'acceptation, recette).

## Contexte

Pages de liens thématiques (`/musique`, `/jeux`…) : avatar, titre, description, liens,
groupes en accordéon, thème et fond animé. Tout est géré dans une administration en ligne
(`/admin`), sans redéploiement. Usage principalement mobile.

**Statut : Production, application dynamique** (déployée par Flux depuis
`infra/apps/linktree/overlays/prod`, tag d'image = SHA de la CI). Démarrage local :
[`README.md`](README.md).

## Architecture

- **Back-end Go** (`net/http`) : `cmd/linktree` (point d'entrée, tâche de purge),
  `internal/api` (routes publiques et `/admin/api`, types JSON), `internal/pages` (règles
  métier et validation, sans I/O), `internal/store` (PostgreSQL via `pgx`), `internal/auth`,
  `internal/images`, `internal/ratelimit`, `internal/render` (coquilles HTML).
- **Front React + TypeScript + Vite** dans `web/`, embarqué dans le binaire : deux
  entrées, `src/public` (visiteurs) et `src/admin`. Le serveur rend le HTML (titre, Open
  Graph, version sans JavaScript) et injecte les données dans `<script id="page-data">`.
- **PostgreSQL**, base `linktree` : `pages`, `items`, `images`, `sessions`
  (`migrations/001_init.sql`). Les images téléversées sont stockées en base.
- Un seul réplica (la limitation des tentatives est en mémoire).

## Décisions closes

- [x] Sous-domaine : **`link.zazi.cool`** seul (2026-10-14).
- [x] Stack : **Go**, front **React + TypeScript + Vite** embarqué.
- [x] **Dynamique** (base PostgreSQL + admin en ligne), pas de site statique.
- [x] **Plusieurs pages** thématiques, chacune avec son slug, ses liens et son thème.
- [x] Accès à l'admin : **mot de passe unique** (hash bcrypt en variable d'environnement),
  sessions en base de 30 jours.
- [x] Identité visuelle : cinq thèmes et huit accents, choisis par page dans l'admin.
- [x] Groupes : **un seul niveau** (un groupe ne contient que des liens).
- [x] Images : **JPEG et PNG** seulement, stockées en base, redimensionnées à 512 px.

## Règles durables (à respecter pour tout changement)

- **Les types JSON sont définis à un seul endroit**, `internal/api/types.go`, alignés avec
  `web/src/api/types.ts` par `contract_test.go` et `web/src/api/contract.json` : modifier
  les deux et la fixture ensemble.
- **Toute règle métier est validée côté Go** (`internal/pages/validate.go`) ; le front ne
  fait que guider. Les messages d'erreur sont en français.
- **Les brouillons ne quittent jamais le serveur** : `Page.Visible()` les retire avant le
  rendu public.
- **URL des liens : `http`, `https`, `mailto`, `tel` uniquement** (`ValidateURL`). Ne jamais
  assouplir : une URL `javascript:` ou `data:` s'exécuterait au clic.
- **Slugs réservés** (`admin`, `api`, `assets`, `images`, `shapes`, `healthz`, `readyz`,
  `version`) : tout nouveau chemin de premier niveau doit s'ajouter à `reservedSlugs`.
- **Enregistrement d'une page = remplacement du document entier**, en une transaction, avec
  versionnage optimiste (`version` ; 409 en cas de conflit).
- **Images** : tout passe par `images.Process` (format, dimensions, redimensionnement,
  ré-encodage qui supprime les métadonnées). Les images orphelines de plus de 24 h sont
  purgées chaque heure.
- **Sécurité** : CSP stricte (pas de script ni de style en ligne ; nonce pour l'admin),
  protection CSRF de l'admin, cookie de session `HttpOnly`/`SameSite=Strict`/`__Host-` en
  HTTPS, aucune ressource d'un tiers, aucun cookie visiteur. Changer
  `ADMIN_PASSWORD_HASH` invalide toutes les sessions.
- **Poids** : JavaScript initial de la page publique ≤ 150 Kio gzip, chaque fond animé
  ≤ 30 Kio gzip (`web/scripts/check-budget.mjs`, lancé par `make build`). Les fonds animés
  respectent `prefers-reduced-motion` et `Save-Data`.
- **Formes** : `web/public/shapes/` est généré par `npm run export-shapes` puis versionné ;
  l'application n'importe jamais `coolshapes-react`. Les fichiers sont la source de vérité
  des formes valides.
- **Configuration** : `PORT`, `DATABASE_URL`, `ADMIN_PASSWORD_HASH`, `PUBLIC_BASE_URL`,
  `TRUST_PROXY`, `RATE_LIMIT_LOGIN`, `MAX_UPLOAD_MB`, `VITE_DEV_URL` (voir `.env.example`).
  `TRUST_PROXY=true` seulement derrière Traefik.

## Décisions ouvertes

Évolutions possibles, aucune ne bloque l'existant (détail dans la spec, §7) :

- [ ] **Statistiques de clics** sans cookie ni pisteur. Proposition : non pour l'instant.
- [ ] **Liens programmés** (publication ou masquage à date). Proposition : non pour
  l'instant.
