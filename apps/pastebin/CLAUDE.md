# Pastebin — `pb.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au pastebin.

## Contexte

Service de partage de texte et de code : on colle un contenu, on obtient une URL courte
à partager. Service personnel hébergé sur `pb.zazi.cool`.

**Statut : création et lecture des pastes en place** (ONBO-5) ; restent la purge (T4),
l'anti-abus (T5) et la coloration syntaxique (T6) de `docs/specs/pastebin.md`, qui fait foi.

## Périmètre envisagé (à confirmer)

### MVP
- Créer un paste (texte brut) et obtenir une URL unique, courte et non devinable
  (ex. identifiant aléatoire de 8 caractères).
- Consulter un paste : vue formatée et vue brute (`/raw/<id>`, `Content-Type: text/plain`).
- Coloration syntaxique avec choix du langage (ou détection automatique).
- Expiration configurable (10 min, 1 h, 1 jour, 1 semaine, jamais…).
- Taille maximale par paste (ex. 512 Ko, configurable).

### Pistes ultérieures
- « Burn after reading » (suppression à la première lecture).
- Protection par mot de passe, ou chiffrement côté client (la clé dans le fragment `#`
  de l'URL, le serveur ne voit jamais le contenu en clair).
- Création via API / `curl` (ex. `curl --data-binary @file https://pb.zazi.cool`).
- Lien de suppression pour l'auteur.

## Points d'attention

- **Abus** : un pastebin public attire spam et contenus illicites. Trancher entre création
  publique (avec rate limiting strict) ou réservée (authentification simple, token).
- Le contenu des pastes est **non fiable** : échapper systématiquement à l'affichage (XSS),
  CSP stricte, servir la vue brute en `text/plain` avec `X-Content-Type-Options: nosniff`.
- Purge des pastes expirés : `CronJob` Kubernetes (ou tâche planifiée applicative), et
  refus de servir un paste expiré même s'il n'est pas encore purgé.

## Données (proposition)

PostgreSQL. Table `pastes` (`migrations/001_create_pastes.sql`) : `id`, `content`,
`language`, `created_at`, `expires_at` (non nul : pas d'expiration « jamais »). Index sur
`expires_at`. `burn_after_read` ou `password_hash` viendront avec les pistes ultérieures.

## Décisions ouvertes

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
  `internal/migrate` (exécuteur de migrations), `migrations/` (SQL embarqué).
- Les durées d'expiration et les langages sont des **constantes du code**
  (`internal/paste`), pas de la configuration : ils font partie de la spec. Les clés de
  langage sont des noms de lexers `chroma` (T6).
- `created_at` et `expires_at` viennent de l'horloge de PostgreSQL (`now()`), dans la même
  requête. Une lecture filtre toujours `expires_at > now()` : un paste expiré non purgé
  donne la même 404 qu'un identifiant inconnu.
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
- Dépendance : `pgx` (pilote PostgreSQL), comme Undercover.
- Ports locaux par défaut : app 8080, base 5434.
