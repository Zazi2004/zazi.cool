# Spec — Linktree (`link.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **En revue** — spec rédigée a posteriori : elle décrit l'application livrée et déjà en production, pas une évolution à construire |
| Issue Multica  | ONBO-47 (audit de l'existant : ONBO-46) |
| Application    | `apps/linktree/` |
| Auteur, valideur | Rédacteur de specs / Noah |

> **Pourquoi a posteriori ?** L'application a été développée avant que le workflow
> `docs/workflow/` n'existe. La première version de cette spec (ONBO-43, PR #62, fermée)
> décrivait un site statique alimenté par un fichier JSON : ce n'est pas ce qui a été
> construit. Cette version la remplace et s'appuie sur le code (`apps/linktree/`) et sur
> les manifests (`infra/apps/linktree/`).

## 1. Contexte

Service personnel de pages de liens, à la manière de Linktree, mais avec **plusieurs pages
thématiques** (`/musique`, `/jeux`…) au lieu d'une seule. Chaque page a un avatar, un
titre, une description, des liens, des groupes de liens en accordéon, un thème visuel
(avec fond animé) et une couleur d'accent. Tout se gère dans une **administration en
ligne** (`/admin`) protégée par un mot de passe, sans redéploiement.

L'application est **dynamique** : serveur Go, base PostgreSQL (`linktree`, sur l'instance
partagée), front React + TypeScript (Vite) embarqué dans le binaire. Elle est déployée
sur `link.zazi.cool` par Flux (`infra/apps/linktree/overlays/prod`). Le choix « statique ou
dynamique » du premier cadrage est donc tranché : **dynamique**.

## 2. Objectifs

- **O1** — Noah crée, modifie, ordonne, publie et supprime des pages de liens depuis son
  téléphone ou son ordinateur, sans toucher au code ni redéployer.
- **O2** — Un visiteur ouvre une page sur mobile et accède à un lien en un geste ; la page
  reste lisible sans JavaScript.
- **O3** — Rien de ce qui est en brouillon ne sort du serveur, et aucun contenu saisi dans
  l'admin ne peut exécuter de script chez un visiteur.
- **O4** — L'admin n'est accessible qu'avec le mot de passe, et le service reste léger sur
  le VPS partagé (un pod, 192 Mio de mémoire au plus).

## 3. Périmètre

### Inclus (livré)
- Index public des pages (`/`), pages publiques (`/{slug}`), page 404.
- Liens (titre, URL, image ou icône de plateforme) et groupes en accordéon (un seul
  niveau).
- Thèmes, couleurs d'accent, fonds animés, avatars (image téléversée ou forme).
- Administration : connexion, liste et ordre des pages, éditeur avec aperçu, upload
  d'images, publication, brouillons.
- Authentification par mot de passe unique, sessions en base, limitation des tentatives.
- Endpoints `/healthz`, `/readyz`, `/version`.

### Hors périmètre (décidé : pas maintenant)
- Statistiques de clics (piste ultérieure, voir D1).
- Liens programmés ou masqués à date (piste ultérieure, voir D2).
- Plusieurs administrateurs, comptes, rôles : un seul mot de passe.
- Groupes imbriqués (un groupe ne contient que des liens).
- Formats d'image autres que JPEG et PNG (pas de SVG, GIF, WebP).
- Domaines personnalisés par page, thèmes créés depuis l'admin.
- Réinitialisation du mot de passe par l'interface : on change `ADMIN_PASSWORD_HASH`.

## 4. Exigences

### Fonctionnelles

**Pages publiques**

- **F1** — `GET /` liste les pages **publiées et listées** (avatar, titre, description),
  dans l'ordre défini par l'admin.
- **F2** — `GET /{slug}` affiche la page publiée portant ce slug, avec son avatar, son
  titre, sa description, ses éléments dans l'ordre de l'admin, son thème et son accent.
  Un slug inconnu, invalide ou non publié répond **404** avec la page « Page introuvable ».
- **F3** — Une page publiée mais **non listée** est accessible par son URL, absente de
  l'index, et marquée `noindex`.
- **F4** — Les éléments en **brouillon** (lien ou groupe non publié) sont retirés **côté
  serveur** avant le rendu : ils n'apparaissent ni dans le HTML ni dans les données JSON.
  Un groupe sans aucun lien visible est lui aussi retiré.
