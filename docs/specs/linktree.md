# Spec — Linktree (`link.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **En revue** — décisions prises avec Noah le 2026-10-14 ; restent D1 à D4 (§7) |
| Issue Multica  | ONBO-43 |
| Application    | `apps/linktree/` |
| Auteur, valideur | Claude (session de cadrage avec Noah) / Noah |

> Cette version remplace le premier brouillon de la PR #62, qui partait de l'ancien
> `CLAUDE.md` de l'app (page unique statique). Noah a choisi le 2026-10-14 une
> application dynamique à plusieurs pages, avec administration en ligne.

## 1. Contexte

Service de **pages de liens thématiques**, à la manière de Linktree, mais avec plusieurs
pages : chaque page rassemble des liens autour d'un thème et a sa propre URL
(`link.zazi.cool/musique`, `link.zazi.cool/jeux`…). Une page contient des liens simples
et des **groupes en accordéon** (ex. une musique et ses plateformes d'écoute). Les pages se
créent et se modifient dans une **administration en ligne** (`/admin`), sans redéployer.

Le **visuel est un objectif à part entière**. Références choisies par Noah :
[coolshap.es](https://coolshap.es/) (formes abstraites à dégradés granuleux) et
[reactbits.dev](https://reactbits.dev/) (fonds animés WebGL, textes animés,
micro-interactions). Usage surtout mobile, depuis un lien partagé.

Existant : le DNS wildcard couvre `link.zazi.cool` ; le job `linktree` de
`.github/workflows/ci.yml` est câblé et s'active dès qu'`apps/linktree/Dockerfile` existe.
Le serveur Go et les pages publiques sont déjà écrits sur la branche `feat/linktree-mvp`
(tickets T1 et T3, §8), à relire en PR après validation de cette spec.

## 2. Objectifs

- **O1** — Créer une page de liens (titre, URL, thème, liens, groupes) depuis un téléphone
  ou un ordinateur, sans toucher au code ni redéployer.
- **O2** — Une page publique soignée et vivante (thèmes, formes, fonds animés) qui reste
  lisible (contraste AA) et rapide (Lighthouse mobile ≥ 90 en performance, 100 en
  accessibilité).
- **O3** — Aucun brouillon, ni aucune donnée d'administration, n'atteint un visiteur.
- **O4** — Le service reste léger sur le VPS partagé (une réplique, base PostgreSQL
  partagée, pas de Node en production).

## 3. Périmètre

### Inclus (MVP)
- Pages publiques `/{slug}`, index `/` des pages publiées et listées, 404 soignée.
- Liens et groupes (accordéons), dans un ordre libre ; brouillons (page, groupe, lien).
- Administration protégée par mot de passe : liste et ordre des pages, éditeur avec
  aperçu en direct, upload d'images.
- Cinq thèmes prédéfinis + huit couleurs d'accent ; formes coolshapes ; fonds animés.
- Icônes de plateformes déduites de l'URL ; Open Graph ; version sans JavaScript.
- `/healthz`, `/readyz`, `/version`.

### Hors périmètre
- Statistiques de vues et de clics ; liens programmés ou masqués temporairement.
- Redirection automatique des anciens slugs ; titres de section entre les éléments.
- Import automatique (`og:image` du lien cible, plateformes via song.link / Odesli).
- Lecteurs intégrés, recadrage d'image, GIF animés, image Open Graph générée par page.
- Enregistrement automatique et historique des modifications ; plusieurs comptes.

## 4. Exigences

### Fonctionnelles

**Contenu**

- **F1** — Une **page** a : titre (1 à 80 caractères), slug, description (0 à 280,
  retours à la ligne permis), avatar optionnel (image uploadée **ou** forme coolshapes),
  thème, accent, statut publiée / brouillon (brouillon à la création), « listée » dans
  l'index (oui par défaut). Une forme est proposée au hasard à la création.
- **F2** — Le **slug** fait 1 à 40 caractères `a-z`, chiffres et tirets
  (`^[a-z0-9]+(-[a-z0-9]+)*$`), est unique, et ne peut pas être `admin`, `api`, `assets`,
  `images`, `shapes`, `healthz`, `readyz` ni `version`. L'admin le propose à partir du
  titre (accents retirés, espaces → tirets) ; changer celui d'une page publiée affiche un
  avertissement (pas de redirection dans le MVP).
- **F3** — Un **lien** a : titre (1 à 80), URL (2048 au plus, schémas `https`, `http`,
  `mailto`, `tel` uniquement ; aucun espace ni caractère de contrôle), image optionnelle,
  statut. Sans image, il affiche l'icône de sa plateforme (F11). Dans l'admin, le titre est
  pré-rempli avec le nom de la plateforme reconnue, et `https://` est ajouté côté client à
  une URL saisie sans schéma.
- **F4** — Un **groupe** (accordéon) a : titre, sous-titre (0 à 80), visuel optionnel
  (image ou forme), « ouvert par défaut », statut, et 0 à 30 liens. Pas de groupe dans un
  groupe ; un lien n'a pas de forme. Plusieurs groupes peuvent être ouverts à la fois.
  Replié, un groupe montre son visuel, son titre, son sous-titre et les icônes de ses
  plateformes ; un groupe sans lien publié n'est pas affiché.
- **F5** — Liens et groupes sont mélangés dans un ordre libre ; au plus 100 éléments au
  premier niveau et 500 au total par page.
- **F6** — **Brouillons** : une page en brouillon répond 404 ; un lien ou un groupe en
  brouillon est retiré **côté Go** avant tout envoi au navigateur.

**Pages publiques**

- **F7** — `/` liste les pages publiées et listées, dans l'ordre choisi dans l'admin ;
  `/{slug}` affiche une page ; tout le reste répond 404 avec une page soignée.
- **F8** — Une page non listée reste accessible par son URL mais porte `noindex`.
- **F9** — Go renvoie une page HTML qui contient déjà : `<title>`, description, balises
  Open Graph et Twitter Card (`og:image` = avatar image de la page, sinon
  `og-default.png`), `data-theme` et `data-accent` (les couleurs sont justes avant le JS),
  les données de la page en JSON, et une version `<noscript>` (liens et groupes en
  `<details>`). React dessine ensuite la page à partir de ces données, sans requête de plus.
- **F10** — Bouton **Partager** (feuille de partage du système, sinon copie de l'URL).
  Les liens s'ouvrent dans le même onglet.
- **F11** — Icônes de plateformes déduites du nom d'hôte ou du schéma : Spotify, Deezer,
  Apple Music, YouTube, YouTube Music, SoundCloud, Bandcamp, Tidal, Amazon Music, Qobuz,
  Instagram, TikTok, X, Bluesky, Threads, Facebook, Reddit, Twitch, Discord, GitHub,
  LinkedIn, Steam, itch.io, Epic Games, PlayStation, Xbox, Nintendo, e-mail, téléphone.
  Icônes Simple Icons (CC0) ; icône générique pour une marque absente (Amazon Music,
  Qobuz, LinkedIn, Xbox, Nintendo) ou un site inconnu.

**Visuel**

- **F12** — **Thèmes** : `zazi` (défaut, suit le mode clair/sombre du système, fond CSS
  et formes flottantes), `nuit` (aurore WebGL, cartes en verre dépoli), `papier` (clair,
  empattements, dégradé granuleux, formes en autocollants), `neon` (noir, fils lumineux
  WebGL, bordures qui brillent, titre balayé de lumière), `pop` (inspiré de coolshapes :
  dégradé vif, grande forme, boutons épais à ombre franche). **Accents** : violet, bleu,
  cyan, vert, ambre, orange, rose, neutre.
- **F13** — **Formes** : les 115 formes coolshapes (MIT, 10 familles), avec leur grain,
  servent d'avatar de page, de visuel de groupe et de décor des thèmes (choisi à partir de
  l'identifiant de la page : une page garde toujours le même décor). La 404 écrit « 404 »
  avec les formes chiffres.
- **F14** — **Fonds animés** (`nuit`, `papier`, `neon`, `pop`) : chargés après
  l'affichage du contenu, en fondu au-dessus du fond CSS du thème ; remplacés par ce fond
  CSS sans WebGL 2, avec `prefers-reduced-motion` ou `Save-Data` ; en pause onglet masqué
  ou hors écran ; densité de pixels plafonnée à 1,5. Au plus un fond WebGL par page.
- **F15** — **Animations** : arrivée en cascade (moins de 600 ms au total), titre qui sort
  d'un flou, ressort au toucher des liens, accordéon à ressort ; toutes coupées avec
  `prefers-reduced-motion`.

**Administration**

- **F16** — **Connexion** par un mot de passe unique ; toute URL `/admin/…` sans session
  ramène à `/admin/login`, puis à l'URL demandée. Déconnexion.
- **F17** — **Liste des pages** : titre, slug, statut, listée, nombre de liens, lien
  « Voir » ; créer (titre + slug), supprimer (confirmation), réordonner par
  glisser-déposer (ordre de l'index).
- **F18** — **Éditeur** : réglages de la page (slug vérifié en direct, avatar image ou
  forme via un sélecteur en grille, thème via des vignettes, accent) ; ajouter, modifier,
  dupliquer, passer en brouillon, supprimer liens et groupes ; glisser-déposer à
  l'intérieur d'un même niveau ; « Déplacer vers… » pour changer un lien de niveau.
- **F19** — **Aperçu en direct** dans un cadre de téléphone (onglet « Aperçu » sur
  mobile), dessiné par les mêmes composants que la page publique à partir de l'état de
  l'éditeur ; interrupteur « Afficher les brouillons » (activé par défaut, brouillons
  semi-transparents avec un badge).
- **F20** — **Enregistrement explicite** (bouton et `Ctrl/Cmd + S`) du document entier ;
  indicateur de modifications non enregistrées ; avertissement en quittant ; erreurs
  affichées sur les champs concernés ; conflit (page modifiée dans un autre onglet)
  signalé avec une proposition de recharger.
- **F21** — **Images** : le navigateur redessine l'image (orientation EXIF appliquée),
  la réduit à 1024 px et l'envoie ; Go n'accepte que JPEG et PNG vérifiés par décodage,
  refuse au-delà de 2048 × 2048 avant décodage et de `MAX_UPLOAD_MB`, réduit à 512 px,
  ré-encode (JPEG qualité 85, PNG si transparence), ce qui supprime toutes les
  métadonnées. Une image qu'aucune page n'utilise depuis 24 h est supprimée.

### Non fonctionnelles

- **NF1** — **Sécurité** : CSP stricte sur les pages HTML (`default-src 'self'`, ni script
  ni style inline ; `img-src` ajoute `data:` et `blob:` pour le grain) ; CSP propre aux
  SVG de formes ; protection CSRF par `http.CrossOriginProtection` sur l'API d'admin, en
  plus du cookie `SameSite=Strict` ; `X-Robots-Tag: noindex` sur l'admin ; contenu saisi
  affiché comme texte uniquement ; URL validées côté Go (F3).
- **NF2** — **Authentification** : seul le hash bcrypt du mot de passe est fourni
  (`ADMIN_PASSWORD_HASH`) ; session en base (seul le SHA-256 du jeton est stocké),
  cookie `HttpOnly`, `SameSite=Strict`, `__Host-` et `Secure` en HTTPS ; durée 30 jours ;
  changer le mot de passe invalide toutes les sessions ; connexion limitée à
  `RATE_LIMIT_LOGIN` essais par IP et par heure.
- **NF3** — **Accessibilité** : contraste WCAG AA vérifié par test pour chaque
  combinaison thème × accent (texte sur le fond, y compris les couleurs extrêmes des fonds
  animés, et texte sur les cartes) ; navigation au clavier, focus visible, accordéons
  accessibles (Radix), cibles tactiles ≥ 44 px.
- **NF4** — **Poids** : JS initial de la page publique ≤ 150 Kio gzip, chaque fond animé
  ≤ 30 Kio gzip (vérifiés au build, qui échoue en cas de dépassement) ; polices
  auto-hébergées, chargées seulement pour le thème utilisé ; code de l'admin dans un
  bundle séparé ; aucune requête vers un tiers.
- **NF5** — **Ressources** : une réplique (le rate limiting est en mémoire) ; requests
  20m / 48Mi, limits 500m / 192Mi (marge pour le décodage d'images) ;
  `readOnlyRootFilesystem` (les uploads sont lus en flux, jamais écrits sur disque).

### Interfaces

Pages et fichiers :

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `GET /` | Index | 200 |
| `GET /{slug}` | Page de liens | 200 ; 404 (inconnue, brouillon, slug invalide) |
| `GET /images/{id}` | Image uploadée, cache d'un an | 200 ; 404 |
| `GET /assets/…`, `GET /shapes/{id}.svg` | Bundles Vite, formes ; cache d'un an | 200 ; 404 |
| `GET /favicon.svg`, `GET /og-default.png`, `GET /robots.txt` | Statiques (`robots.txt` exclut `/admin`) | 200 |
| `GET /admin`, `GET /admin/{path...}` | Application d'admin | 200 |
| `GET /healthz`, `GET /readyz`, `GET /version` | Santé (readyz : base joignable), SHA de build | 200 ; 503 |

API d'admin (JSON ; toutes les routes sauf `login` exigent une session, sinon 401) :

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `POST /admin/api/login` | `{password}` → cookie de session | 200 ; 401 ; 429 |
| `POST /admin/api/logout` | Ferme la session | 204 |
| `GET /admin/api/session` | État de la session | 200 ; 401 |
| `GET /admin/api/pages` | Résumés des pages, ordre de l'index | 200 |
| `POST /admin/api/pages` | `{title, slug}` → page créée en brouillon | 201 ; 422 |
| `GET /admin/api/pages/{id}` | Document complet (réglages, éléments, `version`) | 200 ; 404 |
| `PUT /admin/api/pages/{id}` | Remplace le document complet, en une transaction | 200 `{version}` ; 409 ; 422 |
| `DELETE /admin/api/pages/{id}` | Supprime la page | 204 ; 404 |
| `PUT /admin/api/pages/order` | `{ids}` → ordre de l'index | 204 ; 422 |
| `GET /admin/api/slugs/{slug}?except={id}` | Validité et disponibilité d'un slug | 200 |
| `POST /admin/api/images` | Upload `multipart/form-data` (champ `file`) | 201 ; 413 ; 422 |

- Les éléments portent un **UUID généré par le navigateur** ; un UUID appartenant à une
  autre page est refusé (422). Le document porte un entier `version` (**verrou
  optimiste**) : une version périmée répond 409.
- Erreurs : `{"error": {"code", "message", "fields"}}`, où `fields` est indexé par champ
  (`slug`, `items.<uuid>.url`…) pour que l'éditeur surligne le bon élément.
- Les messages sont définis une fois en Go et en miroir en TypeScript ; un test de contrat
  (fichier `contract.json`) garde les deux alignés, listes de thèmes et d'accents comprises.

### Données

PostgreSQL, base `linktree` sur l'instance partagée :

- `images` : `id` (uuidv7), `content_type` (JPEG ou PNG), `data` (bytea), `width`,
  `height`, `created_at`.
- `pages` : `id`, `slug` (unique), `title`, `description`, `avatar_image_id` ou
  `avatar_shape` (pas les deux), `theme`, `accent`, `published`, `listed`, `position`
  (ordre de l'index), `version`, dates.
- `items` : `id` (UUID du navigateur), `page_id` (cascade), `parent_id` (le groupe d'un
  lien, cascade), `kind` (`link` / `group`), `position`, `title`, `subtitle`, `url`,
  `image_id` ou `shape` (pas les deux ; pas de forme sur un lien), `open_by_default`,
  `published`. Contraintes : URL présente ssi lien, groupe seulement au premier niveau.
- `sessions` : `token_hash`, `password_fingerprint`, `created_at`, `expires_at`.
- Purge horaire par l'app : images orphelines depuis 24 h, sessions expirées.
- La liste des formes valides vient des fichiers SVG embarqués : une forme retirée fait
  échouer la validation plutôt que d'afficher une image cassée.

### Configuration

| Variable | Rôle | Défaut / dev |
|----------|------|--------------|
| `ADMIN_PASSWORD_HASH` | Hash bcrypt du mot de passe d'admin (obligatoire, secret) ; `make hash-password` | hash de `linktree-dev` |
| `PUBLIC_BASE_URL` | URL publique (Open Graph ; cookies `Secure` si `https://`) | `http://localhost:8080` |
| `TRUST_PROXY` | Lire l'IP dans `X-Forwarded-For` (Traefik) | `false` |
| `RATE_LIMIT_LOGIN` | Essais de connexion par IP et par heure | `10` |
| `MAX_UPLOAD_MB` | Taille maximale d'un upload | `5` |
| `VITE_DEV_URL` | Serveur Vite en développement (vide en production) | posé par `make dev` |
| `DB_PORT` | Port local de la base (Undercover et Pastebin ont 5433 et 5434) | `5435` |

## 5. Critères d'acceptation

- **CA1** (F1, F17) — Étant donné une session admin, quand je crée la page « Musique »
  (`musique`), alors elle apparaît en brouillon avec une forme en avatar, et
  `/musique` répond 404 tant qu'elle n'est pas publiée.
- **CA2** (F2) — Quand je crée une page avec le slug `admin`, `Musique` ou un slug déjà
  pris, alors la création est refusée avec un message sur le champ slug.
- **CA3** (F3) — Quand j'enregistre un lien `javascript:alert(1)`, `data:…` ou
  `java\tscript:…`, alors l'enregistrement est refusé et le champ URL est signalé.
- **CA4** (F6, O3) — Étant donné une page publiée qui contient un lien, un groupe et un
  lien de groupe en brouillon, quand un visiteur l'ouvre, alors aucun de leurs titres
  n'apparaît dans le HTML reçu.
- **CA5** (F4, F19) — Quand j'ajoute un groupe « Midnight City » avec Spotify et Deezer,
  ouvert par défaut, alors l'aperçu puis la page publique le montrent ouvert, avec les
  deux icônes, et plusieurs groupes peuvent être ouverts en même temps.
- **CA6** (F20) — Étant donné la même page ouverte dans deux onglets, quand j'enregistre
  dans le premier puis dans le second, alors le second reçoit un conflit et propose de
  recharger, sans écraser le premier.
- **CA7** (F21) — Quand j'envoie une photo de téléphone avec position GPS, alors l'image
  servie fait au plus 512 px et ne contient plus de métadonnées ; un SVG est refusé.
- **CA8** (F9, F8) — Quand je colle l'URL d'une page dans une messagerie, alors l'aperçu
  montre son titre, sa description et son image ; une page non listée porte `noindex`.
- **CA9** (F14, NF3) — Avec `prefers-reduced-motion`, aucun fond animé ni animation
  d'entrée n'est joué ; le test de contraste passe pour les 5 thèmes × 8 accents.
- **CA10** (NF2) — Après 10 mauvais mots de passe en une heure, la connexion répond 429 ;
  après changement de `ADMIN_PASSWORD_HASH`, les anciennes sessions sont refusées.
- **CA11** (NF1) — Une requête d'écriture de l'API venant d'une autre origine est refusée
  (403) même avec un cookie de session valide.
- **CA12** (NF4) — Le build échoue si le JS initial public dépasse 150 Kio gzip ;
  Lighthouse mobile ≥ 90 en performance et 100 en accessibilité sur le thème le plus
  lourd.

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Fonds WebGL lourds sur un téléphone ancien | Moyen | Moyenne | Chargement différé, pause hors écran, DPR plafonné, repli CSS (F14) ; budget vérifié au build |
| Texte illisible sur un fond animé | Moyen | Faible | Opacité des fonds fixée dans les tokens et couverte par le test de contraste (NF3) |
| Licence React Bits (MIT + Commons Clause) | Faible | Faible | Usage dans l'app permis ; composants jamais publiés seuls ; à revérifier si le dépôt devient public |
| Mot de passe d'admin deviné ou volé | Élevé | Faible | bcrypt, rate limiting, sessions invalidées au changement de mot de passe (NF2) |
| Images en base qui grossissent | Faible | Faible | 512 px max, purge des orphelines, sauvegardes `pg_dump` existantes |
| Port de base locale en conflit avec une autre app | Faible | Certaine sans parade | Port 5435 dédié |

## 7. Décisions ouvertes

Décisions prises avec Noah le 2026-10-14 (non rediscutées) : application dynamique à
plusieurs pages et index `/` ; groupes en accordéon ; admin à mot de passe unique ; images
uploadées et stockées en base ; React + TypeScript + Tailwind pour les pages publiques et
l'admin, embarqués dans le binaire Go (écart assumé avec le rendu serveur par défaut du
`CLAUDE.md` racine, voir F9) ; thèmes prédéfinis + accent ; brouillons et icônes de
plateformes dans le MVP ; direction artistique coolshapes + React Bits.

- [ ] **D1** — **Réglages fins des thèmes** (polices, intensité des fonds, décor).
  **Proposition :** valider sur les pages de démo pendant T5 ; les polices actuelles sont
  Geist, Space Grotesk, Fraunces, Unbounded, Bricolage Grotesque. (bloque : T5)
- [ ] **D2** — **Contenu initial** (pages `musique`, `jeux`…). **Proposition :** saisi
  par Noah dans l'admin après le déploiement. (bloque : rien ; la recette utilise une page
  de démo)
- [ ] **D3** — **Identité visuelle partagée avec l'accueil** (`apps/home/`).
  **Proposition :** non dans le MVP ; à reconsidérer quand l'accueil sera cadré.
  (bloque : rien)
- [ ] **D4** — **Image Open Graph par défaut**. **Proposition :** celle de
  `feat/linktree-mvp` (fond sombre, quatre formes, « link.zazi.cool »). (bloque : rien)

## 8. Découpage en tickets

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | **Serveur et API** : migrations, domaine et validation, stockage, authentification, API d'admin, images, purge, `/healthz` `/readyz` `/version`, en-têtes de sécurité, `Dockerfile`, `compose.yaml`, `Makefile`, `.env.example`, `README.md` ; le job CI `linktree` s'exécute — *écrit sur `feat/linktree-mvp`* | F1–F6, F16, F21, NF1, NF2, NF5 ; CA1–CA4, CA6, CA7, CA10, CA11 | — | dev |
| T2 | 1 | **Infra** : base et mot de passe dans `infra/data` (infra partagée, avec accord de Noah), manifests `infra/apps/linktree/{base,overlays/prod}` (Deployment, Service, Ingress `link.zazi.cool`, Middlewares `redirect-https` et `security-headers`, secret SOPS `DATABASE_URL` + `ADMIN_PASSWORD_HASH`), Kustomization Flux dans `infra/flux/clusters/prod/apps.yaml` | NF5 | — | infra |
| T3 | 2 | **Pages publiques** : rendu React, thèmes et tokens, formes, icônes, fonds animés, index, 404, partage, budget du bundle — *écrit sur `feat/linktree-mvp`* | F7–F15, NF3, NF4 ; CA5 (public), CA8, CA9, CA12 (build) | T1 | dev |
| T4 | 2 | **Administration** : connexion, liste et ordre des pages, éditeur (reducer testé, glisser-déposer, panneaux, sélecteurs de formes et de thèmes, upload côté client), aperçu en direct, conflits | F16–F21 ; CA1, CA5, CA6 | T1, T3 | dev |
| T5 | 3 | **Finitions** : réglage des thèmes (D1), passe Lighthouse et accessibilité sur la build de production, test manuel sur téléphone | O2, NF3, NF4 ; CA9, CA12 | T3, T4 | dev |
| T6 | 3 | **Documentation** : règles durables dans `apps/linktree/CLAUDE.md`, statut du tableau du `CLAUDE.md` racine | — | T1–T5 | dev (doc) |
| T7 | 4 | **Déployer linktree `<sha7>`** : `newTag` dans `infra/apps/linktree/overlays/prod/`, PR de bump, vérification `deploy-verify`, recette §9 | — | T1–T6 | déploiement |

T1 et T3 sont déjà écrits et testés (`make lint` et `make test` au vert) : ils passeront en
PR, une par ticket, après validation de cette spec. Un agent ne touche jamais au cluster.

## 9. Recette après déploiement

- [ ] `curl -sI https://link.zazi.cool/healthz` → `200` ; `/readyz` → `200` ;
      `curl -s https://link.zazi.cool/version` → SHA déployé.
- [ ] `curl -sI http://link.zazi.cool/` → redirection vers `https://` ; certificat valide.
- [ ] `/admin` : connexion, création d'une page de démo avec un lien, un groupe et une
      image, publication ; la page s'affiche sur un téléphone, dans chacun des 5 thèmes.
- [ ] Un brouillon de la page de démo n'apparaît pas dans le HTML public (CA4).
- [ ] Coller l'URL dans une messagerie : titre, description et image (CA8).
- [ ] Lighthouse mobile sur la production : performance ≥ 90, accessibilité 100.
- [ ] Ajouter `/healthz` au service d'uptime quand il sera choisi ; supprimer la page de
      démo ; mettre le tableau du `CLAUDE.md` racine à « Production ».
