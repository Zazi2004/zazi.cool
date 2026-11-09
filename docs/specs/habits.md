# Spec — Habit tracker v2 (`habits.zazi.cool`)

| Champ          | Valeur |
|----------------|--------|
| Statut         | **Validée** — cadrage et décisions D7–D10 validés par Noah le 2026-11-06 |
| Version précédente | v1 (ONBO-88), déployée le 2026-10-30 ; texte dans l'historique git de ce fichier |
| Application    | `apps/habits/` |
| Prototype      | branche `proto/habits-redesign` (`apps/habits/prototype-web/`, verdict dans `VERDICT.md`) |
| Auteur, valideur | Claude / Noah |

## 1. Contexte

La v1 fonctionne (cases à cocher par jour, jours concernés, vues semaine / mois / année) mais
ne plaît pas à Noah, dans cet ordre : **l'apparence** est basique et pas assez « app mobile »
(les effets inspirés de React Bits jurent avec le reste), **le modèle** est trop pauvre (oui/non
à jours fixes seulement : impossible de suivre « 2 pilules par jour » ou « 150 séances de sport
dans l'année »), et **l'usage** ne donne pas envie d'ouvrir l'app.

La v2 est une **refonte**, pas une réécriture : l'ossature Go est conservée (authentification,
sessions, rate limiting, CSRF, migrations, `/healthz` `/readyz` `/version`, manifests, CI). Le
modèle des habitudes et toute l'interface sont refaits. Les données d'habitudes de la v1 sont
**abandonnées** (une semaine de recette, sans valeur à conserver).

Inchangé : un seul utilisateur, privé, **mobile d'abord**, `habits.zazi.cool`.

## 2. Objectifs

- **O1** — Enregistrer ce qu'on a fait aujourd'hui en quelques touchers, sans formulaire.
- **O2** — Exprimer toutes les habitudes réelles de Noah : oui/non, quantités, objectifs à la
  semaine, au mois ou à l'année.
- **O3** — Donner envie d'ouvrir l'app : progression visible, rythme, séries, célébration.
- **O4** — Une interface moderne et cohérente, où les effets React Bits sont chez eux.
- **O5** — Statistiques justes : modifier une habitude ne réécrit jamais le passé.

## 3. Périmètre

### Inclus
- Habitudes **oui/non** ou **en quantité** (cible + unité), sur une **période** (jour, semaine,
  mois, année), avec **jours autorisés** optionnels.
- Emoji, couleur (palette de 8), moment de la journée (Matin / Journée / Soir), ordre manuel.
- Écran « Aujourd'hui » (direction visuelle **A « Anneaux »** du prototype), saisie par toucher et
  appui long, glissement entre les jours, anneaux de progression, rythme, série, célébration.
- **Note journalière** en texte brut.
- Statistiques : vue d'ensemble + page par habitude.
- Front **React + TypeScript + Vite**, embarqué dans le binaire Go ; API JSON.
- App **installable** sur l'écran d'accueil (manifest + icônes).

### Hors périmètre
- Plusieurs utilisateurs ; notifications / rappels ; mode hors ligne (service worker).
- Chronomètre intégré ; type « habitude à éviter » (on la formule en positif : « journée sans
  sucre ») ; fréquence « tous les X jours » ; catégories libres ; note par habitude.
- Bilans de semaine / mois mis en avant (piste ultérieure).
- Migration des données v1.

## 4. Modèle

### Habitude

| Attribut | Règle | Versionné |
|----------|-------|-----------|
| `name` | 1 à 60 caractères, unique parmi les habitudes actives | non |
| `emoji` | un graphème | non |
| `color` | une des 8 clés de palette : `amber`, `violet`, `blue`, `red`, `green`, `teal`, `slate`, `pink` | non |
| `moment` | `morning` / `day` / `evening` | non |
| `position` | ordre manuel au sein de son moment | non |
| `unit` | vide (= oui/non si cible 1) ou 1 à 20 caractères (« pilules », « min ») | non |
| `step` | incrément d'un toucher, entier ≥ 1, défaut 1 | non |
| `period` | `day` / `week` / `month` / `year` | **oui** |
| `target` | entier, 1 à 10 000 | **oui** |
| `weekdays` | masque 7 bits (bit 0 = lundi), défaut tous les jours | **oui** |

