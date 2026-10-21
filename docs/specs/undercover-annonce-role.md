# Spec — Undercover : annonce du rôle de l'éliminé (`undercover.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **En revue** |
| Issue Multica  | ONBO-38 |
| Application    | `apps/undercover/` |
| Auteur, valideur | Rédacteur de specs / Noah |

## 1. Contexte

Demande : « Annoncer le rôle de la personne éliminée à chaque tour, comme c'est
actuellement fait pour Mr. White. »

Aujourd'hui, seul un **Mr. White** éliminé voit son rôle annoncé (écran
`GuessScreen` : « ⟨prénom⟩ était… Mr. White », puis saisie du mot). Pour un civil ou un
undercover, `eliminate()` (`web/src/game/engine.ts`) enchaîne directement sur la fin de
partie ou le tour suivant. Une étape de révélation existait (`RevealScreen.tsx`) ; elle a
été **retirée le 2026-10-20** (commit `6c2f8a2`), et le `CLAUDE.md` de l'app consigne la
décision « plus de révélation de rôle à l'élimination ». Cette spec **revient sur cette
décision** à la demande de Noah.

Précision de Noah (commentaire du 2026-10-21) : le rôle est annoncé **à l'élimination**, pas
au début de partie. À la fin d'un tour, le groupe vote et élimine quelqu'un ; c'est à ce
moment que son rôle est annoncé à tout le monde. **Le début de partie ne change pas** :
civils et undercovers ne connaissent pas leur rôle (ils voient seulement leur mot), seuls
les Mr. White savent qu'ils le sont.

Conséquence de jeu à connaître : les undercovers ne connaissent pas leur rôle. Annoncer
« Undercover » à l'élimination est donc le moment où le groupe (et l'intéressé) l'apprend ;
c'est aussi un indice pour la suite (il reste un imposteur de moins).

## 2. Objectifs

- **O1** — À chaque élimination (vote de fin de tour), tout le groupe voit le rôle (civil, undercover ou
  Mr. White) du joueur éliminé avant que la partie continue.
- **O2** — Le comportement de Mr. White reste identique (annonce, puis saisie du mot).
- **O3** — Le début de partie reste inchangé : aucun rôle n'est annoncé aux civils ni aux
  undercovers (seul le Mr. White se sait Mr. White).
- **O4** — Aucun mot n'est révélé avant la fin de la partie.

## 3. Périmètre

### Inclus
- Un écran d'annonce du rôle pour les civils et les undercovers éliminés.
- Mise à jour du moteur, des tests, de la page « Règles » et du `CLAUDE.md` de l'app.

### Hors périmètre
- Le début de partie : distribution des mots et écran de passage de téléphone inchangés.
- Aucun changement d'API, de base de données ni de réglage (pas d'option pour
  désactiver l'annonce).
- Révélation du **mot** de l'éliminé : réservée à la fin de partie.
- « Personne n'est éliminé » : aucune annonce.

## 4. Exigences

### Fonctionnelles

- **F1** — Après confirmation du vote, si le joueur éliminé est un **civil** ou un
  **undercover**, un écran affiche « ⟨prénom⟩ était… » suivi du rôle (`Civil` ou
  `Undercover`, libellés de `roleLabels`), avec l'avatar teinté selon le rôle et le numéro
  de tour.
- **F2** — Cet écran propose un seul bouton : « Continuer » mène à la fin de partie si une
  condition de victoire est remplie, sinon au tour suivant. L'écran apparaît **aussi** pour
  l'élimination qui termine la partie (D1).
- **F3** — Un **Mr. White** éliminé garde le parcours actuel (annonce du rôle puis saisie
  du mot, `GuessScreen`).
- **F3b** — Le début de partie est inchangé : l'écran de distribution ne mentionne jamais
  « Civil » ni « Undercover » ; le rôle n'apparaît qu'à l'élimination (F1, F3) ou en fin de
  partie.
- **F4** — Le bouton « Personne n'est éliminé » enchaîne directement sur le tour suivant,
  sans annonce.
- **F5** — L'écran d'annonce ne montre ni le mot de l'éliminé ni celui des civils.
- **F6** — La phase d'annonce est enregistrée dans l'état persisté (`localStorage`) : un
  rechargement ou une mise en veille sur cet écran le réaffiche, sans perdre l'élimination.
  Une sauvegarde antérieure au changement reste lisible.