- **F5** — Un groupe s'affiche en accordéon : fermé, il montre son visuel, son titre, son
  sous-titre et les icônes de ses plateformes ; ouvert, ses liens. L'option
  « ouvert par défaut » l'ouvre au chargement.
- **F6** — Chaque lien affiche l'icône de sa plateforme déduite de son domaine (liste
  fixe de marques, icône générique sinon) ou l'image téléversée si elle existe.
- **F7** — Un bouton de partage ouvre la feuille de partage du système, ou copie l'URL
  sinon.
- **F8** — Le HTML contient une version **sans JavaScript** de la page (titre, description,
  liens et groupes) ; les métadonnées Open Graph et le titre sont rendus par le serveur
  (image Open Graph : l'avatar de la page s'il est une image, sinon `/og-default.png`).
- **F9** — Les fonds animés (thèmes `nuit`, `papier`, `neon`, `pop`) ne se lancent que si
  WebGL 2 est disponible et que le visiteur n'a demandé ni moins d'animations
  (`prefers-reduced-motion`) ni l'économie de données (`Save-Data`). Sinon, le fond CSS
  du thème reste seul. Le thème `zazi` n'a pas de fond animé.

**Thèmes**

- **F10** — Cinq thèmes (`zazi`, `nuit`, `papier`, `neon`, `pop`) et huit accents
  (`violet`, `bleu`, `cyan`, `vert`, `ambre`, `orange`, `rose`, `neutre`) ; chaque thème
  traduit l'accent en couleurs concrètes. Défauts d'une nouvelle page : `zazi` / `violet`.
  La décoration d'un thème est initialisée par l'identifiant de la page : une page garde
  toujours le même aspect.

**Administration**

- **F11** — `/admin` affiche une application à page unique ; sans session, un écran de
  connexion. Toutes les URL `/admin/…` renvoient la même coquille HTML, `noindex`.
- **F12** — La liste des pages montre, pour chacune, avatar, titre, slug, état (publiée,
  listée) et nombre de liens ; l'ordre se modifie (`PUT /admin/api/pages/order`) et
  détermine l'ordre de l'index public.
- **F13** — Créer une page demande un titre et un slug ; la page est créée en brouillon,
  avec thème et accent par défaut, une forme d'avatar tirée au hasard, en dernière
  position. La disponibilité d'un slug se vérifie avant l'envoi
  (`GET /admin/api/slugs/{slug}`).
- **F14** — L'éditeur modifie le document complet d'une page (réglages, liens, groupes,
  ordre, déplacement d'un lien dans ou hors d'un groupe, suppression), avec un **aperçu**
  fidèle à la page publique où les brouillons sont signalés « Brouillon ». Un seul
  enregistrement (`PUT /admin/api/pages/{id}`) remplace tout le document, atomiquement.
- **F15** — **Versionnage optimiste** : le document porte un `version` ; l'enregistrement
  est refusé (**409**) si la page a été enregistrée entre-temps (autre onglet). Chaque
  enregistrement réussi incrémente la version et la renvoie.
- **F16** — Supprimer une page supprime ses éléments ; l'URL publique répond ensuite 404.
- **F17** — Un avatar (page) ou un visuel (groupe, lien) est **une image téléversée ou une
  forme** (bibliothèque de formes `coolshapes`), jamais les deux. Les liens n'ont que des
  images, jamais de forme.

**Validation des documents** (côté serveur, messages en français par champ)

- **F18** — Slug : 1 à 40 caractères, minuscules sans accents, chiffres et tirets (pas de
  tiret au début, à la fin ni double), **unique**, hors slugs réservés (`admin`, `api`,
  `assets`, `images`, `shapes`, `healthz`, `readyz`, `version`).
- **F19** — Textes : titre 1 à 80 caractères, sous-titre de groupe 0 à 80, description
  0 à 280 (sauts de ligne permis), comptés en caractères, sans caractère de contrôle,
  UTF-8 valide, espaces de bord retirés.
- **F20** — URL d'un lien : 2048 caractères au plus, sans espace ni caractère de contrôle,
  schémas **`http`, `https`, `mailto`, `tel` uniquement** ; `http(s)` avec hôte et sans
  identifiants ; `mailto` avec `@` ; `tel` au format RFC 3966.