- Une habitude est **oui/non** quand `target = 1` et `unit` vide ; sinon c'est une **quantité**.
- Les semaines vont du lundi au dimanche ; mois et années sont calendaires ; le « jour » est
  celui de `TZ_NAME` (inchangé depuis la v1).
- Exemples : complément `day` / 2 / « pilules » ; sport `year` / 150 / « séances » ;
  course `week` / 3 / « sorties » ; méditation `day` / 1 ; « pas d'écran après 22 h » `day` / 1,
  lundi → vendredi.

### Règles versionnées (période, cible, jours)

- Chaque changement de `period`, `target` ou `weekdays` crée une **version** effective à partir
  du **début de la période en cours** (selon la nouvelle période). Les versions dont
  `valid_from` est postérieur ou égal sont remplacées (une seconde modification dans la même
  période remplace donc la précédente). Les périodes passées gardent leur règle : les
  statistiques et les séries ne changent pas (O5).
- La règle d'un jour = la dernière version dont `valid_from` ≤ ce jour. Une période est évaluée
  avec la règle de son premier jour.
- Si la **période** change, la dernière période de l'ancienne règle s'arrête la veille de
  `valid_from` : elle est dite **interrompue** et ne compte ni comme réussie ni comme manquée
  
- Renommer, changer l'emoji, la couleur, le moment, l'ordre, l'unité ou le pas n'est pas
  versionné.
- Archiver (`archived_on`) retire l'habitude d'aujourd'hui et des jours suivants, sans toucher
  à l'historique ; restaurer la fait réapparaître. Pas de suppression définitive.

### Saisies

- Une saisie = (habitude, jour, `value` entier ≥ 1). `value = 0` supprime la saisie.
- On ne saisit que pour un jour **autorisé** par la règle du jour (sinon `422`).
- La cible **peut être dépassée** (3 séances un jour, 10 verres sur 8).
- Fenêtre d'édition : aujourd'hui et les **6 jours précédents** (`EDIT_WINDOW_DAYS = 7`),
  jamais le futur. Au-delà : lecture seule.

### Note journalière

Une note par jour, texte brut, 0 à 1 000 caractères, même fenêtre d'édition. Vide = supprimée.

### Calculs (Go, `internal/habit`, fonctions pures)

- **Total de période** au jour J = somme des saisies du début de la période de J jusqu'à J.
- **Progression** = min(1, total ÷ cible). **Réussie** quand total ≥ cible.
- **Rythme** (périodes semaine / mois / année seulement) : attendu = cible × jours écoulés
  (J inclus) ÷ jours de la période ; écart = total − attendu, arrondi à l'entier. Libellés :
  « objectif atteint » si réussie, sinon « dans le rythme » (écart arrondi à 0),
  « N d'avance », « N de retard ».
- **Progression du jour** (grand anneau) = moyenne des progressions des habitudes **`day`**
  autorisées ce jour-là. Les objectifs sur période n'y entrent pas (ils ont leur rythme).
- **Journée complète** = toutes les habitudes `day` autorisées ce jour-là sont réussies (un jour
  sans aucune habitude `day` n'est ni complet ni manqué).
- **Série globale** = nombre de journées complètes consécutives se terminant aujourd'hui ;
  aujourd'hui non encore complet ne casse pas la série (on compte depuis hier) ; un jour sans
  habitude `day` est sauté.
- **Série d'une habitude** = périodes réussies consécutives (jours concernés pour `day`,
  semaines, mois). La période en cours non encore réussie ne casse pas la série ; un jour non
  autorisé est sauté ; une période interrompue est sautée. **Pas de série pour `year`**.
  Meilleure série = maximum historique.
- **Taux de réussite** d'une habitude sur un intervalle = périodes réussies ÷ périodes
  **closes** (+ la période en cours si elle est déjà réussie). « — » si aucune période ne compte.

