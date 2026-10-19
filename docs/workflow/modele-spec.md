# Modèle de spec

Copier ce fichier vers `docs/specs/<app>.md` (nouveau projet) ou
`docs/specs/<app>-<sujet>.md` (évolution d'un projet existant), puis remplir chaque
section. Supprimer les indications en italique.

Règles :

- **Français**, phrases courtes, pas de code applicatif (des exemples d'API ou de schéma
  suffisent).
- Ce qui est déjà décidé dans le `CLAUDE.md` racine (stack, Docker, CI, sécurité) n'est
  **pas répété** : on ne note que ce qui est propre au projet ou qui s'en écarte.
- Toute question non tranchée va dans « Décisions ouvertes », avec une **proposition par
  défaut**. Une spec peut être validée avec des décisions ouvertes, à condition qu'aucun
  ticket de la première étape n'en dépende.
- Une fois la spec validée, les règles durables (comportement, API, données) sont reportées
  dans `apps/<app>/CLAUDE.md`, qui reste la référence pour le code.

---

```markdown
# Spec — <Projet> (`<sous-domaine>.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | Brouillon / En revue / **Validée le AAAA-MM-JJ** / Livrée |
| Issue Multica  | <ID de l'issue de cadrage> |
| Application    | `apps/<app>/` |
| Auteur, valideur | <qui a rédigé> / Noah |

## 1. Contexte

_Le problème, pour qui, pourquoi maintenant. Ce qui existe déjà (dans le dépôt ou ailleurs)._

## 2. Objectifs

_3 à 5 objectifs vérifiables. Chacun doit pouvoir être constaté après livraison._

- O1 — …
- O2 — …

## 3. Périmètre

### Inclus (cette version)
- …

### Hors périmètre
_Ce qu'on choisit explicitement de ne pas faire maintenant, avec la raison si utile._
- …

## 4. Exigences

### Fonctionnelles
_Numérotées pour pouvoir être citées dans les tickets et les critères d'acceptation._

- **F1** — …
- **F2** — …

### Non fonctionnelles
_Seulement ce qui précise ou dépasse le `CLAUDE.md` racine : performance, limites,
sécurité propre au projet, accessibilité, vie privée…_

- **NF1** — …

### Interfaces
_Pages, endpoints HTTP (méthode, chemin, entrées, sorties, codes d'erreur), formats
d'échange._

| Méthode et chemin | Rôle | Réponses |
|-------------------|------|----------|
| … | … | … |

### Données
_Tables et colonnes principales, index, rétention et purge. « Aucune » si sans base._

### Configuration
_Nouvelles variables d'environnement (nom, rôle, défaut), à reporter dans `.env.example`._

## 5. Critères d'acceptation

_Formulés pour être testés, de préférence en Given / When / Then, et rattachés aux
exigences. Ils servent à la review et à la recette après déploiement._

- **CA1** (F1) — Étant donné …, quand …, alors …
- **CA2** (NF1) — …

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| … | … | … | … |

## 7. Décisions ouvertes

_Chaque décision avec une proposition par défaut, et les tickets qu'elle bloque._

- [ ] D1 — … **Proposition :** … (bloque : T…)

## 8. Découpage en tickets

_Chaque ticket devient une sous-issue Multica de l'issue de cadrage. Un ticket = une PR,
livrable et testable seul, idéalement moins d'une journée de travail. Les étapes
définissent l'ordre : une étape ne démarre que quand la précédente est terminée._

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | … | F1 | — | dev |
| T2 | 2 | … | F2, CA2 | T1 | dev |
| Tn | n | Déployer <app> en production | — | tous | déploiement |

Types : `dev` (code applicatif + tests), `infra` (manifests, CI, base),
`déploiement` (bascule du tag et vérification, voir `docs/workflow/cycle-de-vie.md`).

## 9. Recette après déploiement

_Les vérifications à faire sur la production, reprises dans le ticket de déploiement._

- [ ] `curl -sI https://<sous-domaine>.zazi.cool/healthz` → `200`
- [ ] …
```
