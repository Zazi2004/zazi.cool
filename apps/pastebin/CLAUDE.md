# Pastebin — `pb.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au pastebin.

## Contexte

Service de partage de texte et de code : on colle un contenu, on obtient une URL courte
à partager. Service personnel hébergé sur `pb.zazi.cool`.

**Statut : en production** sur https://pb.zazi.cool depuis le 2026-10-19 : MVP de
`docs/specs/pastebin.md` (qui fait foi) — création, lecture, anti-abus, expiration, purge
et coloration syntaxique (ONBO-5 à ONBO-8, déploiement ONBO-9).

Le tag d'image déployé est celui de `infra/apps/pastebin/overlays/prod/kustomization.yaml`
(`3a6919e` au 2026-10-19). Le rafraîchissement visuel (ONBO-10, `style.css`) est fusionné
sur `main` mais **pas encore déployé** : il part avec le prochain changement de tag. L'état
réel du cluster n'est pas accessible aux agents ; seul Noah le vérifie (`GET /version`).

## Fonctionnalités

- `GET /` : formulaire (contenu, langage, expiration). `POST /` crée le paste et répond
  `303` vers `/<id>`.
- `GET /<id>` : vue formatée avec coloration et bouton « Copier le lien ». `GET /raw/<id>` :
  vue brute en `text/plain`.
- Identifiant : 8 caractères base62 aléatoires (`crypto/rand`).
- Expiration : 10 min (défaut), 1 h, 1 jour, 1 semaine, 1 mois ; pas de « jamais ».
- Taille maximale : `MAX_PASTE_BYTES` (512 Kio par défaut).
- Sondes et diagnostic : `GET /healthz` (liveness, sans base), `GET /readyz` (base
  jointe), `GET /version` (SHA git).

## Points d'attention

- **Abus** : création publique avec rate limiting strict et honeypot (voir « Conventions »).
- Le contenu des pastes est **non fiable** : échappement systématique à l'affichage (XSS),
  CSP stricte, vue brute en `text/plain` avec `X-Content-Type-Options: nosniff`.
- Purge des pastes expirés par l'application (spec §4, Données), et refus de servir un
  paste expiré même s'il n'est pas encore purgé.
- **Une seule réplique** : le rate limiting est en mémoire (voir « Conventions »).

## Pistes ultérieures (hors MVP)

- « Burn after reading », protection par mot de passe, lien de suppression pour l'auteur.
- Création via API / `curl` documentée (le `POST /` en formulaire fonctionne déjà avec
  `curl --data-urlencode content=…`).
- Chiffrement côté client : écarté pour le MVP (spec D2).

## Données