## 5. Interface

Direction visuelle : **variante A « Anneaux »** du prototype (`proto/habits-redesign`).
Thème clair conçu en premier, sombre via `prefers-color-scheme`. Police Inter, **auto-hébergée**
(pas de Google Fonts, NF4).

### Navigation
Barre d'onglets flottante en bas (verre dépoli) : **Aujourd'hui** · **Stats** · **Habitudes**.
Routes côté client : `/`, `/day/{AAAA-MM-JJ}`, `/stats`, `/stats/{id}`, `/habits`, `/login`.

### Aujourd'hui (`/` et `/day/{date}`)
- En-tête : libellé relatif (« Aujourd'hui », « Hier », « Il y a N jours »), date longue,
  pastille de **série globale** (🔥).
- Bande de la semaine : 7 jours, un mini-anneau de progression chacun ; toucher un jour l'ouvre ;
  jours futurs inactifs.
- **Grand anneau** du jour (dégradé) avec pourcentage animé et « N / M du jour ».
- Groupes **Matin / Journée / Soir** (masqués s'ils sont vides) ; dans chaque groupe, ordre
  manuel, habitudes **réussies grisées en bas** du groupe.
- Ligne d'habitude : emoji sur pastille teintée, nom, sous-titre (« 1 / 2 pilules »,
  « 132 / 150 séances cette année · 5 d'avance » en vert / rouge), bouton-anneau à droite.
  Les objectifs sur période apparaissent chaque jour autorisé, même une fois atteints.
- **Toucher** la ligne : oui/non → bascule ; quantité → + `step`. Étincelle à la couleur de
  l'habitude.
- **Appui long** (≈ 450 ms) : panneau du bas avec stepper −/+ et champ numérique pour saisir la
  valeur exacte du jour ; glisser vers le bas pour fermer.
- **Glisser** horizontalement : jour précédent / suivant (jamais au-delà d'aujourd'hui).
- Hors fenêtre d'édition : bandeau « Lecture seule », touchers inactifs.
- **Note du jour** en bas, enregistrée automatiquement (anti-rebond ~800 ms), compteur /1000.
- **Célébration** (confettis) quand la journée devient complète par une action de l'utilisateur.

### Habitudes (`/habits`)
- Liste groupée par moment, réordonnable (glisser une poignée ; flèches haut / bas comme
  alternative accessible). Section repliée « Archivées » avec « Restaurer ».
- Bouton « + » flottant → panneau du bas de création ; toucher une habitude → même panneau en
  modification : nom, emoji, couleur, moment, type (oui/non | quantité), cible, unité, pas,
  période, jours autorisés. Un changement versionné affiche « s'applique à partir du <début de
  la période> ». Bouton « Archiver ».

### Stats (`/stats`, `/stats/{id}`)
- **Vue d'ensemble** : une carte par habitude active (anneau de la période en cours, total /
  cible, rythme, série en cours).
- **Page d'habitude** : heatmap calendaire de l'année (intensité = progression du jour pour
  `day`, valeur saisie sinon ; navigation par année), série en cours et meilleure série, taux
  de réussite (30 derniers jours / année), pour `week`/`month`/`year` la liste des périodes avec
  total et réussite.

### Effets (React Bits)
- Composants React Bits copiés dans le code (ils se distribuent en copier-coller) : CountUp,
  ShinyText, SpotlightCard, ClickSpark. Pas d'aurore ni de GradientText : la variante A n'en a pas.
  Animations via **Motion**. Un composant qui exigerait une autre
  dépendance (ex. `ogl`, `gsap`) demande l'accord de Noah.
- `prefers-reduced-motion` neutralise toutes les animations non essentielles (confettis,
  étincelles, reflets).

## 6. Exigences

### Fonctionnelles
- **F1** — Créer, modifier, réordonner, archiver, restaurer une habitude (§4, §5).
- **F2** — Versionner période / cible / jours (§4) ; les autres attributs s'appliquent partout.
- **F3** — Saisir une valeur pour (habitude, jour) dans la fenêtre d'édition (toucher, appui long).
- **F4** — Écran Aujourd'hui complet (§5) pour n'importe quel jour passé (lecture seule hors
  fenêtre).
- **F5** — Note journalière.
- **F6** — Statistiques : vue d'ensemble et page d'habitude (§5), calculs du §4.
- **F7** — Authentification inchangée sur le fond (mot de passe unique, sessions 30 jours,
  rate limiting, CSRF) ; écran de connexion en React, endpoints en JSON.
- **F8** — `GET /version`, `/healthz`, `/readyz` inchangés.
- **F9** — Manifest web (`name`, `short_name`, icônes 192 / 512 / maskable, `display:
  standalone`, couleurs) et `apple-touch-icon` : l'app s'ajoute à l'écran d'accueil.

### Non fonctionnelles
- **NF1** — Le front est un build Vite **embarqué** (`embed`) : une seule image, pas de Node en
  production. Toute URL hors `/api/`, `/assets/`, `/healthz`, `/readyz`, `/version` sert
  `index.html` (routage client).
- **NF2** — CSP stricte conservée : `default-src 'none'; script-src 'self'; style-src 'self';
  img-src 'self' data:; font-src 'self'; connect-src 'self'; manifest-src 'self';
  base-uri 'none'; form-action 'self'; frame-ancestors 'none'`. React et Motion écrivent les
  styles via le CSSOM (autorisé) ; aucune balise `<style>` ni attribut `style` dans le HTML
  servi. À vérifier dès ONBO-110 ; tout assouplissement est soumis à Noah.
- **NF3** — CSRF inchangé (`Sec-Fetch-Site` / `Origin`) sur toutes les requêtes non `GET` de
  l'API ; l'API ne lit que du JSON (`Content-Type: application/json`, sinon `415`).
- **NF4** — Vie privée : aucune ressource tierce (polices et icônes servies par l'app), seul
  cookie = session.
- **NF5** — **Toute règle métier est calculée en Go** ; le front affiche ce que l'API renvoie
  (progressions, rythme, séries, éditabilité). Les mises à jour sont **optimistes** côté front et
  remplacées par la réponse du serveur (ou annulées avec un message en cas d'erreur).
- **NF6** — Mobile d'abord : 360 px minimum, cibles ≥ 44 px, pas de défilement horizontal,
  contraste AA en clair **et** en sombre, utilisable au clavier (toucher = Entrée / Espace,
  appui long = bouton « Saisir une valeur » accessible).
- **NF7** — Performance : bundle JS initial ≤ 200 Ko gzip ; `GET /api/days/{date}` en au plus
  3 requêtes SQL ; page d'habitude en au plus 3 requêtes quelle que soit l'année.
- **NF8** — Ressources : une réplique ; requests `10m` / `32Mi`, limits `200m` / `128Mi`
  (inchangé, à remesurer).
- **NF9** — Formats d'échange définis **uniquement** au §7 ; les types Go et TS restent alignés
  par des fichiers JSON de référence (golden) produits par les tests Go et vérifiés par les
  tests TS (`satisfies` sur les types).

## 7. API

JSON, sous `/api/`, session obligatoire (sinon `401 {"error":"unauthorized"}`). Erreurs :
`{"error": "<code>", "message": "<texte FR>"}` avec `400` (JSON invalide), `403` (CSRF), `404`,
`409` (nom pris), `415`, `422` (validation, jour hors fenêtre ou non autorisé), `429`.
Dates `AAAA-MM-JJ`.

| Méthode et chemin | Corps → réponse |
|-------------------|-----------------|
| `POST /api/login` | `{password}` → `204` + cookie ; `401` ; `429` |
| `POST /api/logout` | → `204` |
| `GET /api/session` | → `200 {authenticated: true}` (le front s'en sert au démarrage) |
| `GET /api/days/{date}` | → `Day` |
| `PUT /api/days/{date}/entries/{habitId}` | `{value}` → `Day` |
| `PUT /api/days/{date}/note` | `{text}` → `204` |
| `GET /api/habits` | → `{active: Habit[], archived: Habit[], ruleStarts: Record<Period, string>}` (`ruleStarts` : jour où prendrait effet, si on l'enregistrait aujourd'hui, un changement de période / cible / jours, selon la nouvelle période ; sert à annoncer « s'applique à partir du … ») |
| `POST /api/habits` | `HabitInput` → `201 Habit` |
| `PATCH /api/habits/{id}` | `HabitInput` partiel → `Habit` |
| `POST /api/habits/{id}/archive`, `/restore` | → `Habit` |
| `PUT /api/habits/order` | `{moment, ids: number[]}` → `204` |
| `GET /api/stats` | → `{habits: HabitSummary[]}` |
| `GET /api/stats/habits/{id}?year=AAAA` | → `HabitStats` |

```ts
type Moment = "morning" | "day" | "evening";
type Period = "day" | "week" | "month" | "year";
type Color = "amber" | "violet" | "blue" | "red" | "green" | "teal" | "slate" | "pink";

interface Habit {
  id: number; name: string; emoji: string; color: Color; moment: Moment; position: number;
  unit: string; step: number; period: Period; target: number;
  weekdays: number;          // masque 7 bits, bit 0 = lundi (règle en vigueur aujourd'hui)
  archivedOn: string | null;
}
type HabitInput = Omit<Habit, "id" | "position" | "archivedOn">;

interface Pace { expected: number; diff: number; label: string }   // label calculé en Go

interface DayHabit {
  habit: Habit;              // règle en vigueur ce jour-là
  value: number;             // saisie du jour
  total: number;             // total de la période jusqu'à ce jour
  progress: number;          // 0..1
  complete: boolean;
  pace: Pace | null;         // null pour period = "day"
}

interface Day {
  date: string; editable: boolean;
  progress: number;          // grand anneau
  done: number; total: number;   // habitudes `day` réussies / dues (« N / M du jour »)
  complete: boolean;
  streak: number;            // série globale
  week: { date: string; progress: number }[];   // bande des 7 jours (lundi → dimanche)
  groups: { moment: Moment; habits: DayHabit[] }[];   // ordre final (réussies en bas)
  note: string;
}

interface HabitSummary {
  habit: Habit; total: number; progress: number; complete: boolean;
  pace: Pace | null; streak: number | null;   // null pour period = "year"
}

interface HabitStats extends HabitSummary {
  year: number; bestStreak: number | null;
  rate30: number | null; rateYear: number | null;   // null = « — »
  days: { date: string; value: number; level: 0 | 1 | 2 | 3 | 4 | null }[]; // null = non autorisé
  periods: { start: string; end: string; total: number; target: number;
             status: "success" | "missed" | "current" | "interrupted" }[];  // vide pour "day"
}
```

## 8. Données

Base `habits` existante. Migration **`003_v2.sql`** : supprime `completions`,
`habit_schedules`, `habits` (données v1 abandonnées), crée :

| Table | Colonnes principales | Note |
|-------|----------------------|------|
| `habits` | `id` bigserial pk, `name`, `emoji`, `color`, `moment`, `position` int, `unit`, `step` int, `created_on` date, `archived_on` date null | nom unique parmi les actives (index partiel) ; `check` sur `color`, `moment`, `step ≥ 1` |
| `habit_rules` | `habit_id`, `valid_from` date, `period`, `target` int, `weekdays` smallint | pk (`habit_id`, `valid_from`) ; jamais réécrite pour le passé |
| `entries` | `habit_id`, `day` date, `value` int > 0, `updated_at` | pk (`habit_id`, `day`) |
| `day_notes` | `day` date pk, `text` (≤ 1 000), `updated_at` | |
| `sessions` | inchangée | |

## 9. Configuration

Inchangée, sauf `EDIT_WINDOW_DAYS` dont le défaut passe à **`7`** (code, `.env.example`) et qui est
posé explicitement à `7` dans l'overlay de production (`infra/apps/habits/overlays/prod`).

## 10. Critères d'acceptation

- **CA1** (§4) — Une habitude « Vitamine D » `day` / 2 / pilules : deux touchers → 2 / 2,
  réussie, grisée en bas de « Matin » ; un troisième → 3 / 2, toujours réussie.
- **CA2** (§4) — « Sport » `year` / 150 avec 128 saisies au 6 novembre (310ᵉ jour d'une année de
  365) : attendu 127,4, `pace.label` = « 1 d'avance ».
- **CA3** (§4) — « Course » `week` / 3 : la série compte les semaines réussies consécutives ;
  la semaine en cours à 1 / 3 ne la casse pas ; une semaine passée à 2 / 3 la remet à 0.
- **CA4** (F2, O5) — Passer « Course » de 3 à 4 par semaine un jeudi : la semaine en cours est
  évaluée sur 4, les semaines passées restent sur 3 (statuts et série inchangés).
- **CA5** (D8) — Passer une habitude de `week` à `month` le 6 novembre : la version démarre le
  1ᵉʳ novembre, la semaine du 26 au 31 octobre (ancienne règle, tronquée la veille) est
  « interrupted » et n'entre ni dans la série ni dans le taux.
- **CA6** (§4) — Une habitude lundi → vendredi n'apparaît pas un samedi ; une saisie pour un
  samedi renvoie `422`.
- **CA7** (§4) — Saisir aujourd'hui − 6 jours fonctionne ; aujourd'hui − 7 jours ou demain
  renvoie `422` et ne modifie rien ; `Day.editable` vaut `false` pour aujourd'hui − 7.
- **CA8** (§4) — Série globale : 12 journées complètes jusqu'à hier et aujourd'hui incomplet →
  `streak = 12` ; aujourd'hui complet → 13.
- **CA9** (§5) — Appui long sur « Boire de l'eau », saisie 6, « Enregistrer » → `value = 6` ;
  `value = 0` supprime la saisie.
- **CA10** (F5) — La note du jour persiste au rechargement ; 1 001 caractères → `422`.
- **CA11** (§5) — La journée devient complète par un toucher → confettis une fois ; ouvrir un
  jour déjà complet n'en déclenche pas ; avec `prefers-reduced-motion`, aucun confetti.
- **CA12** (NF9) — Les fichiers golden de l'API (un par type du §7) sont générés par les tests
  Go et compilent contre les types TS ; un champ renommé d'un seul côté fait échouer `make test`.
- **CA13** (NF2) — Les en-têtes CSP du §6 sont servis et la console du navigateur ne signale
  aucune violation sur les écrans Aujourd'hui, Habitudes, Stats et Connexion.
- **CA14** (NF6) — À 360 px, aucun défilement horizontal ; cibles ≥ 44 px ; contraste AA en
  clair et en sombre (revue manuelle sur téléphone).
- **CA15** (F9) — Sur iOS (Safari) et Android (Chrome), « Ajouter à l'écran d'accueil » donne
  une icône propre et une app sans barre d'adresse.
- **CA16** (production) — Après déploiement : `/healthz`, `/readyz` → `200`, `/version` → SHA
  déployé ; les anciennes tables v1 n'existent plus ; une session ouverte avant la bascule reste
  valide.

## 11. Risques

| Risque | Impact | Probabilité | Parade |
|--------|--------|-------------|--------|
| Règles de période / versionnement mal comprises | Moyen | Moyenne | §4 explicite, fonctions pures testées en table (CA2–CA8), ticket dédié avant tout écran |
| CSP incompatible avec un composant (style inline) | Moyen | Moyenne | Vérifié dès ONBO-110 (CA13) ; on écarte le composant plutôt que d'assouplir la CSP sans accord |
| Bundle trop lourd (React + Motion + effets) | Faible | Moyenne | NF7, import ciblé de Motion, pas de bibliothèque d'icônes complète |
| Dérive des types Go / TS | Moyen | Moyenne | Golden files (NF9, CA12) |
| Gestes en conflit (glisser / appui long / défilement) | Moyen | Moyenne | Déjà résolu dans le prototype (mouvement > 10 px annule l'appui) ; test manuel sur téléphone |
| Perte des données v1 | Faible | Certaine | Acceptée par Noah (données de recette) |

## 12. Décisions

### Prises (2026-11-06, cadrage par questions)
- Refonte (ossature Go conservée), données v1 abandonnées, un seul utilisateur, mobile d'abord.
- Modèle période + cible + jours autorisés, oui/non ou quantité ; rythme au prorata ; série en
  périodes consécutives, pas de série pour `year` ; versionnement à partir de la période en cours.
- Habitudes à éviter formulées en positif ; moments fixes Matin / Journée / Soir ; emoji +
  couleur de palette ; note journalière ; fenêtre d'édition de 7 jours.
- React + TypeScript + Vite + Motion (dépendances approuvées par Noah), API JSON, golden files.
- Leviers de motivation : anneaux, célébration, série et rythme visibles. Pas de notifications.
- Direction visuelle **A « Anneaux »** ; thème clair d'abord ; trois onglets ; app installable.

### Prises (2026-11-06, relecture de la spec)
- [x] **D7** — Pas d'incrément par habitude (`step`, défaut 1).
- [x] **D8** — Changer la **période** rend « interrompue » la période en cours de l'ancienne
  règle (ni réussie ni manquée).
- [x] **D9** — Composants React Bits limités à ceux qui ne demandent que React + Motion.
- [x] **D10** — Bascule en un bloc à la fin (ONBO-118) ; la v1 reste en ligne jusque-là.

### Ouvertes
Aucune.

## 13. Découpage en tickets

Issue parente **ONBO-109** (projet Multica Habits). Tranches verticales (schéma → API → écran →
tests), toutes mergées dans la **branche d'intégration `feat/habits-redesign`** ; seule ONBO-118
la merge dans `main` (habits est géré par Flux et la v2 se déploie d'un bloc, D10).

| Étape | Issue | Tranche | Bloquée par |
|-------|-------|---------|-------------|
| 1 | ONBO-110 | Socle SPA et connexion (React embarqué, build / CI Go + TS, CSP, auth JSON, écran vide + onglets) | — |
| 2 | ONBO-111 | Créer et lister des habitudes (migration 003, onglet Habitudes, golden files) | 110 |
| 3 | ONBO-112 | Cocher les habitudes du jour (Aujourd'hui, toucher, grand anneau ; période `day`) | 111 |
| 4 | ONBO-113 | Naviguer dans les jours, saisie exacte, série, célébration et note | 112 |
| 4 | ONBO-114 | Objectifs sur une période (semaine / mois / année, rythme, jours autorisés) | 112 |
| 5 | ONBO-115 | Modifier, réordonner, archiver (règles versionnées, période interrompue) | 111, 114 |
| 6 | ONBO-116 | Statistiques : vue d'ensemble et page d'habitude | 114, 115 |
| 7 | ONBO-117 | Effets React Bits, finitions (sombre, clavier, 360 px) et app installable | 113, 116 |
| 8 | ONBO-118 | **Déployer habits v2 `<sha7>`** (merge dans `main`, recette §14) | toutes |

## 14. Recette après déploiement

- [ ] `curl -sI https://habits.zazi.cool/healthz` → `200` ; `/readyz` → `200` ;
      `curl -s https://habits.zazi.cool/version` → SHA déployé.
- [ ] La session ouverte avant la bascule est toujours valide.
- [ ] Depuis le téléphone : installer l'app sur l'écran d'accueil ; créer les habitudes réelles
      (complément 2 / jour, sport 150 / an, course 3 / semaine, une habitude lun → ven).
- [ ] Toucher, appui long, glisser entre les jours, note ; compléter la journée → confettis.
- [ ] Modifier la cible de « Course » et vérifier que la semaine passée n'a pas changé.
- [ ] Vue d'ensemble et page d'habitude cohérentes avec les saisies.
- [ ] Console sans violation CSP.
