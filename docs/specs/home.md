# Spec — Accueil (`zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **En revue** — rédigée le 2026-10-26, décisions D1 à D6 à trancher |
| Issue Multica  | ONBO-64 |
| Application    | `apps/home/` |
| Auteur, valideur | Rédacteur de specs / Noah |

## 1. Contexte

L'apex `zazi.cool` n'affiche encore rien : `apps/home/` ne contient que son `CLAUDE.md`,
qui prévoyait une page **statique** listant les services. Trois services sont en
production (Undercover, Pastebin, Linktree), chacun sur son sous-domaine, et rien ne
permet de les découvrir depuis le domaine principal.

La demande (ONBO-64) précise le besoin :

- l'accueil est un **hub** qui renvoie vers tous les services de `zazi.cool` ;
- il reprend le **style soigné du linktree** (`link.zazi.cool`) ;
- il porte une **bannière image, un titre et un paragraphe** que Noah modifie dans une
  **mini administration**.

Ce dernier point change l'architecture prévue : un contenu modifiable sans redéploiement
doit être stocké quelque part. L'accueil devient donc une **petite application dynamique**
(base PostgreSQL `home`, admin protégée par mot de passe), sur le modèle du linktree mais
bien plus réduite : un seul document, trois champs. Le `CLAUDE.md` de l'app sera mis à
jour en conséquence (T1).

`www.zazi.cool` redirige en 301 vers `zazi.cool` (décision du 2026-10-14, inchangée).

## 2. Objectifs

- **O1** — Depuis `zazi.cool`, un visiteur voit en un écran (mobile compris) ce qu'est le
  site et ouvre n'importe quel service en un geste.
- **O2** — Noah change la bannière, le titre ou le paragraphe depuis son téléphone en
  moins d'une minute, sans PR ni redéploiement.
- **O3** — La page a le même niveau de finition visuelle que le linktree : même famille
  de typographie, de couleurs et de cartes, mode clair et sombre.
- **O4** — Aucun contenu saisi dans l'admin ne peut exécuter de script chez un visiteur,
  et l'admin n'est accessible qu'avec le mot de passe.
- **O5** — La page reste légère : affichée sans JavaScript, Lighthouse mobile ≥ 95 en
  performance et en accessibilité.

## 3. Périmètre

### Inclus (cette version)
- Page publique `/` : bannière, titre, paragraphe, grille de cartes vers les services.
- Liste des services définie dans le code (D3).
- Mini admin `/admin` : connexion par mot de passe, formulaire unique (titre, paragraphe,
  bannière avec texte alternatif, suppression de la bannière), déconnexion.
- Bannière téléversée (JPEG ou PNG), redimensionnée, nettoyée de ses métadonnées,
  stockée en base.
- Métadonnées de partage (Open Graph), favicon, `robots.txt`, page 404.
- Redirection `www.zazi.cool` → `zazi.cool` (301) et HTTP → HTTPS, par Traefik.
- Endpoints `/healthz`, `/readyz`, `/version`.

### Hors périmètre
- **Liste des services éditable dans l'admin** : elle ne change qu'à l'arrivée d'une
  nouvelle app, qui passe de toute façon par une PR (D3).
- **Statut en direct des services** (pastille « en ligne ») : demanderait des sondes
  côté serveur ; l'uptime externe du `CLAUDE.md` racine s'en charge.
- Brouillon et publication différée : un enregistrement est publié immédiatement.
- Mise en forme du paragraphe (gras, liens, Markdown) : texte brut avec sauts de ligne.
- Plusieurs bannières, carrousel, point focal de recadrage.
- Fond animé WebGL comme sur certains thèmes du linktree (D2).
- Plusieurs administrateurs, réinitialisation du mot de passe par l'interface.
- Statistiques de visite, cookies côté visiteur.

## 4. Exigences

### Fonctionnelles

**Page publique**

- **F1** — `GET /` affiche, dans cet ordre : la bannière (si elle existe), le titre, le
  paragraphe, puis une carte par service.
- **F2** — Sans bannière enregistrée, la zone de bannière est remplacée par le dégradé du
  thème, à hauteur réduite : la page reste équilibrée.
