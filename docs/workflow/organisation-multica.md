# Organisation Multica

> **Validée par Noah le 2026-10-16**, pas encore mise en place : rien de ce qui suit
> n'est encore créé dans le workspace. La création des projets, agents et autopilots
> fera l'objet d'une étape suivante, avec un aperçu confirmé avant chaque création.

---

## Projets

Un projet Multica par application, plus un projet pour l'infrastructure partagée.
Chaque projet est **lié au dépôt** `Zazi2004/zazi.cool` et porte en contexte le chemin de
l'app et son `CLAUDE.md`, pour que chaque run démarre informé.

| Projet Multica | Périmètre | Contexte à lier | Premier contenu |
|----------------|-----------|-----------------|-----------------|
| **Undercover** | `apps/undercover/`, `infra/apps/undercover/` | `apps/undercover/CLAUDE.md` | Évolutions et correctifs du jeu en production |
| **Pastebin** | `apps/pastebin/`, `infra/apps/pastebin/` | `apps/pastebin/CLAUDE.md`, `docs/specs/pastebin.md` | Issue de cadrage pilote + ses sous-issues |
| **Linktree** | `apps/linktree/`, `infra/apps/linktree/` | `apps/linktree/CLAUDE.md` | Cadrage (statique ou dynamique) |
| **Accueil** | `apps/home/`, `infra/apps/home/` | `apps/home/CLAUDE.md` | Cadrage |
| **Infra** | `infra/cluster`, `infra/data`, `.github/` commun | `infra/README.md` | Uptime externe, mises à jour, tests de restauration |

