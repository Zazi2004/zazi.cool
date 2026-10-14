# Undercover / Jeu de l'imposteur — `undercover.zazi.cool`

> Les règles communes (conventions, Docker, Compose, déploiement, Git) sont dans le
> [`CLAUDE.md` racine](../../CLAUDE.md). Ce fichier ne décrit que ce qui est propre au jeu.

## Contexte

Jeu de société multijoueur en ligne, inspiré d'**Undercover** / du jeu de l'imposteur,
jouable depuis le téléphone de chaque joueur, typiquement autour d'une même table ou en appel.

**Statut : cadrage.** Les règles exactes et les spécifications seront complétées lors
d'une prochaine session.

## Principe du jeu (règles classiques d'Undercover, à confirmer/adapter)

- Chaque joueur reçoit secrètement un rôle et un mot :
  - **Civils** : reçoivent le mot A (majorité).
  - **Undercover(s)** : reçoivent le mot B, proche du mot A (ex. « chat » / « tigre »).
    Ils ne savent pas forcément qu'ils sont undercover.
  - **Mr. White** (optionnel) : ne reçoit aucun mot.
- À chaque tour, chaque joueur donne un indice sur son mot, puis tout le monde **vote**
  pour éliminer un joueur.
- Un Mr. White éliminé peut tenter de deviner le mot des civils pour gagner.
- Les civils gagnent quand tous les imposteurs sont éliminés ; les imposteurs gagnent
  s'ils survivent jusqu'à être en position de majorité / dans les derniers joueurs.

## Périmètre envisagé (à confirmer)

### MVP
- Création d'une partie → code de salon court (ex. 4 lettres) + lien / QR code à partager.
- Rejoindre une partie avec un pseudo, sans création de compte.
- L'hôte configure : nombre d'undercovers, présence de Mr. White, puis lance la partie.
- Distribution des rôles et des mots, affichage privé sur l'écran de chaque joueur.
- Déroulé des tours, vote, élimination, détection de fin de partie, écran de résultats.
- Banque de paires de mots en français.

### Pistes ultérieures
- Saisie des indices dans l'app (ou non : les indices peuvent être donnés à l'oral).
- Catégories / thèmes de mots, paires proposées par les joueurs.
- Rejouer avec les mêmes joueurs, scores cumulés.
- Autres variantes du jeu de l'imposteur.

## Points techniques

- **Temps réel** : WebSockets (ou SSE) pour synchroniser l'état de la partie.
- **Le serveur fait autorité** : rôles et mots ne sont envoyés qu'au joueur concerné, jamais
  diffusés à tous ; les votes sont validés côté serveur.
- **Reconnexion** : un joueur qui recharge la page ou perd le réseau doit retrouver sa place
  (identifiant de session stocké côté client).
- **État des parties** : éphémère (en mémoire ou Redis), avec expiration des salons inactifs.
  PostgreSQL seulement pour les données durables (banque de mots, éventuelles statistiques).
- **Kubernetes** : démarrer avec **1 réplique** (état en mémoire). Passer à plusieurs
  répliques imposerait Redis (état + pub/sub) ou des sessions persistantes (sticky sessions).
- Interface **mobile-first**, utilisable d'une main, lisible en soirée (mode sombre).

## Décisions ouvertes

- [x] Sous-domaine : **`undercover.zazi.cool`** (décidé le 2026-10-14).
- [x] Langage du back : **Go** (décision commune, voir `CLAUDE.md` racine §9).
- [ ] Règles exactes et variantes retenues (Mr. White, conditions de victoire, indices oraux ou écrits).
- [ ] Nombre de joueurs min/max.
- [ ] Source et volume de la banque de paires de mots.
- [ ] Transport temps réel (WebSocket via une dépendance type `coder/websocket`, ou SSE en
      bibliothèque standard) et approche front.