- **F3** — La bannière s'affiche dans un cadre de ratio fixe (D6), recadrée au centre
  (`object-fit: cover`), avec son texte alternatif (`alt=""` s'il est vide : image
  décorative). Le HTML porte ses dimensions pour éviter tout décalage à l'affichage.
- **F4** — Le paragraphe est du texte brut : chaque ligne vide sépare deux paragraphes
  HTML, un saut de ligne simple devient `<br>`. Aucun autre balisage n'est interprété.
- **F5** — Chaque carte de service affiche son icône, son nom, une phrase de description
  et son domaine (ex. `pb.zazi.cool`) ; toute la carte est cliquable et ouvre le service
  dans le même onglet. Services au lancement (D3) :

  | Service | URL | Description proposée |
  |---------|-----|----------------------|
  | Undercover | `https://undercover.zazi.cool` | Le jeu d'ambiance à jouer entre amis sur un seul téléphone. |
  | Pastebin | `https://pb.zazi.cool` | Partager un texte ou du code avec un lien qui expire. |
  | Liens | `https://link.zazi.cool` | Mes pages de liens : musique, jeux… |

- **F6** — Les métadonnées sont rendues par le serveur : `<title>` = titre ; `og:title`,
  `og:description` (paragraphe tronqué à 200 caractères), `og:url` (`PUBLIC_BASE_URL`),
  `og:image` (la bannière si elle existe, sinon `/og-default.png`), `og:type=website`.
- **F7** — Tout autre chemin public renvoie **404** avec une page « Page introuvable » au
  style de l'accueil et un lien de retour.
- **F8** — Au premier démarrage, la base contient un contenu par défaut (migration) :
  titre « zazi.cool », un paragraphe de présentation court, pas de bannière.

**Administration**

- **F9** — `GET /admin` sans session affiche le formulaire de connexion (mot de passe
  seul) ; avec session, le formulaire d'édition prérempli. Toutes les pages d'admin sont
  `noindex` et `Cache-Control: no-store`.
- **F10** — Le formulaire d'édition contient : titre, paragraphe, bannière actuelle
  (aperçu) avec son texte alternatif, champ de fichier pour une nouvelle bannière, case
  « Supprimer la bannière », bouton « Enregistrer » et lien « Voir la page ». Il
  fonctionne **sans JavaScript** (formulaire HTML classique, `multipart/form-data`).
- **F11** — `POST /admin` enregistre le tout en une transaction puis redirige (`303`)
  vers `/admin` avec un message « Modifications enregistrées ». Le contenu est visible
  sur `/` immédiatement.
- **F12** — **Versionnage optimiste** : le formulaire porte la `version` lue ; si le
  contenu a été enregistré entre-temps (autre appareil), l'enregistrement est refusé
  (**409**), le formulaire est réaffiché avec la saisie et le message « Le contenu a été
  modifié ailleurs, rechargez la page ».
- **F13** — Validation côté serveur, messages en français par champ, toutes les erreurs
  d'un coup (**422**, formulaire réaffiché avec la saisie, rien n'est enregistré) :
  - titre : 1 à 80 caractères ;
  - paragraphe : 0 à 600 caractères, sauts de ligne permis ;
  - texte alternatif : 0 à 150 caractères ;
  - pour les trois : UTF-8 valide, sans caractère de contrôle (hors sauts de ligne du
    paragraphe), espaces de bord retirés ; longueurs comptées en caractères.
  - envoyer une nouvelle bannière **et** cocher « Supprimer » est refusé.

**Bannière**

