# Accueil — `zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre à la page d'accueil.

## Contexte

Page d'accueil servie sur l'apex `zazi.cool`. Elle présente brièvement le site et renvoie
vers les services hébergés (pastebin, linktree, undercover…).
`www.zazi.cool` redirige en 301 vers `zazi.cool` (middleware Traefik `redirectRegex`, sans
logique applicative).

**Statut : cadrage.** Les spécifications détaillées seront complétées lors d'une prochaine session.

## Périmètre envisagé (à confirmer)

### MVP
- Une page statique, légère, responsive, avec mode clair/sombre.
- Liste des services avec un lien vers chacun et une courte description.
- Métadonnées de partage (Open Graph, favicon).

## Choix d'architecture

- **Statique, sans base de données.** Contenu embarqué dans le binaire (`embed`), servi par
  un petit serveur **Go** en bibliothèque standard, qui expose aussi `/healthz` et `/readyz`
  comme les autres apps.
- Le `compose.yaml` local se limite à l'application.

## Décisions ouvertes

- [ ] Contenu et ton de la page (simple liste de services ou petite présentation personnelle ?).
- [ ] Identité visuelle (partagée avec le linktree ?).
