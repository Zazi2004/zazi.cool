# Pastebin — `pb.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au pastebin.

## Contexte

Service de partage de texte et de code : on colle un contenu, on obtient une URL courte
à partager. Service personnel hébergé sur `pb.zazi.cool`.

**Statut : création, lecture, anti-abus, expiration et purge en place** (ONBO-5, ONBO-7,
ONBO-6) ; reste la coloration syntaxique (T6) de `docs/specs/pastebin.md`, qui fait foi.

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
- Purge des pastes expirés par l'application (spec §4, Données), et refus de servir un
  paste expiré même s'il n'est pas encore purgé.

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
  `internal/purge` (purge périodique),
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
- Dépendance : `pgx` (pilote PostgreSQL), comme Undercover.
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