- **F14** — Le fichier envoyé doit être du **JPEG ou du PNG** (format vérifié sur le
  contenu, pas sur l'extension), de `MAX_UPLOAD_MB` Mo au plus (5 par défaut) et de
  **4000 × 4000 pixels** au plus (dimensions lues avant décodage complet). Le serveur le
  **redimensionne à 1600 px de large** au plus et le **ré-encode** (JPEG qualité 85 si
  l'image est opaque, PNG sinon), ce qui supprime toutes les métadonnées (EXIF, position
  GPS). Taille dépassée : 413 ; format ou dimensions refusés : 422.
- **F15** — `GET /images/{id}` sert l'image avec son type et
  `Cache-Control: public, max-age=31536000, immutable` ; un identifiant inconnu ou
  invalide répond 404. Une nouvelle bannière crée un nouvel identifiant.
- **F16** — Les images qui ne sont plus la bannière courante, créées depuis plus de
  24 h, sont supprimées par une tâche interne toutes les heures (idempotente).

**Authentification**

- **F17** — Un **mot de passe unique** protège l'admin ; l'application ne connaît que
  son hash bcrypt (`ADMIN_PASSWORD_HASH`, D5). `make hash-password` en génère un
  (12 caractères au moins, 72 octets au plus).
- **F18** — `POST /admin/login` ouvre une session de **30 jours** (jeton aléatoire de
  256 bits, seule l'empreinte SHA-256 est en base) et pose un cookie `HttpOnly`,
  `SameSite=Strict`, et en HTTPS `Secure` avec le préfixe `__Host-` ; mauvais mot de
  passe : 401 et formulaire avec « Mot de passe incorrect ». `POST /admin/logout` ferme
  la session. Les sessions expirées sont purgées toutes les heures.
- **F19** — Changer `ADMIN_PASSWORD_HASH` invalide toutes les sessions existantes
  (chaque session mémorise l'empreinte du hash qui l'a créée).
- **F20** — Les tentatives de connexion sont limitées **par IP** (`RATE_LIMIT_LOGIN`
  par heure, 10 par défaut, en mémoire) : au-delà, 429, même avec le bon mot de passe.

**Exploitation**

- **F21** — `GET /version` renvoie le SHA git du build ; `/readyz` répond 200 si la base
  est joignable, 503 sinon.

### Non fonctionnelles

- **NF1** — **Style** (D2) : reprise du thème `zazi` du linktree — police Geist
  (variable, auto-hébergée), dégradé de fond et grain, cartes translucides à bord fin,
  rayon et ombres identiques, accent violet ; **clair ou sombre selon le système**
  (`prefers-color-scheme`). Les couleurs viennent des jetons du linktree
  (`apps/linktree/web/src/themes/tokens.ts`), recopiés en variables CSS. Survol et focus
  des cartes animés en CSS seulement, désactivés si `prefers-reduced-motion: reduce`.
- **NF2** — **Mise en page** : une colonne centrée de 720 px au plus ; cartes en grille
  d'une colonne sous 640 px, deux au-delà ; rien ne déborde à 320 px de large.
- **NF3** — **Poids** : page publique **sans JavaScript** (D1) ; HTML + CSS + police
  ≤ **80 Kio** transférés, bannière exclue. Bannière en `fetchpriority="high"`.
  Lighthouse mobile ≥ 95 en performance, accessibilité et bonnes pratiques.
- **NF4** — **Accessibilité** : contraste AA (4,5:1) du texte sur le fond et sur les
  cartes, dans les deux modes (le texte n'est jamais posé sur la bannière) ; focus visible ; titres
  hiérarchisés (`h1` = titre) ; la grille de services est une liste.
- **NF5** — **Sécurité** : rendu uniquement par `html/template` ; CSP
  `default-src 'none'; style-src 'self'; img-src 'self'; font-src 'self';
  form-action 'self'; base-uri 'none'; frame-ancestors 'none'` (aucun script) ;
  `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`.
  Aucune ressource d'un tiers.
- **NF6** — **CSRF** : les `POST` de l'admin sont refusés (403) si `Sec-Fetch-Site`
  vaut `cross-site` ou si `Origin` ne correspond pas à `PUBLIC_BASE_URL`.
- **NF7** — **Vie privée** : aucun cookie ni pisteur côté visiteur ; seul l'admin
  dépose un cookie de session ; l'IP n'est gardée qu'en mémoire (limitation).
- **NF8** — **Ressources** : un seul réplica (limitation en mémoire) ; requests `10m`
  CPU / `32Mi`, limits `300m` / `192Mi` (le décodage d'une image de 4000 px demande
  ~64 Mio). Corps de requête bornés : 16 Kio hors admin, `MAX_UPLOAD_MB` + 64 Kio pour
  `POST /admin`.
- **NF9** — Le contenu de la page est lu en base à chaque requête (une ligne) : pas de
  cache applicatif, pour que F11 reste vrai sans invalidation.

### Interfaces

Pages en **rendu serveur** (`html/template`), sans front TypeScript (D1).

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `GET /` | Page d'accueil | 200 |
| `GET /images/{id}` | Bannière | 200 ; 404 |
| `GET /assets/…`, `/favicon.svg`, `/og-default.png`, `/robots.txt` | Fichiers statiques embarqués (assets : cache d'un an, noms avec empreinte) | 200 ; 404 |
| `GET /admin` | Connexion ou formulaire d'édition | 200 |
| `POST /admin/login` | Formulaire `password` → session, 303 vers `/admin` | 303 ; 401 ; 403 ; 429 |
| `POST /admin/logout` | Ferme la session, 303 vers `/admin` | 303 ; 403 |
| `POST /admin` | Formulaire `title`, `body`, `banner_alt`, `banner` (fichier), `remove_banner`, `version` | 303 ; 401 ; 403 ; 409 ; 413 ; 422 |
| `GET /healthz`, `/readyz`, `/version` | Santé et version | 200 ; 503 pour `/readyz` |
| tout autre chemin | Page 404 | 404 |

`robots.txt` autorise tout sauf `/admin`. Côté Traefik (T2) : `www.zazi.cool/*` → 301
`https://zazi.cool/*` (middleware `redirectRegex`), HTTP → HTTPS, HSTS.

### Données

PostgreSQL, base `home` (migration `001_init.sql`) :

- **`content`** — une seule ligne (`id smallint primary key check (id = 1)`) : `title`,
  `body`, `banner_image_id` (nullable, clé étrangère vers `images`), `banner_alt`,
  `version` (entier, incrémenté à chaque enregistrement), `updated_at`. La migration
  insère le contenu par défaut (F8).
- **`images`** — `id` (UUID), `content_type` (JPEG ou PNG), `data` (`bytea`, déjà
  redimensionnée), `width`, `height`, `created_at`. Une bannière de 1600 px pèse
  ~200 à 400 Kio.
- **`sessions`** — `token_hash` (SHA-256), `password_fingerprint`, `created_at`,
  `expires_at`.

Rétention : images orphelines de plus de 24 h et sessions expirées, purgées toutes les
heures. Tout est inclus dans le `pg_dump` quotidien.

### Configuration

À reporter dans `apps/home/.env.example` :

| Variable | Rôle | Défaut |
|----------|------|--------|
| `PORT` | Port d'écoute | `8080` |
| `DATABASE_URL` | Connexion PostgreSQL (obligatoire) | — |
| `ADMIN_PASSWORD_HASH` | Hash bcrypt du mot de passe de l'admin (obligatoire) | — |
| `PUBLIC_BASE_URL` | URL publique : Open Graph, vérification `Origin`, cookies `Secure` si `https://` | `http://localhost:8080` |
| `TRUST_PROXY` | Lire l'IP client dans `X-Forwarded-For` (derrière Traefik seulement) | `false` |
| `RATE_LIMIT_LOGIN` | Tentatives de connexion par IP et par heure | `10` |
| `MAX_UPLOAD_MB` | Taille maximale d'une bannière envoyée | `5` |

La liste des services, les limites de longueur et la largeur de la bannière sont des
constantes du code : elles font partie de la spec.

**Dépendances** : `pgx` (PostgreSQL), `golang.org/x/crypto` (bcrypt) et
`golang.org/x/image` (redimensionnement), déjà approuvées pour le linktree. Le code
d'authentification, de limitation et de traitement d'image du linktree est **recopié et
simplifié** dans `apps/home/internal/` (chaque app a son module Go ; pas de module
partagé, voir §6).

## 5. Critères d'acceptation

- **CA1** (F1, F5) — Étant donné le contenu par défaut, quand je charge `/`, alors je
  vois le titre, le paragraphe et trois cartes (Undercover, Pastebin, Liens) dont les
  liens pointent vers leurs sous-domaines en `https://`.
- **CA2** (F2, F3) — Sans bannière, la page affiche le dégradé à la place et aucune
  balise `<img>` de bannière ; avec une bannière et un texte alternatif, l'`<img>` porte
  `alt`, `width` et `height`.
- **CA3** (F4, NF5) — Étant donné un paragraphe `<script>alert(1)</script>` suivi d'une
  ligne vide et de « deux », quand je charge `/`, alors le script s'affiche comme du
  texte, il y a deux `<p>`, et la réponse porte la CSP de NF5 (test HTTP).
- **CA4** (F6) — Le HTML de `/` contient `og:title`, `og:description` (≤ 200 caractères),
  `og:url` et `og:image` ; sans bannière, `og:image` vaut `/og-default.png` en URL
  absolue.
- **CA5** (F7) — `GET /inconnu` répond 404 avec la page « Page introuvable ».
- **CA6** (F9, F18) — Sans session, `GET /admin` affiche la connexion ; un mauvais mot de
  passe donne 401 sans cookie ; le bon pose un cookie `HttpOnly`, `SameSite=Strict`
  (et `Secure` + `__Host-` en HTTPS) et `GET /admin` affiche le formulaire prérempli.
- **CA7** (F11) — Connecté, quand j'enregistre un nouveau titre, alors je suis redirigé
  vers `/admin` avec « Modifications enregistrées » et `/` affiche le nouveau titre.
- **CA8** (F12) — Étant donné le formulaire ouvert sur deux appareils, quand j'enregistre
  sur le premier puis sur le second, alors le second reçoit 409 avec sa saisie conservée
  et le contenu du premier reste en base.
- **CA9** (F13) — Un titre vide ou de 81 caractères, un paragraphe de 601 caractères, un
  texte contenant un caractère de contrôle, ou « nouvelle bannière + Supprimer », donne
  422 avec un message par champ en erreur, et rien n'est enregistré.
- **CA10** (F14) — Un GIF, un SVG ou un fichier texte renommé en `.jpg` : 422 ; plus de
  `MAX_UPLOAD_MB` : 413 ; 4001 px de côté : 422. Un JPEG de 3000 × 1000 avec EXIF est
  servi en 1600 × 533, sans EXIF.
- **CA11** (F15) — `GET /images/{id}` de la bannière porte
  `Cache-Control: public, max-age=31536000, immutable` ; un identifiant inconnu ou
  malformé répond 404.
- **CA12** (F16) — Étant donné une ancienne bannière remplacée il y a plus de 24 h, quand
  la purge s'exécute, alors elle est supprimée ; la bannière courante et une image de
  moins de 24 h sont conservées (test d'intégration).
- **CA13** (F10) — Avec JavaScript désactivé, je me connecte, change le paragraphe et
  envoie une bannière avec succès.
- **CA14** (F19) — Étant donné une session ouverte, quand `ADMIN_PASSWORD_HASH` change
  et que l'application redémarre, alors cette session est refusée (401 sur `POST /admin`,
  connexion affichée sur `GET /admin`).
- **CA15** (F20) — La 11ᵉ tentative de connexion depuis une même IP dans l'heure
  reçoit 429, y compris avec le bon mot de passe.
- **CA16** (NF6) — Un `POST /admin` avec `Sec-Fetch-Site: cross-site` ou un `Origin`
  étranger répond 403 et ne modifie rien.
- **CA17** (NF1, NF4) — Un test vérifie le contraste AA des couleurs de texte sur le fond
  et sur les cartes, en clair et en sombre.
- **CA18** (NF2, NF3) — À 320 px de large, aucun défilement horizontal ; la page publique
  ne charge aucun script ; Lighthouse mobile ≥ 95 en performance, accessibilité et
  bonnes pratiques (noté à la recette).
- **CA19** (F21) — `/readyz` répond 200 avec la base et 503 sans ; `/version` renvoie le
  SHA du build.
- **CA20** (production) — `https://www.zazi.cool/x` répond 301 vers `https://zazi.cool/x` ;
  `http://zazi.cool/` redirige vers HTTPS ; certificat valide pour les deux noms.

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Mot de passe d'admin deviné ou fuité | Défiguration de la page d'accueil | Faible | bcrypt, limitation par IP, cookie strict, mot de passe de 12 caractères au moins ; contenu restaurable depuis `pg_dump` |
| Code d'auth et d'image recopié du linktree : deux copies à maintenir | Correctif de sécurité appliqué à une seule app | Moyenne | Mentionner la copie dans les deux `CLAUDE.md` ; envisager un module partagé seulement si une troisième app en a besoin |
| Bannière lourde ou mal cadrée sur mobile | Page lente, sujet coupé | Moyenne | Redimensionnement à 1600 px, ratio fixe (D6), conseil de dimensions affiché dans l'admin |
| Image de 4000 px décodée en mémoire | Pod tué (OOM) | Faible | Dimensions lues avant décodage, limite mémoire 192 Mio (NF8), une seule bannière à la fois |
| Liste des services en dur oubliée à l'arrivée d'une app | Service absent du hub | Moyenne | Ajouter « carte sur l'accueil » à la recette des futures specs d'app |
| Le premier `host` de l'Ingress doit être `zazi.cool` | `deploy-verify` vérifierait `www` (qui répond 301) | Faible | Ordre imposé dans T2 et vérifié en review |
| HSTS sur l'apex avec `includeSubDomains` | Un futur sous-domaine en HTTP seul deviendrait inaccessible | Faible | Tous les sous-domaines sont déjà en HTTPS ; le middleware des autres apps l'applique déjà |
| Écart avec le `CLAUDE.md` actuel (« statique, sans base ») | Agents qui suivent l'ancienne consigne | Moyenne | Mise à jour du `CLAUDE.md` de l'app et du tableau racine dans T1 |

## 7. Décisions ouvertes

- [ ] **D1** — **Technique du front** : rendu serveur (`html/template` + CSS écrit à la
  main, zéro JavaScript), ou React + Vite comme le linktree ? **Proposition :** rendu
  serveur. Une page et un formulaire n'ont pas besoin de React ; on évite Node dans la
  chaîne, et la page reste très légère. Le style du linktree se reproduit en CSS à partir
  de ses jetons. (bloque : T1, T3, T5)
- [ ] **D2** — **Identité visuelle** : reprendre le thème `zazi` du linktree (Geist,
  dégradé et grain, clair/sombre selon le système, accent violet), sans fond animé ?
  **Proposition :** oui. Un fond animé demanderait du JavaScript et WebGL pour un gain
  faible ; il pourra faire l'objet d'une évolution. (bloque : T3)
- [ ] **D3** — **Liste des services** : en dur dans le code (nom, URL, description,
  icône), avec les trois services et les descriptions de F5 ? **Proposition :** oui ;
  libellé « Liens » pour le linktree, plus parlant pour un visiteur. Noah valide ou
  corrige les noms et descriptions. (bloque : T3)
- [ ] **D4** — **Base `home`** dans `infra/data` : l'agent prépare la PR (ligne
  `create_app_db home`, entrée du mot de passe dans le secret SOPS à chiffrer par Noah
  avec `sops edit`), Noah l'applique, comme pour le pastebin (D5 de sa spec).
  **Proposition :** oui. (bloque : T2, T7)
- [ ] **D5** — **Mot de passe de l'admin** : propre à l'accueil (`ADMIN_PASSWORD_HASH`
  dans le secret de `home`), sessions séparées de celles du linktree ? **Proposition :**
  oui ; rien n'empêche Noah de choisir le même mot de passe. Une connexion unique entre
  apps demanderait un cookie partagé sur `.zazi.cool`, hors de proportion. (bloque : T4)
- [ ] **D6** — **Format de la bannière** : cadre de ratio **3:1** au-dessus de 640 px de
  large et **2:1** en dessous, recadrage centré, image stockée à 1600 px de large au
  plus ; l'admin conseille « 1600 × 533 px, sujet au centre ». **Proposition :** oui.
  (bloque : T6)

## 8. Découpage en tickets

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | Squelette de l'app : `cmd/home`, `/healthz` `/readyz` `/version`, arrêt propre, exécuteur de migrations et `001_init.sql` (tables et contenu par défaut), `Makefile` (cibles standard + `hash-password`), `compose.yaml` (avec PostgreSQL), `Dockerfile`, `.env.example`, `README.md` ; mise à jour de `apps/home/CLAUDE.md` (app dynamique, renvoi vers cette spec) et du tableau du `CLAUDE.md` racine. La CI se déclenche dès que le `Dockerfile` existe (`ci.yml` déjà câblé). | F8, F21, NF8 ; CA19 | — | dev + infra |
| T2 | 1 | Manifests `infra/apps/home/{base,overlays/prod}` : namespace, Deployment, Service, Ingress `zazi.cool` (premier host) et `www.zazi.cool`, middlewares `redirect-https`, `security-headers` et `www` → apex, secret SOPS (`DATABASE_URL`, `ADMIN_PASSWORD_HASH`, valeurs chiffrées par Noah) ; Kustomization Flux `home` dans `infra/flux/clusters/prod/apps.yaml` ; base `home` dans `infra/data` (D4) | NF8, D4, D5 ; CA20 | — | infra |
| T3 | 2 | Page publique : gabarit, thème (CSS issu des jetons `zazi`, police Geist embarquée, clair/sombre), bannière ou dégradé, titre, paragraphe, cartes des services, Open Graph, favicon, `og-default.png`, `robots.txt`, 404, en-têtes de sécurité | F1–F7, NF1–NF5, NF7, NF9 ; CA1–CA5, CA17, CA18 | T1 | dev |
| T4 | 2 | Authentification de l'admin : connexion, sessions en base, déconnexion, invalidation au changement de hash, limitation par IP, protection CSRF, purge des sessions | F9 (connexion), F17–F20, NF6 ; CA6, CA14–CA16 | T1 | dev |
| T5 | 3 | Édition du texte : formulaire (titre, paragraphe, texte alternatif), validation, versionnage optimiste, message de confirmation | F9–F13 ; CA7–CA9, CA13 (sans bannière) | T3, T4 | dev |
| T6 | 3 | Bannière : envoi, vérification, redimensionnement et ré-encodage, `/images/{id}`, suppression, purge des orphelines, conseil de dimensions dans l'admin | F14–F16, D6 ; CA2, CA10–CA13 | T5 (même étape, après T5) | dev |
| T7 | 4 | **Déployer home `<sha7>`** : premier déploiement (base dans `infra/data` appliquée par Noah, secret, `newTag`, réconciliation Flux), recette §9, ajout au service d'uptime quand il sera choisi | CA18, CA20 | T1–T6 | déploiement |

T3 et T4 avancent en parallèle sur des branches distinctes. Un premier déploiement peut
se faire dès la fin de l'étape 2 (page publique avec le contenu par défaut), l'étape 3
suivant avec son propre ticket de déploiement : à décider par Noah au moment de
promouvoir l'étape 3.

## 9. Recette après déploiement

- [ ] `curl -sI https://zazi.cool/healthz` → `200` ; `/readyz` → `200` ;
      `curl -s https://zazi.cool/version` → SHA déployé (vérifié par `deploy-verify`).
- [ ] `curl -sI http://zazi.cool/` → redirection vers `https://zazi.cool/`.
- [ ] `curl -sI https://www.zazi.cool/test` → `301`, `Location: https://zazi.cool/test`.
- [ ] Certificat Let's Encrypt valide pour `zazi.cool` et `www.zazi.cool`.
- [ ] Sur un téléphone, en clair puis en sombre : bannière, titre, paragraphe, trois
      cartes ; chaque carte ouvre le bon service.
- [ ] Admin depuis un téléphone : connexion, changement du titre et du paragraphe,
      envoi d'une bannière (photo avec EXIF) ; `/` à jour aussitôt ; l'image servie par
      `/images/{id}` ne contient plus d'EXIF.
- [ ] Mauvais mot de passe → message ; 11 essais d'affilée → 429.
- [ ] Cookie `__Host-session` : `HttpOnly`, `Secure`, `SameSite=Strict`.
- [ ] En-têtes : `Content-Security-Policy`, `Strict-Transport-Security`,
      `X-Content-Type-Options`, `Referrer-Policy`.
- [ ] Aperçu de partage (Open Graph) correct en collant `https://zazi.cool` dans une
      messagerie.
- [ ] Lighthouse mobile sur `/` : scores notés dans le ticket (≥ 95 attendus).
- [ ] Sauvegarde du lendemain : base `home` présente dans le `pg_dump`.
- [ ] Mettre à jour le tableau des projets du `CLAUDE.md` racine (statut « Production »).
