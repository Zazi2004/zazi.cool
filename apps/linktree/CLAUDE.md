# Linktree — `link.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au linktree.

## Contexte

Page personnelle de liens, à la manière de Linktree : une page unique, soignée et
responsive (usage principalement mobile), qui regroupe mes liens (réseaux sociaux,
projets, contact…).

**Statut : cadrage.** Les spécifications détaillées seront complétées lors d'une prochaine session.

## Périmètre envisagé (à confirmer)

### MVP
- Page publique : photo/avatar, nom, courte bio, liste ordonnée de liens (titre, URL, icône).
- Design mobile-first, rapide (page légère, score Lighthouse élevé), mode clair/sombre.
- Métadonnées de partage (Open Graph, favicon).

### Pistes ultérieures
- Interface d'administration protégée pour gérer les liens sans redéployer.
- Statistiques de clics simples et respectueuses de la vie privée (sans cookie ni tracker tiers).
- Liens programmés ou temporairement masqués.

## Choix d'architecture à trancher

Deux approches, de la plus simple à la plus riche :

1. **Site statique** généré depuis un fichier de configuration (YAML/JSON) versionné dans
   le dépôt, servi par un conteneur web minimal. Pas de base de données, modification
   = commit + déploiement. Recommandé pour démarrer.
2. **Application dynamique** avec base PostgreSQL et interface d'administration.
   Nécessaire seulement si l'admin en ligne ou les statistiques deviennent prioritaires.

Dans le cas 1, le `compose.yaml` local se limite au serveur web.

## Décisions ouvertes

- [x] Sous-domaine : **`link.zazi.cool`** seul (décidé le 2026-10-14).
- [x] Stack : **Go** (décision commune, voir `CLAUDE.md` racine §9).
- [ ] Statique ou dynamique.
- [ ] Liste initiale des liens et identité visuelle.