Le projet **Infra** porte ce qui n'appartient à aucune app : les décisions ouvertes du
`CLAUDE.md` racine (service d'uptime, offre VPS), la maintenance et la **trajectoire vers
le déploiement automatisé** (voir [`cycle-de-vie.md`](cycle-de-vie.md#trajectoire)).

---

## Agents

Trois agents spécialisés, plus Mika comme coordinateur. Tous travaillent dans le même
dépôt et suivent le `CLAUDE.md` racine ; leurs instructions ne font que préciser leur rôle.

| Agent | Rôle | Reçoit | Livre | Ne fait jamais |
|-------|------|--------|-------|----------------|
| **Mika** (existant) | Coordination : crée les issues de cadrage, transforme le découpage d'une spec validée en sous-issues, promeut les étapes, résume l'avancement. | Une demande de Noah, une spec mergée | Issues, sous-issues, statuts | Coder |
| **Rédacteur de specs** | Cadrage d'un projet ou d'une évolution, avec le [modèle](modele-spec.md). Pose les questions, propose des choix par défaut. | Issue de cadrage | PR `docs/specs/<app>.md` | Écrire du code applicatif |
| **Développeur zazi** | Tickets de dev et d'infra applicative (Go, TypeScript, Kustomize de l'app, workflow CI de l'app) ; prépare les PR de déploiement et fait la vérification externe (étapes 1 et 2), puis outille l'automatisation (job `verify`, PR de bump, manifests Flux) dans le projet Infra. | Sous-issue `todo` | PR liée, CI verte, `make lint`/`make test` rapportés | `kubectl` sur la prod, toucher `infra/cluster`/`infra/data` sans accord |
| **Reviewer** | Relecture des PR : conformité à la spec et au `CLAUDE.md`, sécurité (entrées, XSS, secrets), tests. Commente la PR et l'issue ; ne merge pas. | Issue passée en `in_review` (sur demande) | Commentaire de review avec verdict | Merger, pousser du code |

Compétences (skills) à donner aux agents, pour les rendre réutilisables :

- **Rédacteur de specs** : le modèle de spec et ce cycle de vie (`docs/workflow/`).
- **Développeur zazi** : conventions Go du `CLAUDE.md` racine, procédure de
  `infra/README.md` §6, et la consigne « le développeur découvre Go » (expliquer les
  idiomes dans le commentaire de livraison).
- **Reviewer** : une grille de review tirée du `CLAUDE.md` racine (§5, §6, §8).

Le Reviewer apporte surtout un second regard sécurité (pastebin public, contenu non
fiable) ; il devient indispensable à l'étape 4 du déploiement, où le merge d'une PR de
fonctionnalité suffit à mettre en production.

---

## Automatisations

Par ordre d'utilité. Aucune n'est indispensable pour démarrer.

| Automatisation | Type | Effet |
|----------------|------|-------|
| **Enchaînement des étapes** | Étapes de sous-issues (`--stage`) | Quand tous les tickets d'une étape sont `done`, l'étape suivante peut être promue en `todo` (par Mika ou Noah) : pas de démarrage prématuré. |
| **Attente de la CI** | Réveil d'issue `--until-pr checks` | Un agent qui a ouvert une PR ne surveille pas la CI : l'issue est réveillée quand les checks de la PR liée sont terminés, et l'agent corrige si elle est rouge. |
| **Attente de la vérification** (étapes 1 à 3 du déploiement) | Réveil d'issue sur commentaire de Noah, puis `--until-pr checks` sur la PR de bump | Après « déployé » commenté par Noah (étape 1), ou quand le job `verify` est terminé (étapes 2 et 3), l'agent consigne le résultat et la recette. |
| **Maintenance hebdomadaire** | Autopilot planifié (lundi 9 h, Europe/Paris), projet Infra | Crée une issue « Revue hebdo » : mises à jour disponibles (Go, Node, images de base, k3s, cert-manager), branches mergées à supprimer, PR en attente. Lecture seule. |
| **Rappel de test de restauration** | Autopilot planifié (mensuel), assigné à Noah | Crée une issue avec les commandes du test de restauration (`infra/README.md` §7), qui demande l'accès au cluster. |
| **Trace des déploiements** (étape 4) | Autopilot webhook appelé par le job CI `verify` | Crée « Déployer <app> <sha7> » avec le résultat de la vérification ; en cas d'échec (rollback automatique fait), l'issue est assignée au Développeur zazi en `blocked` avec le constat. |

---

## Décisions

### Prises (2026-10-16)

- [x] **Organisation** : projets Undercover, Pastebin, Linktree, Accueil et **Infra** ;
      agents **Rédacteur de specs**, **Développeur zazi** et **Reviewer**.
- [x] **Emplacement des specs** : `docs/specs/<app>.md` pendant le cadrage, puis report des
      règles durables dans `apps/<app>/CLAUDE.md`.
- [x] **Vérification de version** : chaque app expose `GET /version` (SHA de build).
- [x] **Déploiement** : objectif d'un **déploiement 100 % automatisé à terme**, par étapes
      (voir [`cycle-de-vie.md`](cycle-de-vie.md#déploiement)) ; le flux manuel actuel reste
      l'étape 1.
- [x] **Spec pilote pastebin** : décisions tranchées (voir
      [`../specs/pastebin.md`](../specs/pastebin.md) §7), expiration par défaut 10 minutes.
- [x] **Branches** : suppression automatique des branches après merge, à activer par Noah
      dans les réglages GitHub du dépôt.

### Ouvertes

- [ ] **Outil de déploiement continu** : Flux en GitOps pull (recommandé) plutôt qu'un
      push depuis GitHub Actions ; à confirmer avant l'étape 3.
- [ ] **Clé age du cluster** : accepter une clé age dédiée stockée dans le cluster (second
      destinataire SOPS), ce qui assouplit la règle « clé age hors du VPS » ; requis par Flux.
- [ ] **Bump automatique (étape 4)** : PR de bump en auto-merge ou commit direct de la CI
      sur `main` (dépend de la protection de branche) ; clôture automatique ou non du
      ticket de trace.

### Mise en place

- Le dépôt est enregistré dans le workspace Multica (2026-10-16).
- L'intégration GitHub de Multica est installée sur le dépôt (2026-10-16) : les PR sont
  liées aux issues et les réveils `--until-pr checks` peuvent fonctionner.
