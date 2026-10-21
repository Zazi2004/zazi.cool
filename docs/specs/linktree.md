# Spec — Linktree (`link.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **Brouillon** — à relire avec Noah (décisions D1 à D5) |
| Issue Multica  | ONBO-43 |
| Application    | `apps/linktree/` |
| Auteur, valideur | Rédacteur de specs / Noah |

## 1. Contexte

Page personnelle de liens, à la manière de Linktree : une page unique, soignée, lue
surtout sur mobile, qui regroupe les liens de Noah (réseaux, projets, contact). Le dossier
`apps/linktree/` ne contient aujourd'hui que son `CLAUDE.md` (périmètre envisagé, deux
décisions ouvertes). Le DNS wildcard `*.zazi.cool` couvre déjà `link.zazi.cool` ; l'app n'a
ni code, ni manifests. Le job `linktree` de `.github/workflows/ci.yml` est déjà câblé et
s'exécutera dès que `apps/linktree/Dockerfile` existera.

## 2. Objectifs

- **O1** — Un visiteur voit l'identité de Noah et tous ses liens sans défiler inutilement
  ni zoomer, sur un téléphone.
- **O2** — Modifier la liste des liens = modifier un fichier versionné, relire en PR,
  déployer ; aucun code à toucher.
- **O3** — Page légère et rapide : moins de 50 Kio transférés hors avatar, aucune
  requête vers un tiers, Lighthouse ≥ 95 (performance, accessibilité, bonnes pratiques, SEO).
