# Organisation Multica proposée

> **Proposition uniquement** (2026-10-16). Rien de ce qui suit n'est créé dans le
> workspace : chaque élément sera mis en place après validation de Noah.

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
| **Infra** (optionnel) | `infra/cluster`, `infra/data`, `.github/` commun | `infra/README.md` | Uptime externe, mises à jour, tests de restauration |

Le projet **Infra** n'était pas demandé, mais les décisions ouvertes du `CLAUDE.md`
racine (service d'uptime, offre VPS) et la maintenance n'appartiennent à aucune app.

---

## Agents

Trois agents spécialisés, plus Mika comme coordinateur. Tous travaillent dans le même
dépôt et suivent le `CLAUDE.md` racine ; leurs instructions ne font que préciser leur rôle.

| Agent | Rôle | Reçoit | Livre | Ne fait jamais |
|-------|------|--------|-------|----------------|
| **Mika** (existant) | Coordination : crée les issues de cadrage, transforme le découpage d'une spec validée en sous-issues, promeut les étapes, résume l'avancement. | Une demande de Noah, une spec mergée | Issues, sous-issues, statuts | Coder |
| **Rédacteur de specs** | Cadrage d'un projet ou d'une évolution, avec le [modèle](modele-spec.md). Pose les questions, propose des choix par défaut. | Issue de cadrage | PR `docs/specs/<app>.md` | Écrire du code applicatif |
| **Développeur zazi** | Tickets de dev et d'infra applicative (Go, TypeScript, Kustomize de l'app, workflow CI de l'app) ; prépare les PR de déploiement et fait la vérification externe. | Sous-issue `todo` | PR liée, CI verte, `make lint`/`make test` rapportés | `kubectl` sur la prod, toucher `infra/cluster`/`infra/data` sans accord |
| **Reviewer** | Relecture des PR : conformité à la spec et au `CLAUDE.md`, sécurité (entrées, XSS, secrets), tests. Commente la PR et l'issue ; ne merge pas. | Issue passée en `in_review` (sur demande) | Commentaire de review avec verdict | Merger, pousser du code |

Compétences (skills) à donner aux agents, pour les rendre réutilisables :

- **Rédacteur de specs** : le modèle de spec et ce cycle de vie (`docs/workflow/`).
- **Développeur zazi** : conventions Go du `CLAUDE.md` racine, procédure de
  `infra/README.md` §6, et la consigne « le développeur découvre Go » (expliquer les
  idiomes dans le commentaire de livraison).
- **Reviewer** : une grille de review tirée du `CLAUDE.md` racine (§5, §6, §8).

Le Reviewer est le plus facultatif : pour un développeur seul, Noah peut relire
lui-même. Il devient utile pour un second regard sécurité (pastebin public, contenu non
fiable).

---

## Automatisations

Par ordre d'utilité. Aucune n'est indispensable pour démarrer.

| Automatisation | Type | Effet |
|----------------|------|-------|
| **Enchaînement des étapes** | Étapes de sous-issues (`--stage`) | Quand tous les tickets d'une étape sont `done`, l'étape suivante peut être promue en `todo` (par Mika ou Noah) : pas de démarrage prématuré. |
| **Attente de la CI** | Réveil d'issue `--until-pr checks` | Un agent qui a ouvert une PR ne surveille pas la CI : l'issue est réveillée quand les checks de la PR liée sont terminés, et l'agent corrige si elle est rouge. |
| **Attente de la vérification** | Réveil d'issue sur commentaire de Noah | Après « déployé » commenté par Noah sur le ticket de déploiement, l'agent lance la recette externe. |
| **Maintenance hebdomadaire** | Autopilot planifié (lundi 9 h, Europe/Paris), projet Infra | Crée une issue « Revue hebdo » : mises à jour disponibles (Go, Node, images de base, k3s, cert-manager), branches mergées à supprimer, PR en attente. Lecture seule. |
| **Rappel de test de restauration** | Autopilot planifié (mensuel), assigné à Noah | Crée une issue avec les commandes du test de restauration (`infra/README.md` §7), qui demande l'accès au cluster. |
| **Ticket de déploiement automatique** (plus tard) | Autopilot webhook GitHub (push sur `main`) | Crée « Déployer <app> <sha7> » dès qu'une image est publiée. À n'envisager que si les déploiements deviennent fréquents. |

---

## Décisions demandées

1. **Organisation** : valider les projets (avec ou sans projet Infra) et les agents
   (avec ou sans Reviewer).
2. **Emplacement des specs** : `docs/specs/<app>.md` pendant le cadrage, puis report des
   règles durables dans `apps/<app>/CLAUDE.md` (proposé) — ou tout directement dans
   `apps/<app>/CLAUDE.md`, comme l'annonce aujourd'hui le `CLAUDE.md` racine.
3. **Vérification de version** : ajouter `GET /version` (SHA de build) aux apps pour que
   la vérification du déploiement prouve la version servie sans `kubectl`.
4. **Déploiement** : rester manuel (proposé) ou préparer un passage GitOps plus tard.
5. **Spec pilote pastebin** : trancher ses décisions ouvertes (voir
   [`../specs/pastebin.md`](../specs/pastebin.md) §7).
6. **Branches** : activer la suppression automatique des branches après merge sur GitHub.
