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

**Statut : en développement, application dynamique.** Le squelette (T1) et la page
publique (T3 : `/`, 404, fichiers statiques, Open Graph) sont en place. Démarrage local :
[`README.md`](README.md).

## Architecture

- **Back-end Go** (`net/http`) : `cmd/home` (point d'entrée, configuration, arrêt
  propre), `cmd/hashpw` (`make hash-password`), `internal/server` (routes HTTP, en-têtes
  de sécurité), `internal/content` (le document de la page, les règles d'affichage sans
  I/O et sa lecture en base ; liste des services), `internal/web` (gabarits, CSS, police
  et images embarqués, noms avec empreinte),
  `internal/migrate` (exécuteur de migrations), `internal/auth` (hash du mot de passe ;
  sessions à venir), `internal/dbtest` (base PostgreSQL de test dans un schéma temporaire).
- **Rendu serveur** (`html/template` + CSS), pas de React (D1). Seul le fond animé sera
  en TypeScript + Vite sans framework, dans `web/`, embarqué dans le binaire (T8).
- **PostgreSQL**, base `home` (`migrations/001_init.sql`) : `content` (une seule ligne :
  titre, paragraphe, choix de bannière, image téléversée, texte alternatif, fond,
  version), `images` (bannières téléversées), `sessions`.
- Un seul réplica (la limitation des tentatives de connexion sera en mémoire).

## Décisions closes

- [x] **Dynamique** (base PostgreSQL + admin protégée par mot de passe), et non plus une
  page statique sans base (spec, 2026-10-26).
- [x] Rendu serveur, thème `zazi` du linktree, fonds animés (D1, D2).
- [x] Liste des services **en dur dans le code** (D3).
- [x] Mot de passe d'admin propre à l'accueil, sessions séparées du linktree (D5).
- [x] Bannière au ratio 3:1 (2:1 sous 640 px), stockée à 1600 px de large au plus (D6).

## Règles durables (à respecter pour tout changement)

- **Code recopié du linktree** : l'exécuteur de migrations, `cmd/hashpw` et le hash du
  mot de passe viennent de `apps/linktree/` ; l'authentification, la limitation et le
  traitement d'image le seront aussi (T4, T6), simplifiés. Chaque app garde son module Go :
  **un correctif de sécurité dans l'une doit être reporté dans l'autre.**
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
- Base locale sur le port **5436**.

## Décisions ouvertes

Voir la spec, §7 :

- [ ] **D7** — Fonds proposés (`aurore`, `fils`, `grain`, `statique`) et couleurs.
- [ ] **D8** — Illustrations par défaut de la bannière.
