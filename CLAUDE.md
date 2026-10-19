# zazi.cool — Services web personnels

Ce fichier donne le contexte global du dépôt et les règles communes à tous les projets.
Chaque application a son propre `CLAUDE.md` (dans `apps/<nom>/`) qui précise son contexte
et **prime sur ce fichier** en cas de conflit pour ce qui la concerne.

---

## 1. Contexte

- Ensemble de **services web personnels** hébergés sur un **VPS distant personnel**.
- Nom de domaine : **`zazi.cool`**. Chaque projet est exposé sur **son propre sous-domaine**.
- Objectif de déploiement : **simple mais efficace**, reproductible, automatisé, sans
  sur-ingénierie. Un seul développeur, des charges faibles, des ressources limitées (un VPS).
- En local, chaque projet doit pouvoir démarrer **avec ses dépendances (bases de données,
  cache…) via Docker Compose**, en une seule commande.

### Projets

| Projet       | Dossier            | Sous-domaine                                    | Statut     |
|--------------|--------------------|-------------------------------------------------|------------|
| Accueil      | `apps/home/`       | `zazi.cool` (+ `www.zazi.cool` → redirection 301) | Cadrage    |
| Pastebin     | `apps/pastebin/`   | `pb.zazi.cool`                                  | Production (2026-10-19) |
| Linktree     | `apps/linktree/`   | `link.zazi.cool`                                | Cadrage    |
| Undercover   | `apps/undercover/` | `undercover.zazi.cool`                          | Production (2026-10-14) |

Les spécifications sont rédigées dans `docs/specs/<app>.md` pendant le cadrage, puis leurs
règles durables sont reportées dans le `CLAUDE.md` de l'app. Workflow complet : `docs/workflow/`.

---

## 2. Architecture cible

```
                Internet
                   │
        DNS *.zazi.cool → IP du VPS
                   │
┌──────────────────┴───────────────────────────────┐
│ VPS OVHcloud (2 vCPU / 4 Go RAM)                  │
│  k3s (Kubernetes mono-nœud)                       │
│   ├─ Traefik (Ingress, inclus dans k3s)           │
│   ├─ cert-manager (TLS Let's Encrypt)             │
│   ├─ ns: home        → Deployment + Service       │
│   ├─ ns: pastebin    → Deployment + Service       │
│   ├─ ns: linktree    → Deployment + Service       │
│   ├─ ns: undercover  → Deployment + Service       │
│   └─ ns: data        → PostgreSQL (+ Redis si besoin)
└───────────────────────────────────────────────────┘
                   │ pg_dump quotidien
                   ▼
        OVHcloud Object Storage (S3)
```

### Choix retenus (validés le 2026-10-14)

- **Hébergement : VPS OVHcloud, 2 vCPU / 4 Go RAM**. Assez pour k3s, PostgreSQL,
  cert-manager et les applis légères ; à réévaluer si la mémoire devient un problème.
- **Stack applicative : Go côté serveur, TypeScript pour les fronts interactifs.**
  - **Back-end : Go pour toutes les applications** (bibliothèque standard en priorité :
    `net/http`, `html/template`, `log/slog`, `embed`…). Binaire unique, images minimales,
    faible empreinte mémoire.
  - **Front par défaut : rendu serveur** (`html/template`) avec peu ou pas de JS.
  - **Front interactif (ex. Undercover) : TypeScript + Vite**, dans `web/`. Le build produit
    des fichiers statiques **embarqués dans le binaire Go** (`embed`) : une seule image,
    un seul déploiement, pas de serveur Node en production.
  - Les dépendances restent soumises à la règle du §5.

- **Kubernetes : k3s, un seul cluster mono-nœud**. Un cluster par projet serait du
  surdimensionnement pour un VPS unique ; l'isolation se fait par **namespace**.
  Un namespace/cluster de staging pourra être ajouté plus tard si le besoin se présente.
- **Ingress : Traefik** (fourni par k3s). Un `Ingress` par application, un host par sous-domaine.
- **TLS : cert-manager + Let's Encrypt** (challenge HTTP-01 par défaut, DNS-01 si un
  certificat wildcard devient utile).
- **DNS** : un enregistrement `A` (et `AAAA` si IPv6) pour `zazi.cool` et un wildcard
  `*.zazi.cool` pointant vers le VPS.
- **Manifests : Kustomize** (`base/` + `overlays/`), suffisant et sans templating complexe.
  Helm uniquement pour installer des composants tiers (cert-manager, opérateurs…).
