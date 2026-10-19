# Pastebin — `pb.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au pastebin.

## Contexte

Service de partage de texte et de code : on colle un contenu, on obtient une URL courte
à partager. Service personnel hébergé sur `pb.zazi.cool`.

**Statut : squelette en place** (ONBO-3) ; fonctionnalités à venir selon `docs/specs/pastebin.md`.

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

PostgreSQL. Table `pastes` : `id`, `content`, `language`, `created_at`, `expires_at`
(nullable), `burn_after_read`, éventuellement `password_hash`. Index sur `expires_at`.

## Décisions ouvertes

- [x] Stack : **Go**, front en rendu serveur (décision commune, voir `CLAUDE.md` racine §9).
      Un petit module TypeScript seulement si le chiffrement côté client est retenu.
- [ ] Création publique ou authentifiée.
- [ ] Chiffrement côté client ou non.
- [ ] Durées d'expiration proposées et durée par défaut.

## Conventions de l'app

- Paquets : `cmd/pastebin` (point d'entrée, configuration, arrêt propre), `internal/server`
  (handlers HTTP), `internal/migrate` (exécuteur de migrations), `migrations/` (SQL embarqué).
- Le SHA git est injecté dans `main.version` (`-ldflags "-X main.version=…"`), via
  `--build-arg VERSION=<sha7>` du Dockerfile ; il est servi par `GET /version`.
- Dépendance : `pgx` (pilote PostgreSQL), comme Undercover.
- Ports locaux par défaut : app 8080, base 5434.
- Anti-abus (NF2, ONBO-7) dans `internal/server/abuse.go` : `POST /` passe par `guardCreate`
  (rate limiting puis honeypot) avant le handler de création.
  - Limite : `RATE_LIMIT_CREATE` créations par IP sur une **fenêtre glissante d'une heure**,
    `429` + `Retry-After` au-delà. Toute tentative compte, y compris invalide ou piégée.
  - IP du client : **dernière** entrée de `X-Forwarded-For` (celle ajoutée par Traefik ; les
    précédentes viennent du client et sont falsifiables), sinon `RemoteAddr`. Les IPv6
    sont regroupées par /64.
  - Compteurs **en mémoire seulement** (NF4) : ils repartent à zéro au redémarrage et
    imposent **une seule réplique** (sinon la limite est multipliée par le nombre de pods).
  - Honeypot : champ `website` caché ; rempli → `303` vers `/`, rien n'est créé.
  - Le plafond de taille du corps (`http.MaxBytesReader`, NF3) doit s'appliquer **avant**
    `guardCreate`, qui lit le formulaire pour le honeypot.
