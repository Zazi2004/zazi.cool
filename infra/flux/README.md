# Flux (GitOps) — étape 3 de la trajectoire

Flux tourne dans le cluster et **tire** `main` : merger la PR `newTag` d'une app la déploie,
sans `kubectl`. Choix et comparaison : [`cycle-de-vie.md`](../../docs/workflow/cycle-de-vie.md#déploiement).

```
infra/flux/clusters/prod/
├── apps.yaml          # une Kustomization Flux par app (chemin infra/apps/<app>/overlays/prod)
└── flux-system/       # généré à l'étape 2 du runbook (contrôleurs + source Git)
```

**Périmètre** : Flux ne réconcilie que `infra/apps/<app>/overlays/prod` et son propre
dossier `infra/flux/clusters/prod`. `infra/cluster` et `infra/data` restent appliqués à la
main par Noah (`kubectl apply`, Helm) ; aucune `Kustomization` ne les cible.

**Installation minimale** : seuls `source-controller` et `kustomize-controller` sont
installés (pas de Helm, d'image ni de notification controller), pour économiser la mémoire
du VPS.

Aucune commande ci-dessous n'est lancée par un agent.

## Pourquoi pas `flux bootstrap` directement

`flux bootstrap github` pousse un commit directement sur `main`, ce que le ruleset de
`main` interdit (bypass list vide, voir [`securite-ci.md`](../../docs/workflow/securite-ci.md)).
Le runbook fait donc la même chose en deux temps : les manifestes passent par une PR
relue et mergée, puis Noah les applique. Le résultat est identique à un bootstrap.

## Runbook (Noah)

Prérequis : `flux` (`brew install fluxcd/tap/flux`), `kubectl` via le tunnel
([`infra/README.md`](../README.md) §3), `gh`, `age`, la clé privée age du dépôt.
`export KUBECONFIG=~/.kube/zazi.yaml` et le tunnel ouvert.

### 1. Vérifier le cluster

```sh
flux check --pre
```

### 2. Générer les manifestes Flux dans une PR

```sh
git switch -c chore/infra-flux-install
mkdir -p infra/flux/clusters/prod/flux-system
flux install --components=source-controller,kustomize-controller \
  --export > infra/flux/clusters/prod/flux-system/gotk-components.yaml
flux create source git flux-system \
  --url=ssh://git@github.com/Zazi2004/zazi.cool --branch=main \
  --secret-ref=flux-system --interval=1m --export \
  > infra/flux/clusters/prod/flux-system/gotk-sync.yaml
flux create kustomization flux-system \
  --source=GitRepository/flux-system --path=./infra/flux/clusters/prod \
  --prune=true --interval=10m --export \
  >> infra/flux/clusters/prod/flux-system/gotk-sync.yaml
```

Ouvrir la PR `chore(infra): install flux (ONBO-16)`, attendre `ci / required`, **merger**.

### 3. Clé de déploiement GitHub (lecture seule)

```sh
ssh-keygen -t ed25519 -N '' -C flux-zazi.cool -f ./flux-deploy-key
gh repo deploy-key add ./flux-deploy-key.pub --repo Zazi2004/zazi.cool --title flux-zazi.cool
# lecture seule par défaut : ne PAS ajouter --allow-write
ssh-keyscan github.com > ./known_hosts
```

### 4. Appliquer Flux, la clé de déploiement et la clé age

```sh
git switch main && git pull
kubectl apply -f infra/flux/clusters/prod/flux-system/gotk-components.yaml
kubectl -n flux-system create secret generic flux-system \
  --from-file=identity=./flux-deploy-key --from-file=known_hosts=./known_hosts
kubectl -n flux-system create secret generic sops-age \
  --from-file=age.agekey="$HOME/Library/Application Support/sops/age/keys.txt"
kubectl apply -f infra/flux/clusters/prod/flux-system/gotk-sync.yaml
rm ./flux-deploy-key ./flux-deploy-key.pub ./known_hosts
```

La clé age est désormais lisible dans le cluster (`Secret sops-age`) : c'est l'assouplissement
de la règle « clé age hors du VPS » déjà décidé pour Flux. Variante plus stricte : générer
une clé age dédiée au cluster, l'ajouter comme destinataire dans `.sops.yaml`, lancer
`sops updatekeys` sur chaque `*.enc.yaml`, et ne mettre que celle-là dans `sops-age`.

### 5. Vérifier

```sh
flux get sources git
flux get kustomizations          # flux-system et pastebin : Ready=True
kubectl -n pastebin get deploy pastebin
```

`pastebin` est déjà déployé à la main : Flux adopte les objets existants (server-side apply)
sans redémarrer le pod si rien ne change.

### 6. Valider avec pastebin

1. Merger une PR `chore(infra): deploy pastebin <sha7>` (ouverte par la CI).
2. Dans la minute qui suit (intervalle de la source) : `flux get kustomization pastebin`
   passe à la nouvelle révision, sans `kubectl apply`. Accélérer si besoin :
   `flux reconcile kustomization pastebin --with-source`.
3. Vérifier depuis l'extérieur : `curl -s https://pb.zazi.cool/version` renvoie `<sha7>`, ou
   `gh workflow run verify.yml -f app=pastebin -f url=https://pb.zazi.cool -f sha=<sha7>`.

Critère d'acceptation : cette séquence a fonctionné sans aucun `kubectl`.

## Migrer une app (après pastebin)

Pour `undercover`, puis les suivantes, en **une PR** par app :

1. Ajouter `secret.enc.yaml` dans `resources` de `infra/apps/<app>/overlays/prod/kustomization.yaml`
   et corriger son en-tête (voir celui de pastebin).
2. Ajouter à `base/namespace.yaml` l'annotation `kustomize.toolkit.fluxcd.io/prune: disabled`.
3. Ajouter dans `infra/flux/clusters/prod/apps.yaml` un bloc `Kustomization` copié de
   pastebin (nom et `path` de l'app).
4. Après le merge, `flux get kustomization <app>` doit être `Ready`. À partir de là, ne plus
   jamais lancer `kubectl apply -k` sur cette app (il appliquerait les valeurs chiffrées).

Une nouvelle app suit la même PR, plus le namespace et le secret, désormais portés par Flux
(seule sa base de données, dans `infra/data`, reste manuelle).

## Retour arrière

| Situation | Action |
|-----------|--------|
| Mauvaise version d'une app | `git revert` de la PR `newTag`, merge : Flux réapplique l'ancienne version. |
| Urgence, tout de suite | `flux suspend kustomization <app>`, corriger à la main, puis `flux resume kustomization <app>` une fois `main` corrigé. |
| Une app ne doit plus être gérée par Flux | `flux suspend kustomization <app>`, puis retirer son bloc de `apps.yaml` en ajoutant `prune: false` d'abord (PR séparée) : sans cela, `prune: true` supprimerait ses objets au retrait du bloc. |
| Désinstaller Flux | `flux suspend kustomization --all`, puis `flux uninstall --keep-namespace` : les objets déjà déployés restent en place. |

## Dépannage

```sh
flux get all -A
flux logs --level=error
kubectl -n flux-system describe kustomization pastebin    # message d'erreur détaillé
```

- `decryption failed` : le `Secret sops-age` est absent ou ne contient pas la bonne clé.
- `authentication required` / `permission denied (publickey)` : la clé de déploiement
  n'est pas enregistrée sur le dépôt ou le `Secret flux-system` est mal construit.
- `timeout` sur `wait` : le Deployment n'est pas prêt (sondes, image introuvable) ; la
  version précédente reste en service grâce au rolling update.