- **F7** — La page « Règles » (`/regles`) et le `CLAUDE.md` de l'app décrivent la nouvelle
  règle (remplacent « pas de révélation de rôle à l'élimination »).

### Non fonctionnelles

- **NF1** — Accessibilité : le focus va sur le verdict à l'affichage (`useFocusOnMount`),
  le bouton est atteignable au clavier, animations limitées à `transform`/`opacity` et
  coupées par `prefers-reduced-motion`.
- **NF2** — Aucun style en ligne, aucune ressource externe (CSP stricte) ; couleurs via
  les variables CSS existantes.

### Interfaces

Aucune route ni modification d'API. Côté front, nouvelle phase du moteur :
`{ kind: "reveal"; eliminated: number }`, et un nouvel écran dans `GamePage.tsx`.

### Données

Aucune (état de partie côté navigateur uniquement).

### Configuration

Aucune.

## 5. Critères d'acceptation

- **CA1** (F1) — Étant donné une partie avec un civil à éliminer et plusieurs civils en
  vie, quand on confirme son élimination, alors l'écran « ⟨prénom⟩ était… Civil »
  s'affiche et la phase est `reveal`.
- **CA2** (F1) — Idem avec un undercover : l'écran affiche « Undercover ».
- **CA3** (F2) — Depuis `reveal`, « Continuer » démarre un nouveau tour (ordre tiré, tour +1)
  si aucune condition de victoire n'est remplie.
- **CA4** (F2) — Étant donné qu'éliminer le dernier undercover donne la victoire des
  civils, quand on confirme, alors `reveal` s'affiche d'abord, puis « Continuer » mène à
  l'écran de fin avec `winner: "civils"`.
- **CA5** (F3) — Éliminer un Mr. White mène à `guess` sans passer par `reveal` ; les tests
  existants sur les réponses (correcte, fausse, « valider quand même ») passent sans
  modification.
- **CA5b** (F3b) — Pendant la distribution, aucun écran ne contient le libellé « Civil » ou
  « Undercover » ; seul un Mr. White voit son rôle (comportement actuel, test inchangé).
- **CA6** (F4) — « Personne n'est éliminé » mène directement à `round`, aucune phase
  `reveal`.
- **CA7** (F5) — Le DOM de l'écran `reveal` ne contient ni `player.word` ni le mot des
  civils.
- **CA8** (F6) — Recharger la page en phase `reveal` réaffiche le même écran ; un état
  sauvegardé avant déploiement (sans phase `reveal`) se charge sans erreur
  (`storage.test.ts`).
- **CA9** (NF1) — Au clavier, le focus arrive sur le verdict puis « Continuer » ; avec
  `prefers-reduced-motion: reduce`, aucune animation.
- **CA10** (F7) — `/regles` et `apps/undercover/CLAUDE.md` ne disent plus que le rôle
  n'est montré qu'en fin de partie ; ils décrivent l'annonce à chaque élimination.
- **CA11** — `make test` et la CI de l'app passent (Vitest et Go).

## 6. Risques et points d'attention

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Une partie en cours lors du déploiement a une phase inconnue | Écran blanc | Faible | Pas de migration d'état nécessaire : les phases existantes restent valides ; CA8 le vérifie |
| Le jeu devient plus facile pour les civils (info donnée à chaque tour) | Équilibrage | Certaine | Voulu par la demande ; l'équilibre se juge à l'usage |
| Un joueur lit l'écran avant que le groupe ne regarde | Mineur | Moyenne | Un seul bouton, pas de secret à protéger : le rôle est public par nature |

## 7. Décisions ouvertes

- [x] D1 — Annoncer aussi le rôle quand l'élimination **termine** la partie.
      **Tranchée (déduite du commentaire du 2026-10-21 : « quand un joueur est éliminé »)** :
      oui, annonce à chaque élimination, puis écran de fin. À corriger si Noah veut
      l'inverse.
- [x] D2 — Faut-il annoncer le **mot** de l'éliminé en plus du rôle ?
      **Tranchée (Noah, 2026-10-21) :** non (F5), seulement le rôle : annoncer le mot
      dévoilerait celui des civils avant la fin.

## 8. Découpage en tickets

| # | Étape | Ticket | Exigences | Dépend de | Type |
|---|-------|--------|-----------|-----------|------|
| T1 | 1 | Undercover : annoncer le rôle de l'éliminé (moteur `reveal`, écran, tests, persistance) | F1–F6, NF1, NF2, CA1–CA9, CA11 | — | dev |
| T2 | 1 | Undercover : mettre à jour la page Règles et `CLAUDE.md` | F7, CA10 | — | dev |
| T3 | 2 | Déployer undercover en production | — | T1, T2 | déploiement |

T1 couvre aussi un test garantissant que le début de partie n'annonce aucun rôle (CA5b).
T1 et T2 touchent des fichiers différents ; si Noah préfère une seule PR, les fusionner.

## 9. Recette après déploiement

- [ ] `curl -sI https://undercover.zazi.cool/healthz` → `200`
- [ ] Début de partie : aucun rôle affiché, sauf pour un Mr. White.
- [ ] Partie à 5 joueurs : éliminer un civil → « ⟨prénom⟩ était… Civil », « Continuer »
      lance le tour suivant.
- [ ] Éliminer un undercover → « Undercover » annoncé.
- [ ] Éliminer un Mr. White → annonce puis saisie du mot, comme avant.
- [ ] « Personne n'est éliminé » → pas d'annonce.
- [ ] Recharger la page sur l'écran d'annonce → l'écran est réaffiché.
- [ ] `/regles` décrit l'annonce du rôle.
