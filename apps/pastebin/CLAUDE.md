# Pastebin — `pb.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au pastebin.

## Contexte

Service de partage de texte et de code : on colle un contenu, on obtient une URL courte
à partager. Service personnel hébergé sur `pb.zazi.cool`.

**Statut : cadrage.** Les spécifications détaillées seront complétées lors d'une prochaine session.

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