- **Images : GitHub Container Registry (ghcr.io)**, construites par la CI.
- **Base de données** : une instance **PostgreSQL** partagée dans le cluster, avec
  **une base et un utilisateur dédiés par application** (aucune app n'accède aux données
  d'une autre). Redis uniquement si un projet en a réellement besoin (ex. temps réel).
- **Sauvegardes** : `CronJob` `pg_dump` quotidien vers **OVHcloud Object Storage**
  (S3-compatible), hors du disque du VPS. Le VPS et les sauvegardes partagent le même
  fournisseur : ce risque est accepté pour garder un seul fournisseur. Une sauvegarde non testée n'est pas une sauvegarde : prévoir un test
  de restauration.
- **Secrets** : jamais en clair dans le dépôt. Chiffrés avec **SOPS + age** ; en local,
  fichiers `.env` non versionnés. La clé privée age est conservée hors du dépôt et du VPS.
- **Monitoring : uptime externe uniquement** (service gratuit type UptimeRobot / Better Stack)
  qui teste le `/healthz` de chaque application et alerte. Rien ne tourne sur le VPS pour ça.
  Métriques (Prometheus/Grafana ou équivalent léger) seulement si un besoin concret apparaît.

---

## 3. Structure du dépôt

```
zazi/
├── CLAUDE.md                  # ce fichier
├── .sops.yaml                 # règles de chiffrement SOPS (clé publique age)
├── .github/workflows/<nom>.yml  # CI par application
├── apps/
│   ├── home/
│   ├── pastebin/
│   ├── linktree/
│   └── undercover/
└── infra/                     # procédures : infra/README.md
    ├── cluster/               # bootstrap du VPS, tunnel kubectl, Traefik, ClusterIssuers
    ├── data/                  # PostgreSQL, bases par application, sauvegardes
    └── apps/<nom>/            # manifests Kustomize par application
        ├── base/
        └── overlays/prod/     # tag d'image + secret.enc.yaml (SOPS)
```

Les secrets sont des fichiers `*.enc.yaml` chiffrés par SOPS (seules les valeurs le sont),
appliqués avec `sops decrypt <fichier> | kubectl apply -f -` (Kustomize ne déchiffre pas).

Structure attendue de chaque application :

```
apps/<nom>/
├── CLAUDE.md                  # contexte et spécifications du projet
├── README.md                  # démarrage rapide pour un humain
├── Dockerfile                 # multi-stage, image finale minimale, utilisateur non-root
├── compose.yaml               # app + dépendances pour le dev local
├── .env.example               # toutes les variables d'environnement, documentées
├── Makefile                   # cibles standard (voir §4)
├── go.mod
├── cmd/<nom>/main.go          # point d'entrée
├── internal/ …                # code applicatif (non importable hors de l'app)
├── migrations/                # migrations SQL (si base de données)
└── web/                       # front TypeScript + Vite (seulement si front interactif)
```

Les tests Go sont placés à côté du code testé (`*_test.go`), selon la convention Go.

---

## 4. Développement local

- Prérequis : Docker (avec Compose v2), `make`, Go, et Node.js pour les apps qui ont un
  front TypeScript (`web/`).
- Chaque application expose les **mêmes cibles `make`**. Si l'app a un front `web/`,
  les cibles couvrent **les deux parties** (ex. `make lint` = lint Go + lint TS) :

| Cible         | Rôle                                                        |
|---------------|-------------------------------------------------------------|
| `make up`     | Démarre l'app et ses dépendances via `docker compose up`    |
| `make down`   | Arrête et nettoie les conteneurs                            |
| `make dev`    | Lance l'app en mode développement (hot reload) sur les dépendances Compose |
| `make test`   | Lance la suite de tests                                     |
| `make lint`   | Lint + vérification du formatage                            |
| `make fmt`    | Formate le code                                             |
| `make build`  | Construit l'image Docker                                    |
| `make migrate`| Applique les migrations de base de données (si applicable)  |

- `compose.yaml` doit : démarrer la base avec un **healthcheck**, faire attendre l'app
  (`depends_on: condition: service_healthy`), appliquer les migrations au démarrage,
  et utiliser un volume nommé pour persister les données locales.
- Copier `.env.example` en `.env` pour le dev local. `.env` est ignoré par git.

---

## 5. Conventions de code

### Langue
- **Code, identifiants, commentaires, messages de commit : anglais.**
- **Documentation (`CLAUDE.md`, `README.md`, specs) : français.**
- **Interface utilisateur** : français par défaut, sauf indication contraire dans le projet.

### Principes
- Code **lisible avant d'être malin**. Fonctions courtes, noms explicites, pas de code mort.
- **Pas de dépendance ajoutée sans justification** ; préférer la bibliothèque standard
  quand elle suffit.
- Commentaires pour expliquer le **pourquoi**, pas le quoi.
- Gestion d'erreurs explicite ; aucune erreur avalée silencieusement.
- Valider toutes les entrées utilisateur côté serveur.

### Applications « cloud-ready » (12-factor)
- **Configuration uniquement par variables d'environnement**, toutes listées dans `.env.example`.
- **Sans état local** : aucune donnée persistante sur le disque du conteneur (tout en base
  ou en stockage externe), pour pouvoir redémarrer/remplacer un pod sans perte.
- **Logs structurés (JSON) sur stdout/stderr**, jamais dans des fichiers.
- Endpoints de santé : **`GET /healthz`** (liveness, le process répond) et
  **`GET /readyz`** (readiness, dépendances joignables).
- Arrêt propre sur `SIGTERM` (fin des requêtes en cours, fermeture des connexions).
- Écoute sur le port défini par `PORT` (défaut `8080`).

### Go
- Code **idiomatique et simple** : formaté par `gofmt`, vérifié par `go vet`, erreurs
  retournées et enrichies (`fmt.Errorf("…: %w", err)`), jamais ignorées.
- Pas de framework web : `net/http` (routage avec motifs depuis Go 1.22) suffit.
- `context.Context` propagé dans les appels I/O (base, HTTP) pour les timeouts et l'arrêt propre.

### TypeScript (fronts interactifs)
- `"strict": true`, aucune utilisation de `any` sans justification.
- Vite pour le dev (HMR) et le build ; `package-lock.json` versionné, installation via `npm ci`.
- Les messages échangés avec le serveur Go (HTTP/WebSocket) ont un **format défini à un seul
  endroit** et documenté ; les types Go et TS doivent rester alignés (tests à l'appui).
- Ne jamais faire confiance au client : toute règle métier est validée côté Go.

### Docker
- Build **multi-stage**, image finale minimale (distroless/alpine/slim), **utilisateur non-root**.
  Avec un front `web/` : étape Node (build Vite) → étape Go (compile en embarquant le
  build) → image finale distroless contenant uniquement le binaire.
- Tags d'images versionnés (SHA git et/ou semver), **jamais `latest` en production**.
- Fichier `.dockerignore` systématique.

### Base de données
- Schéma géré **exclusivement par des migrations versionnées** (jamais de modification
  manuelle en prod).
- Migrations rétro-compatibles quand c'est possible (déploiement sans interruption).
- Outil : **petit exécuteur maison**, sans dépendance. Les fichiers SQL de `migrations/`
  sont embarqués dans le binaire (`embed`) et appliqués dans l'ordre. Une table
  `schema_migrations` garde la trace de ceux déjà passés. L'exécuteur tourne au démarrage
  de l'app et via `make migrate`.

---

## 6. Qualité et tests

- Tests unitaires pour la logique métier ; tests d'intégration contre une vraie base
  (via Compose ou testcontainers) pour la couche de persistance.
- Un bug corrigé = un test de non-régression.
- `make lint` et `make test` doivent passer avant tout commit.
- **CI (GitHub Actions)**, par application, déclenchée uniquement si `apps/<nom>/**` change :
  lint → tests → build de l'image → push sur ghcr.io (sur `main`).

---

## 7. Git

- Branche principale : `main`, toujours déployable.
- Travail sur branches courtes : `feat/<app>-<sujet>`, `fix/<app>-<sujet>`, `chore/…`.
- **Conventional Commits** avec le nom de l'app en scope :
  `feat(pastebin): add expiration`, `fix(undercover): handle player reconnect`,
  `chore(infra): bump cert-manager`.
- Commits atomiques ; aucun secret, `.env`, binaire ou artefact de build versionné.

---

## 8. Déploiement et sécurité

### Déploiement
1. Merge sur `main` → la CI construit et pousse l'image taguée avec le SHA.
2. Mise à jour du tag dans `infra/apps/<nom>/overlays/prod/` → `kubectl apply -k`.
3. Objectif (décidé le 2026-10-16) : **déploiement 100 % automatisé à terme** (GitOps avec
   Flux recommandé), par étapes : voir `docs/workflow/cycle-de-vie.md`.

### Chaque Deployment doit définir
- `resources.requests` et `resources.limits` (le VPS est partagé entre toutes les apps).
- `livenessProbe` / `readinessProbe` sur `/healthz` et `/readyz`.
- `securityContext` : `runAsNonRoot: true`, `readOnlyRootFilesystem: true` si possible,
  `allowPrivilegeEscalation: false`.

### Sécurité du VPS et des apps
- SSH par clé uniquement, root désactivé, pare-feu n'ouvrant que 22/80/443
  (l'API Kubernetes 6443 n'est pas exposée publiquement : accès via tunnel SSH ou VPN).
- HTTPS partout, redirection HTTP → HTTPS, en-têtes de sécurité (HSTS, CSP…).
- **Rate limiting** sur les endpoints publics d'écriture (middleware Traefik ou applicatif).
- Mises à jour régulières du système, de k3s et des images de base.

---

## 9. Décisions

### Prises (2026-10-14)

- [x] Sous-domaine du linktree : **`link.zazi.cool`** seul (pas de `links.zazi.cool`).
- [x] Sous-domaine du jeu Undercover : **`undercover.zazi.cool`**.
- [x] Apex `zazi.cool` : **page d'accueil dédiée** (`apps/home/`, statique) ;
      `www.zazi.cool` redirige en 301 vers l'apex.
- [x] Stack technique : **Go pour tous les back-ends** ; front en rendu serveur par défaut,
      **TypeScript + Vite** (embarqué dans le binaire Go) pour les fronts interactifs comme
      Undercover. Les choix détaillés (bibliothèques, temps réel) se font au cadrage de
      chaque projet.
- [x] VPS : **OVHcloud, 2 vCPU / 4 Go RAM**. Sauvegardes sur **OVHcloud Object Storage**.
- [x] Choix d'infra du §2 **validés** (k3s, Traefik, cert-manager, Kustomize, SOPS + age,
      PostgreSQL partagé).
- [x] Monitoring : **uptime externe uniquement** pour commencer.
- [x] Migrations : **exécuteur maison** (`embed` + `schema_migrations`), commun aux apps (§5).

### Prises (2026-10-14, mise en production)

- [x] VPS sous **Ubuntu**, administré avec l'utilisateur **`ubuntu`** (sudo), SSH par clé
      uniquement. Bootstrap scripté : `infra/cluster/bootstrap.sh`.
- [x] Accès `kubectl` par **tunnel SSH** (port local 16443), kubeconfig dans `~/.kube/zazi.yaml`.
- [x] Images sur ghcr.io sous **`ghcr.io/zazi2004/<app>`**, **packages publics** (le dépôt
      reste privé ; les images ne contiennent aucun secret) : pas de secret de pull.
- [x] Let's Encrypt **sans adresse e-mail** (pas d'avis d'expiration ; renouvellement automatique).
- [x] PostgreSQL : **StatefulSet** `postgres:18.6-alpine` simple (pas d'opérateur), volume
      `local-path` du nœud ; bases créées par le Job `create-databases`.
- [x] Sauvegardes : `pg_dump` par base + rôles, envoyés par **rclone** vers Object Storage,
      rétention 30 jours ; Job de test de restauration.
- [x] Traefik : `externalTrafficPolicy: Local` (IP réelle du client pour le rate limiting).
      Redirection HTTP → HTTPS **par Ingress** (Middleware `redirect-https` dans chaque
      namespace d'app) et non globale : une redirection globale empêche la validation
      HTTP-01 de Let's Encrypt.

### Ouvertes

- [ ] Choix du service d'uptime externe.
- [ ] Gamme/offre précise du VPS OVHcloud et région.

---

## 10. Instructions pour Claude

- **Lire le `CLAUDE.md` de l'application concernée** avant toute modification dans `apps/<nom>/`.
- Respecter les conventions de ce fichier ; si une convention semble inadaptée,
  **le signaler et proposer** plutôt que de la contourner silencieusement.
- **Demander avant** : d'ajouter une dépendance significative, de changer de stack,
  de modifier l'infrastructure partagée (`infra/cluster`, `infra/data`), ou toute action
  touchant à la production (déploiement, `kubectl` contre le cluster, migrations en prod).
- Ne jamais écrire de secret en clair, ni dans le code, ni dans les manifests, ni dans les docs.
- Tenir la documentation à jour : quand une décision est prise ou une spec évolue,
  mettre à jour le `CLAUDE.md` concerné (section « Décisions ouvertes » incluse).
- Livrer du code testé : lancer `make lint` et `make test` et rapporter fidèlement le résultat.
- Rester simple : pas d'abstraction, de microservice ou d'outil d'infra supplémentaire
  sans besoin concret.
- **Le développeur découvre Go** : écrire du Go idiomatique et sans astuce, et expliquer
  dans les réponses (pas dans les commentaires du code) les idiomes Go non évidents
  (gestion d'erreurs, interfaces, goroutines/channels, `context`…).
