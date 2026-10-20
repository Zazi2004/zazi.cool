# Protection de `main` et sécurité de la chaîne CI

Objectif : **aucun chemin de déploiement ne contourne les checks**. Le déploiement
automatique ([`cycle-de-vie.md`](cycle-de-vie.md#trajectoire)) ne doit partir que de
code passé par une PR dont la CI est verte.

## Checks

| Workflow | Rôle |
|----------|------|
| `ci.yml` | Point d'entrée sur chaque PR et chaque push sur `main`. Détecte les apps modifiées, appelle leurs workflows, scanne les secrets (Gitleaks) et les dépendances/manifests (Trivy), puis le job final **`ci / required`** agrège le tout. |
| `app.yml` | Workflow **réutilisable** (`workflow_call`, entrées `app` et `node`), appelé uniquement par `ci.yml` si `apps/<app>/**` ou `app.yml` a changé et que l'app a un `Dockerfile` : lint, tests, puis (sur `main`) scan Trivy de l'image, build et push. |

`ci / required` est le **seul check requis** : il réussit si tous les jobs ont réussi
ou été ignorés (app non modifiée), et échoue sinon. Une PR qui ne touche aucune app
(docs, infra) reste donc mergeable. Une nouvelle app s'ajoute en l'appelant dans
`ci.yml` et dans `needs` de `ci / required`.

## Ruleset de `main` (à appliquer par Noah)

Settings → Rules → Rulesets, cible `main`, statut **Active**, **bypass list vide** :

- [x] Restrict deletions
- [x] Block force pushes
- [x] Require a pull request before merging (0 approbation si Noah est seul)
- [x] Require status checks to pass : `ci / required`, avec « Require branches to be up to date before merging »
- [ ] Require linear history (optionnel)

Dans Settings → General → Pull Requests : cocher **Allow auto-merge** (utile à
l'étape 4 de la trajectoire). Dans Settings → Actions → General : « Workflow
permissions » en lecture seule par défaut. Le nom du check n'apparaît dans le ruleset
qu'après un premier passage sur une PR.

## Chaîne d'approvisionnement

- **Actions épinglées par SHA** (commentaire `# vX` pour la lisibilité) ; Dependabot
  propose les mises à jour par PR.
- **Dependabot** (`.github/dependabot.yml`) : actions, modules Go, npm et images de
  base Docker, chaque semaine.
- **Trivy** : dépendances et manifests (job `ci / scan`) et image avant push
  (HIGH/CRITICAL corrigeables bloquent). Une image vulnérable n'est donc jamais publiée.
- **Gitleaks** : secrets dans le diff et l'historique.
- Permissions des workflows en `contents: read`, `packages: write` seulement pour
  le job qui pousse l'image.

## Limites

- Le push de l'image a lieu après le merge ; la protection empêche d'y arriver sans CI verte.
- Le token `GITHUB_TOKEN` ne déclenche pas de workflows sur les PR qu'il ouvre : la PR de
  bump de l'étape 2+ devra être ouverte avec une GitHub App ou un PAT pour que `ci / required` s'exécute.