- **O4** — Le service reste négligeable sur le VPS partagé (pas de base, pas d'état).

## 3. Périmètre

### Inclus (MVP)
- Page publique unique : avatar, nom, bio courte, liste ordonnée de liens (titre, URL,
  icône), pied de page minimal.
- Contenu lu dans un fichier JSON embarqué dans le binaire (D1).
- Mode clair/sombre selon la préférence du système, sans JS.
- Métadonnées de partage (Open Graph, favicon), `robots.txt`.
- Endpoints `/healthz`, `/readyz`, `/version`.

### Hors périmètre
- Interface d'administration, base de données (D1) : piste ultérieure du `CLAUDE.md` de l'app.
- Statistiques de clics, liens programmés ou masqués temporairement.
- Bascule manuelle clair/sombre (nécessiterait du JS et un cookie ou `localStorage`).
- Plusieurs pages ou plusieurs profils.

## 4. Exigences

### Fonctionnelles

- **F1** — `GET /` renvoie une page HTML rendue côté serveur (`html/template`) avec
  l'avatar, le nom, la bio et la liste des liens dans l'ordre du fichier de contenu.
- **F2** — Chaque lien est un `<a>` (titre, icône) vers son URL ; les liens externes
  s'ouvrent dans le même onglet avec `rel="noopener noreferrer me"`. Les liens `mailto:`
  sont acceptés.
- **F3** — Le contenu vient d'un fichier `content/links.json` embarqué (`embed`) : profil
  (`name`, `bio`, `avatar`) et liste de liens (`title`, `url`, `icon`). Il est **validé au
  démarrage** : un fichier invalide (champ vide, URL non `https:`/`mailto:`, icône
  inconnue, liste vide) empêche l'app de démarrer avec un message clair.
- **F4** — Les icônes sont des SVG embarqués, choisis dans une liste fermée (clé `icon`).
  Une icône générique « lien » sert de défaut.
- **F5** — Le thème suit `prefers-color-scheme` (CSS uniquement) ; la page est lisible et
  utilisable sans JavaScript (il n'y en a pas).
- **F6** — La page contient les balises Open Graph et Twitter Card (titre, description,
  image), un favicon et `<meta name="theme-color">`.
- **F7** — Les ressources statiques (`/static/…`) sont servies avec un cache long
  (`Cache-Control: public, max-age=31536000, immutable`) ; leur nom contient un hash de
  contenu. La page HTML est servie avec `max-age=300`.
- **F8** — `GET /version` renvoie le SHA git de l'image (injecté au build), pour la
  vérification des déploiements (`docs/workflow/cycle-de-vie.md`).

### Non fonctionnelles

- **NF1** — Sécurité : CSP stricte sans JS (`default-src 'none'; style-src 'self';
  img-src 'self'; form-action 'none'; frame-ancestors 'none'; base-uri 'none'`),
  `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`,
  HSTS posé par Traefik ou l'app. Pas de police, script ni image externe.
- **NF2** — Accessibilité : un seul `<h1>`, contraste AA en clair et en sombre, cibles
  tactiles ≥ 44 px, navigation clavier avec focus visible, `alt` sur l'avatar, `lang="fr"`.
- **NF3** — Vie privée : aucun cookie, aucun tracker, aucune statistique ; les logs
  d'accès ne contiennent pas d'IP complète au-delà de ce que Traefik journalise déjà.
- **NF4** — Ressources : requests `10m` CPU / `16Mi` mémoire, limits `100m` / `64Mi`
  (à ajuster après mesure) ; `readOnlyRootFilesystem: true` (rien n'est écrit).
- **NF5** — Poids : CSS inline ou un seul fichier ≤ 10 Kio, avatar ≤ 30 Kio (WebP, 256 px).

### Interfaces

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| `GET /` | Page des liens | 200 |
| `GET /static/{fichier}` | CSS, icônes, avatar, favicon, image de partage | 200 ; 404 |
| `GET /robots.txt` | Autorise l'indexation de `/` | 200 |
| `GET /healthz`, `GET /readyz` | Santé (readyz : contenu chargé et valide) | 200 ; 503 |
| `GET /version` | SHA de build (`text/plain`) | 200 |

Toute autre méthode que `GET`/`HEAD` renvoie `405` ; tout autre chemin, une page `404`
sobre dans le même style.

### Données

Aucune base. Fichier `content/links.json` (exemple de forme, valeurs provisoires, D2) :

```json
{
  "name": "Noah",
  "bio": "Développeur. Projets perso et liens utiles.",
  "avatar": "avatar.webp",
  "links": [
    { "title": "GitHub", "url": "https://github.com/Zazi2004", "icon": "github" },
    { "title": "Pastebin", "url": "https://pb.zazi.cool", "icon": "link" },
    { "title": "Contact", "url": "mailto:noah.maire@gmail.com", "icon": "mail" }
  ]
}
```

Le JSON est choisi plutôt que le YAML : la bibliothèque standard le lit, sans dépendance
(`CLAUDE.md` racine §5).

### Configuration

| Variable | Rôle | Défaut |
|----------|------|--------|
| `PORT` | Port d'écoute | `8080` |
| `BASE_URL` | URL publique, pour `og:url` et l'URL absolue de l'image de partage | `https://link.zazi.cool` |

## 5. Critères d'acceptation

- **CA1** (F1, F2) — Étant donné un `links.json` de trois liens, quand j'ouvre `/`, alors
  la page contient le nom, la bio, l'avatar et exactement trois liens, dans l'ordre du
  fichier, avec les bons `href` (test HTTP sur le HTML rendu).
- **CA2** (F3) — Un `links.json` avec un champ vide, une URL `http:` ou `javascript:`, une
  icône inconnue ou aucun lien fait échouer le chargement avec une erreur nommant le champ
  fautif (tests unitaires table-driven) ; `/readyz` ne répond jamais `200` avec un contenu invalide.
- **CA3** (F4, F5) — Aucun `<script>` dans la page ; la feuille de style contient un bloc
  `@media (prefers-color-scheme: dark)` ; toutes les icônes référencées existent.
- **CA4** (F6) — La page contient `og:title`, `og:description`, `og:image` (URL absolue
  construite depuis `BASE_URL`) et un lien favicon ; ces ressources répondent `200`.
- **CA5** (F7) — `/static/…` renvoie `Cache-Control` immutable, `/` renvoie `max-age=300`.
- **CA6** (F8) — `/version` renvoie le SHA passé au build (`dev` en local).
- **CA7** (NF1) — Chaque réponse HTML porte les en-têtes de NF1 (test HTTP).
- **CA8** (NF2) — Audit Lighthouse mobile ≥ 95 sur les quatre catégories et aucun échec
  de contraste ; navigation complète au clavier (relevé en commentaire de PR).
- **CA9** (NF5, O3) — Poids de `/` + CSS ≤ 50 Kio non compressés ; avatar ≤ 30 Kio
  (vérifié par un test qui lit les tailles embarquées).
- **CA10** (production) — `https://link.zazi.cool/healthz` et `/readyz` répondent `200`,
  `/version` renvoie le SHA déployé, `http://` redirige vers `https://`, certificat valide.

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Liste de liens ou identité visuelle non fournies | Moyen (retard de la recette) | Moyenne | Valeurs provisoires (D2, D3) ; le contenu est un fichier qu'on change sans toucher au code |
| Modifier un lien exige un déploiement | Faible (assumé pour un site perso) | Certaine | Déploiement par Flux : merge de la PR = en ligne ; D1 garde la porte ouverte à une admin |
| Lien malveillant ajouté par erreur (`javascript:`) | Moyen | Faible | Validation au démarrage (F3, CA2) |
| Avatar trop lourd ou personnel publié sans accord | Faible | Faible | Limite de poids (CA9) ; l'avatar est choisi par Noah (D3) |
| `Dockerfile` absent : le job CI `linktree` ne tourne pas | Faible | Certaine tant que T1 n'est pas mergé | T1 ajoute le `Dockerfile` ; vérifier que le job devient requis dans `ci / required` |

## 7. Décisions ouvertes

- [ ] **D1** — Statique ou dynamique. **Proposition :** **statique** (contenu JSON
  embarqué, sans base) pour le MVP, comme le recommande le `CLAUDE.md` de l'app. Si Noah
  veut l'admin en ligne ou les statistiques tout de suite, le découpage change (base,
  authentification, ticket d'admin). (bloque : tout le découpage ; à trancher avant validation)
- [ ] **D2** — **Liste initiale des liens.** **Proposition :** GitHub (`Zazi2004`), le
  pastebin, Undercover et un lien de contact par e-mail ; Noah fournit la liste réelle
  (réseaux, projets, contact) avant la recette. (bloque : T7 ; T2 peut avancer avec la liste provisoire)
- [ ] **D3** — **Identité visuelle** : nom affiché, bio, avatar, palette, police.
  **Proposition :** nom « Noah », bio d'une ligne, avatar fourni par Noah (initiale sur
  fond uni tant qu'il n'est pas livré), palette sobre claire/sombre avec une couleur
  d'accent, police système (aucun chargement externe). (bloque : T3, T7)
- [ ] **D4** — **Partage de l'identité visuelle avec l'accueil** (`apps/home/`).
  **Proposition :** non partagée dans le MVP ; chaque app embarque son propre CSS
  (pas de module commun avant d'avoir deux apps stables). (ne bloque rien)
- [ ] **D5** — **Indexation par les moteurs de recherche.** **Proposition :** indexable
  (`robots.txt` autorisant `/`), puisque c'est une page de présentation publique.
  (bloque : T2)

D1 se tranche à la validation de la spec. Ensuite, seules D2 et D3 demandent une action de Noah avant la fin de la première étape de dev ;
la spec peut être validée avec elles ouvertes, aucun ticket de l'étape 1 n'en dépend.

## 8. Découpage en tickets

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | Squelette de l'app : `cmd/linktree`, `/healthz` `/readyz` `/version` (SHA injecté au build), arrêt propre, en-têtes de sécurité (NF1), `Makefile` (cibles standard), `compose.yaml`, `Dockerfile` multi-stage non-root, `.env.example`, `README.md` ; vérifier que le job CI `linktree` s'exécute | F8, NF1, NF4 ; CA6, CA7 | — | dev + infra |
| T2 | 1 | Manifests `infra/apps/linktree/{base,overlays/prod}` : namespace, Deployment (ressources, sondes, `securityContext`), Service, Ingress `link.zazi.cool` + Middleware `redirect-https`, Kustomization Flux dans `infra/flux/clusters/prod/apps.yaml` ; aucun secret ni base nécessaires | NF4 | — | infra |
| T3 | 2 | Contenu et rendu : chargement et validation de `content/links.json`, template `html/template`, icônes SVG, avatar, liste fournie en D2 (provisoire si besoin) | F1–F4 ; CA1–CA3 | T1 | dev |
| T4 | 2 | Style : CSS mobile-first, clair/sombre, accessibilité, cache des statiques avec hash | F5, F7, NF2, NF5 ; CA3, CA5, CA8, CA9 | T3 | dev |
| T5 | 2 | Métadonnées de partage : Open Graph, Twitter Card, favicon, image de partage, `robots.txt` | F6 ; CA4 | T3 | dev |
| T6 | 3 | Reporter les règles durables dans `apps/linktree/CLAUDE.md` (statut, contenu, endpoints, décisions) et le statut du tableau du `CLAUDE.md` racine | — | T1–T5 | dev (doc) |
| T7 | 3 | **Déployer linktree `<sha7>`** : premier déploiement (`newTag` dans `infra/apps/linktree/overlays/prod/`, PR de bump, vérification `deploy-verify`), recette §9, ajout au service d'uptime quand il sera choisi | CA10 | T1–T6, D2, D3 | déploiement |

T4 et T5 avancent en parallèle sur des branches distinctes, après T3. Le déploiement suit
l'étape 3 de la trajectoire (`docs/workflow/cycle-de-vie.md`) : Noah merge la PR de bump,
Flux réconcilie, le job `verify` contrôle. Un agent ne touche jamais au cluster.

## 9. Recette après déploiement

- [ ] `curl -sI https://link.zazi.cool/healthz` → `200` ; `/readyz` → `200` ;
      `curl -s https://link.zazi.cool/version` → SHA déployé (vérifié par le job `verify`).
- [ ] `curl -sI http://link.zazi.cool/` → redirection vers `https://`.
- [ ] Certificat Let's Encrypt valide pour `link.zazi.cool`.
- [ ] Ouvrir la page sur un téléphone en clair puis en sombre : tous les liens de D2
      sont présents, dans l'ordre, et s'ouvrent.
- [ ] Coller l'URL dans un aperçu de partage (messagerie) : titre, description et image.
- [ ] Lighthouse mobile ≥ 95 sur la production.
- [ ] Mettre à jour le tableau des projets du `CLAUDE.md` racine (statut « Production »)
      et retirer `link.zazi.cool` des tâches de cadrage.