- **F21** — Structure : au plus **100 éléments au premier niveau**, **500 au total**,
  **30 liens par groupe** ; groupes au premier niveau seulement ; identifiants d'éléments
  = UUID uniques générés par l'éditeur, jamais partagés entre deux pages.
- **F22** — Une validation échouée répond **422** avec la liste des champs en erreur
  (`slug`, `title`, `items.<uuid>.url`…), que l'éditeur surligne ; toutes les erreurs
  sont renvoyées d'un coup.

**Images**

- **F23** — `POST /admin/api/images` (multipart, champ `file`) accepte du **JPEG ou du
  PNG** de 2048 × 2048 pixels au plus et de `MAX_UPLOAD_MB` Mo au plus (5 par défaut). Le
  serveur vérifie le format, **redimensionne à 512 px** de côté au plus et **ré-encode**
  (JPEG qualité 85 si l'image est opaque, PNG sinon), ce qui supprime toutes les
  métadonnées (EXIF, position GPS). L'image est stockée **en base**.
- **F24** — `GET /images/{id}` sert l'image avec son type, en cache public d'un an
  (`immutable`) : un nouvel envoi crée un nouvel identifiant.
- **F25** — Les images qu'aucune page ni aucun élément n'utilise, créées depuis plus de
  24 h (remplacées ou abandonnées), sont **supprimées** toutes les heures.

**Authentification**

- **F26** — Un **mot de passe unique** protège l'admin ; l'application ne connaît que son
  hash bcrypt (`ADMIN_PASSWORD_HASH`). `make hash-password` en génère un (12 caractères
  au moins, 72 octets au plus).
- **F27** — `POST /admin/api/login` ouvre une session de **30 jours** (jeton aléatoire de
  256 bits ; seule l'empreinte SHA-256 est stockée en base) et pose un cookie `HttpOnly`,
  `SameSite=Strict`, `Secure` et préfixé `__Host-` en HTTPS. `POST /admin/api/logout` la
  ferme. Les sessions expirées sont purgées toutes les heures.
- **F28** — Changer `ADMIN_PASSWORD_HASH` **invalide toutes les sessions** existantes
  (chaque session mémorise l'empreinte du hash avec lequel elle a été créée).
- **F29** — Les tentatives de connexion sont limitées **par IP** (`RATE_LIMIT_LOGIN` par
  heure, 10 par défaut, fenêtre glissante, en mémoire) : au-delà, **429**. Seul le
  endpoint de connexion est limité ; les autres routes admin exigent la session.

### Non fonctionnelles

- **NF1** — **Budget de poids** vérifié par `make build` (`scripts/check-budget.mjs`) :
  JavaScript initial de la page publique ≤ **150 Kio gzip** ; chaque fond animé ≤ **30 Kio
  gzip**, chargé après l'affichage du contenu.
- **NF2** — **CSP stricte** sur toutes les pages (`default-src 'self'`, pas de script ni de
  style en ligne, `frame-ancestors 'none'`) ; l'admin y ajoute un nonce par réponse pour
  les styles injectés par ses boîtes de dialogue. Aucune ressource d'un tiers.
- **NF3** — **Protection CSRF** des requêtes d'écriture de l'admin (refus des requêtes
  d'une autre origine, selon `Sec-Fetch-Site` / `Origin`) ; réponses de l'API en
  `Cache-Control: no-store`.
- **NF4** — Un seul réplica : la limitation des tentatives est en mémoire.
- **NF5** — Corps de requête bornés (16 Kio pour les petites requêtes, 1 Mio pour un
  document de page) ; JSON strict (champs inconnus refusés).
- **NF6** — Accessibilité : navigation au clavier avec focus visible, accordéon accessible,
  animations réduites si le visiteur le demande.
- **NF7** — Vie privée : aucun cookie côté visiteur, aucun pisteur, aucune requête vers un
  tiers ; seul l'admin dépose un cookie de session.

### Interfaces

Pages publiques (HTML, `GET` uniquement) :

| Chemin | Rôle | Réponses |
|--------|------|----------|
| `/` | Index des pages publiées et listées | 200 |
| `/{slug}` | Page de liens | 200 ; 404 (inconnue, brouillon ou slug invalide) |
| `/images/{id}` | Image téléversée | 200 ; 404 |
| `/assets/…`, `/shapes/…`, `/favicon.svg`, `/og-default.png`, `/robots.txt` | Fichiers statiques (assets et formes : cache d'un an) | 200 ; 404 |
| `/healthz`, `/readyz`, `/version` | Liveness, readiness (base joignable), SHA du build | 200 ; 503 pour `/readyz` |
| `/admin`, `/admin/…` | Coquille de l'application d'admin | 200 |

API d'administration (JSON ; sauf `login`, session requise → 401 sinon) :

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `POST /admin/api/login` | `{password}` → ouvre la session | 200 ; 401 `bad_password` ; 429 |
| `POST /admin/api/logout` | Ferme la session | 204 |
| `GET /admin/api/session` | Session valide ? | 200 ; 401 |
| `GET /admin/api/pages` | Liste des pages (résumés) | 200 |
| `POST /admin/api/pages` | `{title, slug}` → crée un brouillon | 201 (document) ; 422 |
| `PUT /admin/api/pages/order` | `{ids}` : ordre de toutes les pages | 204 ; 422 |
| `GET /admin/api/pages/{id}` | Document complet (brouillons inclus) | 200 ; 404 |
| `PUT /admin/api/pages/{id}` | Remplace le document (avec `version`) → `{version}` | 200 ; 404 ; 409 ; 413 ; 422 |
| `DELETE /admin/api/pages/{id}` | Supprime la page | 204 ; 404 |
| `GET /admin/api/slugs/{slug}?except={id}` | Slug valide et disponible ? | 200 |
| `POST /admin/api/images` | Téléverse une image → visuel | 201 ; 400 ; 413 ; 422 |

Les erreurs ont la forme `{error: {code, message, fields?}}` ; les codes
(`bad_request`, `validation`, `unauthorized`, `bad_password`, `not_found`, `conflict`,
`too_large`, `rate_limited`, `internal`) et tous les messages sont définis à **un seul
endroit** : `internal/api/types.go`, aligné avec `web/src/api/types.ts` par un test de
contrat (`web/src/api/contract.json`). Les pages publiques reçoivent leurs données dans
`<script id="page-data">`.

### Données

PostgreSQL, base `linktree` (migration `001_init.sql`, identifiants UUIDv7) :

- **`pages`** — `slug` (unique), `title`, `description`, `avatar_image_id` ou
  `avatar_shape` (au plus un), `theme`, `accent`, `published`, `listed`, `position`,
  `version`, `created_at`, `updated_at`.
- **`items`** — liens et groupes d'une page : `page_id` (suppression en cascade),
  `parent_id` (le groupe d'un lien ; groupes au premier niveau seulement), `kind`
  (`link` | `group`), `position`, `title`, `subtitle`, `url` (liens seulement),
  `image_id` ou `shape` (formes : groupes seulement), `open_by_default`, `published`.
  Des contraintes `CHECK` rejouent les règles de F17 et F21.
- **`images`** — `content_type` (JPEG ou PNG), `data` (`bytea`, déjà redimensionnée),
  `width`, `height`, `created_at`. Sauvegardées avec le reste par le `pg_dump` quotidien.
- **`sessions`** — `token_hash` (SHA-256), `password_fingerprint`, `created_at`,
  `expires_at`.

Rétention : images orphelines de plus de 24 h et sessions expirées, purgées toutes les
heures par une tâche interne (idempotente).

### Configuration

Toutes dans `apps/linktree/.env.example` :

| Variable | Rôle | Défaut |
|----------|------|--------|
| `PORT` | Port d'écoute | `8080` |
| `DATABASE_URL` | Connexion PostgreSQL (obligatoire) | — |
| `ADMIN_PASSWORD_HASH` | Hash bcrypt du mot de passe de l'admin (obligatoire) | — |
| `PUBLIC_BASE_URL` | URL publique : URL Open Graph absolues, cookies `Secure` si `https://` | `http://localhost:8080` |
| `TRUST_PROXY` | Lire l'IP client dans `X-Forwarded-For` (derrière Traefik seulement) | `false` |
| `RATE_LIMIT_LOGIN` | Tentatives de connexion par IP et par heure | `10` |
| `MAX_UPLOAD_MB` | Taille maximale d'une image envoyée | `5` |
| `VITE_DEV_URL` | Serveur Vite en développement (vide en production) | vide |

## 5. Critères d'acceptation

Les critères sont en grande partie couverts par les tests du dépôt (`make test`) ; la
recette de production est en §9.

- **CA1** (F1, F3) — Étant donné trois pages publiées dont une non listée, quand je
  charge `/`, alors seules les deux listées apparaissent, dans l'ordre de l'admin ;
  `/{slug}` de la troisième répond 200 avec `noindex`.
- **CA2** (F2) — Quand je charge `/inconnu`, `/Admin` ou le slug d'une page en brouillon,
  alors la réponse est 404 avec la page « Page introuvable ».
- **CA3** (F4) — Étant donné un lien en brouillon et un groupe dont tous les liens sont en
  brouillon, quand je charge la page publique, alors ni leur titre ni leur URL ne figurent
  dans le HTML.
- **CA4** (F5) — Étant donné un groupe « ouvert par défaut », quand la page se charge,
  alors il est ouvert ; les autres sont fermés et s'ouvrent au clic ou au clavier.
- **CA5** (F8) — Quand JavaScript est désactivé, alors la page affiche titre, description,
  liens et groupes ; `og:title`, `og:description`, `og:url` et `og:image` sont présents.
- **CA6** (F9) — Étant donné `prefers-reduced-motion: reduce` (ou `Save-Data`, ou pas de
  WebGL 2), quand j'ouvre une page au thème `neon`, alors aucun fond animé n'est chargé.
- **CA7** (F13, F18) — Quand je crée une page avec le slug `admin`, `Mon Slug` ou un slug
  déjà pris, alors la réponse est 422 avec une erreur sur `slug` ; avec `musique`, la page
  est créée en brouillon.
- **CA8** (F15) — Étant donné la même page ouverte dans deux onglets, quand j'enregistre
  dans le premier puis dans le second, alors le second reçoit 409 et aucune donnée n'est
  écrasée.
- **CA9** (F20) — Quand j'enregistre un lien `javascript:alert(1)`, `data:…`,
  `https://u:p@exemple.fr` ou `https://a b.fr`, alors la réponse est 422 sur
  `items.<uuid>.url` ; `https://…`, `mailto:a@b.fr` et `tel:+33123456789` sont acceptés.
- **CA10** (F21) — Quand j'enregistre 101 éléments au premier niveau, un groupe de
  31 liens, ou un groupe dans un groupe, alors la réponse est 422.
- **CA11** (F17) — Quand j'enregistre un avatar avec une image **et** une forme, ou un
  lien avec une forme, alors la réponse est 422.
- **CA12** (F23) — Quand j'envoie un GIF, un SVG ou un fichier texte, alors 422 ; un
  fichier de plus de `MAX_UPLOAD_MB` : 413 ; une image de plus de 2048 px de côté : 422 ;
  un JPEG de 1500 × 1000 est servi en 512 × 341, sans EXIF.
- **CA13** (F24) — Quand je demande `/images/{id}`, alors l'en-tête
  `Cache-Control: public, max-age=31536000, immutable` est présent ; un identifiant
  invalide ou inconnu répond 404.
- **CA14** (F25) — Étant donné une image créée il y a plus de 24 h et utilisée nulle part,
  quand la purge s'exécute, alors elle est supprimée ; une image utilisée ou récente est
  conservée.
- **CA15** (F27) — Quand je me connecte avec le bon mot de passe, alors la réponse pose un
  cookie `HttpOnly`, `SameSite=Strict` (et `Secure` + `__Host-` en HTTPS) ; avec un mauvais
  mot de passe, 401 `bad_password`, sans cookie.
- **CA16** (F28) — Étant donné une session ouverte, quand `ADMIN_PASSWORD_HASH` change et
  que l'application redémarre, alors la session précédente reçoit 401.
- **CA17** (F29) — Quand j'envoie plus de `RATE_LIMIT_LOGIN` tentatives en une heure depuis
  la même IP, alors la suivante reçoit 429, y compris avec le bon mot de passe.
- **CA18** (NF3) — Quand un site tiers envoie un `PUT` ou un `POST` vers `/admin/api/…`
  (`Sec-Fetch-Site: cross-site`), alors la réponse est 403.
- **CA19** (NF2) — Les réponses HTML portent une `Content-Security-Policy` sans
  `unsafe-inline` pour les scripts ; un titre contenant `</script><script>…` s'affiche
  comme du texte et ne s'exécute pas.
- **CA20** (NF1) — `make build` échoue si le JavaScript initial de la page publique
  dépasse 150 Kio gzip ou un fond animé 30 Kio gzip.
- **CA21** (F12) — Quand je réordonne les pages avec une liste incomplète ou en double,
  alors 422 ; avec la liste complète, l'index public suit le nouvel ordre.
- **CA22** — `GET /readyz` répond 200 quand la base est joignable et 503 sinon ;
  `GET /version` renvoie le SHA du build déployé.

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Mot de passe unique deviné ou fuité | Modification ou effacement des pages | Faible | bcrypt, limitation par IP, cookie strict, mot de passe de 12 caractères au moins ; sauvegardes `pg_dump` |
| Limitation des tentatives en mémoire : remise à zéro à chaque redémarrage, inefficace avec plusieurs réplicas | Contournement partiel | Faible | Un seul réplica (NF4) ; revoir si le service passe à plusieurs pods |
| Images en base : la base grossit | Sauvegardes et disque plus lourds | Faible | Redimensionnement à 512 px, purge des orphelines ; revoir si le stockage dépasse quelques centaines de Mio |
| `X-Forwarded-For` falsifié si `TRUST_PROXY=true` hors Traefik | Limitation contournée | Faible | Seule l'entrée ajoutée par le proxy est lue ; activer seulement derrière Traefik |
| Fonds animés lourds sur mobile | Fluidité, batterie | Moyenne | Chargement différé, désactivation selon préférences (F9), budget NF1 |
| Édition simultanée sur deux appareils | Perte de modifications | Moyenne | Versionnage optimiste (F15) |

## 7. Décisions ouvertes

Aucune ne bloque l'existant : ce sont des évolutions possibles.

- [ ] D1 — **Statistiques de clics** (sans cookie ni pisteur) ? **Proposition :** non pour
  l'instant ; si oui, compteur agrégé par lien et par jour, sans IP (bloque : tout ticket
  d'évolution « statistiques »).
- [ ] D2 — **Liens programmés** (publication ou masquage à date) ? **Proposition :** non
  pour l'instant (publier/dépublier à la main suffit) (bloque : tout ticket d'évolution
  « programmation »).
- [ ] D3 — **Sauvegarde des images** : les images sont dans la base, donc dans le `pg_dump`
  quotidien. **Proposition :** rien à faire ; vérifier la taille de la base à la recette
  (bloque : rien).

## 8. Découpage en tickets

Le développement et le déploiement initial sont faits : aucun ticket de développement
n'est à créer pour cette spec. Les évolutions (D1, D2) donneront leur propre spec
(`docs/specs/linktree-<sujet>.md`) si Noah les retient.

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | Documenter l'existant : cette spec, `apps/linktree/CLAUDE.md`, tableau du `CLAUDE.md` racine (ONBO-47) | toutes | — | dev (documentation) |
| T2 | 2 | Recette de production de la version déployée (§9), écarts consignés | CA1 à CA22 | T1 | déploiement |

## 9. Recette après déploiement

À jouer sur `https://link.zazi.cool` (mot de passe d'admin en main) :

- [ ] `curl -sI https://link.zazi.cool/healthz` → `200` ; `/readyz` → `200`
- [ ] `curl -s https://link.zazi.cool/version` → le SHA du tag de
  `infra/apps/linktree/overlays/prod/kustomization.yaml`
- [ ] `curl -sI http://link.zazi.cool/` → redirection vers HTTPS ; certificat valide
- [ ] `/` liste uniquement les pages publiées et listées ; `/inconnu` → 404
- [ ] Une page de test : création en brouillon (404 en public), publication (200), ajout
  d'un groupe avec deux liens, d'un lien en brouillon (absent du HTML : `curl -s … | grep`)
- [ ] Mauvais mot de passe → message d'erreur ; 11 essais d'affilée → 429
- [ ] Cookie de session `__Host-session` : `HttpOnly`, `Secure`, `SameSite=Strict`
- [ ] Upload d'un JPEG avec EXIF : l'image servie par `/images/{id}` n'en contient plus
- [ ] En-têtes : `Content-Security-Policy`, `X-Content-Type-Options`, `Referrer-Policy`
- [ ] Sur un téléphone : accordéon, partage, thème au fond animé ; avec « réduire les
  animations » activé, fond statique
- [ ] Lighthouse mobile sur une page publique (performance, accessibilité, bonnes pratiques,
  SEO) : noter les scores dans le ticket
- [ ] Sauvegarde du jour présente et base `linktree` incluse (`infra/data`)