PostgreSQL 18. Table `pastes` (`migrations/001_create_pastes.sql`) : `id`, `content`,
`language`, `created_at`, `expires_at` (non nul : pas d'expiration « jamais »). Index sur
`expires_at`. Les migrations sont embarquées et appliquées au démarrage
(`internal/migrate`) ; `pastebin migrate` les applique puis quitte.

## Configuration

Variables d'environnement (voir `.env.example`) : `DATABASE_URL` (obligatoire), `PORT`
(`8080`), `BASE_URL` (`https://pb.zazi.cool`), `MAX_PASTE_BYTES` (`524288`),
`RATE_LIMIT_CREATE` (créations par IP et par heure), `PURGE_INTERVAL` (`15m`, max `1h`).
En production, `PORT` et `BASE_URL` sont dans `infra/apps/pastebin/base/deployment.yaml` et
`DATABASE_URL` dans le Secret SOPS `overlays/prod/secret.enc.yaml`.

## Commandes

Depuis `apps/pastebin/` (détails dans le `README.md`) : `make up` / `make down` (app +
base en conteneurs), `make dev` (base en Compose, serveur Go sur l'hôte), `make test`
(avec intégration PostgreSQL), `make lint` (gofmt + `go vet`), `make fmt`, `make build`
(image taguée avec le SHA git), `make migrate`.

## Déploiement

- Image `ghcr.io/zazi2004/pastebin:<sha7>` construite par la CI (Dockerfile multi-étapes,
  binaire statique, image distroless non-root).
- Manifests dans `infra/apps/pastebin/` : `base/` (namespace `pastebin`, Deployment à
  1 réplique, Service, Ingress Traefik `pb.zazi.cool` avec certificat Let's Encrypt,
  Middlewares de redirection HTTPS et d'en-têtes de sécurité) et `overlays/prod/` (tag
  d'image + Secret chiffré).
- Pod durci : non-root, système de fichiers en lecture seule, capacités retirées,
  `RuntimeDefault` seccomp, sondes `/readyz` et `/healthz`.
- Pour déployer : mettre à jour `newTag` dans `overlays/prod/kustomization.yaml`, puis
  `kubectl apply -k` (par Noah ; aucun agent n'accède au cluster).

## Décisions

- [x] Stack : **Go**, front en rendu serveur (décision commune, voir `CLAUDE.md` racine §9).
      Un petit module TypeScript seulement si le chiffrement côté client est retenu.
- [x] Création **publique** avec rate limiting et honeypot (spec D1).
- [x] **Pas** de chiffrement côté client pour le MVP (spec D2).
- [x] Durées : 10 min, 1 h, 1 jour, 1 semaine, 1 mois ; **défaut 10 min** ; pas de
      « jamais » (spec D3).

## Conventions de l'app

- Paquets : `cmd/pastebin` (point d'entrée, configuration, arrêt propre), `internal/paste`
  (identifiant, langages et durées autorisés), `internal/store` (accès PostgreSQL),
  `internal/server` (handlers HTTP, templates et fichiers statiques embarqués),
  `internal/purge` (purge périodique), `internal/highlight` (coloration `chroma`),
  `internal/migrate` (exécuteur de migrations), `migrations/` (SQL embarqué).
- Les durées d'expiration et les langages sont des **constantes du code**
  (`internal/paste`), pas de la configuration : ils font partie de la spec. Les clés de
  langage sont des noms de lexers `chroma` (T6).
- `created_at` et `expires_at` viennent de l'horloge de PostgreSQL (`now()`), dans la même
  requête. Une lecture filtre toujours `expires_at > now()` : un paste expiré non purgé
  donne la même 404 qu'un identifiant inconnu.
- Purge (F6, ONBO-6) : une goroutine lancée par `run` (`internal/purge`) exécute
  `DELETE FROM pastes WHERE expires_at <= now()` au démarrage puis toutes les
  `PURGE_INTERVAL` (`15m` par défaut, refusé au-delà de `1h` pour tenir F6). La borne
  `<=` est le complément exact du filtre de lecture `> now()`. Un échec est journalisé
  et retenté au tick suivant. À l'arrêt, `run` annule le contexte et attend la fin de la
  purge avant de fermer le pool.
- Sécurité : rendu uniquement par `html/template` ; CSP stricte, `nosniff`,
  `X-Robots-Tag: noindex`, `Referrer-Policy: no-referrer` et `Cache-Control: no-store` sur
  toutes les réponses (`internal/server/headers.go`) ; la vue brute ajoute
  `Content-Security-Policy: sandbox`. Aucun style ni script inline : tout passe par
  `/static/`. Le seul script est le bouton « Copier le lien ».
- Corps de `POST /` plafonné à `3 × MAX_PASTE_BYTES + 4 Kio` (l'encodage URL triple au pire
  la taille du texte non ASCII) ; le contenu décodé est ensuite comparé à `MAX_PASTE_BYTES`.
- Le contenu est stocké **octet pour octet** (les fins de ligne `\r\n` envoyées par les
  navigateurs sont conservées dans la vue brute).
- Le SHA git est injecté dans `main.version` (`-ldflags "-X main.version=…"`), via
  `--build-arg VERSION=<sha7>` du Dockerfile ; il est servi par `GET /version`.
- Coloration (F3, ONBO-8) : `internal/highlight` découpe le contenu en lignes de
  `Span{Class, Text}` avec le lexer `chroma` du langage (texte brut si le lexer est inconnu
  ou échoue). Le HTML est écrit par `html/template` (`view.html`), jamais par le
  formateur HTML de `chroma` : chaque token reste échappé. Les couleurs (styles `github` et
  `github-dark`, selon `prefers-color-scheme`) sont générées au démarrage et servies en
  fichier par `GET /static/highlight.css`, chargé **avant** `style.css` : aucun style
  inline, la CSP ne change pas. Les classes `chroma` sont préfixées `hl-` pour ne pas
  heurter celles de l'app (`.line`). Un test vérifie que chaque langage du formulaire a
  un lexer.
- Dépendances : `pgx` (pilote PostgreSQL), comme Undercover ; `chroma` (D4).
- Ports locaux par défaut : app 8080, base 5434.
- Anti-abus (NF2, ONBO-7) : `POST /` passe par `guardCreate` (`internal/server/abuse.go`,
  rate limiting) **avant toute lecture du corps**, puis par le handler de création, qui
  plafonne le corps, lit le formulaire, vérifie le honeypot puis valide.
  - Limite : `RATE_LIMIT_CREATE` créations par IP sur une **fenêtre glissante d'une heure**,
    `429` + `Retry-After` au-delà. Toute tentative compte, y compris invalide ou piégée.
  - IP du client : **dernière** entrée de `X-Forwarded-For` (celle ajoutée par Traefik ; les
    précédentes viennent du client et sont falsifiables), sinon `RemoteAddr`. Les IPv6
    sont regroupées par /64.
  - Compteurs **en mémoire seulement** (NF4) : ils repartent à zéro au redémarrage et
    imposent **une seule réplique** (sinon la limite est multipliée par le nombre de pods).
  - Honeypot : champ `website` hors écran (classe CSS `.trap`) ; rempli → `303` vers `/`,
    rien n'est créé.
  - Aucun middleware ne doit lire le formulaire avant le handler de création : le plafond
    de taille (`http.MaxBytesReader`, NF3) y est posé, et une erreur de taille avalée par
    un `PostFormValue` en amont ferait passer un paste trop gros pour un paste vide.
