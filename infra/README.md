# Infrastructure — zazi.cool

Un VPS OVHcloud (Ubuntu) avec k3s mono-nœud : Traefik (inclus dans k3s), cert-manager,
PostgreSQL partagé et une application par namespace. Choix et principes : voir le
[`CLAUDE.md` racine](../CLAUDE.md), §2 et §8.

```
infra/
├── cluster/   # bootstrap du VPS, accès kubectl, Traefik, cert-manager, ClusterIssuers
├── data/      # PostgreSQL, bases par application, sauvegardes
├── flux/      # GitOps : Flux réconcilie infra/apps/*/overlays/prod (voir flux/README.md)
└── apps/<nom>/{base,overlays/prod}/
```

Dans les commandes ci-dessous, `VPS` désigne l'IP (ou le nom) du VPS :
`export VPS=<ip-du-vps>`.

## Prérequis sur le poste

`kubectl`, `helm`, `sops`, `age` (`brew install helm sops age`), une clé SSH
(`~/.ssh/id_ed25519`) et la clé privée age dans
`~/Library/Application Support/sops/age/keys.txt` (macOS). **La clé age est conservée hors
du dépôt et du VPS** et sauvegardée dans un gestionnaire de mots de passe : sans elle,
les secrets du dépôt sont perdus.

## 1. Préparer le VPS (une fois)

```sh
ssh-copy-id -i ~/.ssh/id_ed25519.pub ubuntu@$VPS     # demande le mot de passe une dernière fois
scp infra/cluster/bootstrap.sh ubuntu@$VPS:
ssh ubuntu@$VPS 'sudo bash bootstrap.sh'
```

Le script met le système à jour, active les mises à jour de sécurité automatiques,
n'autorise plus que les clés SSH (root désactivé), n'ouvre que les ports 22/80/443 et
installe k3s. Vérifier ensuite que le mot de passe est bien refusé :

```sh
ssh -o PubkeyAuthentication=no ubuntu@$VPS    # doit répondre « Permission denied »
```

## 2. DNS (une fois)

Chez le registrar de `zazi.cool` : un enregistrement `A` pour `zazi.cool` et un pour
`*.zazi.cool` vers l'IP du VPS (et `AAAA` si le VPS a une IPv6). Vérifier :
`dig +short undercover.zazi.cool`.

## 3. Accès au cluster

L'API Kubernetes (6443) n'est pas exposée : `kubectl` passe par un tunnel SSH.

```sh
infra/cluster/fetch-kubeconfig.sh $VPS        # écrit ~/.kube/zazi.yaml (une fois)
infra/cluster/tunnel.sh $VPS                  # dans un terminal dédié, à laisser ouvert
export KUBECONFIG=~/.kube/zazi.yaml           # dans le terminal de travail
kubectl get nodes
```

## 4. Composants du cluster (une fois, puis à chaque mise à jour)

```sh
helm repo add jetstack https://charts.jetstack.io && helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --version v1.21.2 --values infra/cluster/cert-manager-values.yaml --wait
kubectl apply -k infra/cluster                # Traefik (IP client) + ClusterIssuers
```

## 5. PostgreSQL

```sh
kubectl apply -f infra/data/namespace.yaml
for f in infra/data/secrets/*.enc.yaml; do sops decrypt "$f" | kubectl apply -f -; done
kubectl apply -k infra/data
kubectl -n data rollout status statefulset/postgres
kubectl -n data wait --for=condition=complete job/create-databases --timeout=120s
```

Le Job `create-databases` crée une base et un utilisateur par application. Pour une
nouvelle application : ajouter son mot de passe dans
`infra/data/secrets/app-db-passwords.enc.yaml` (`sops edit …`), une ligne
`create_app_db` dans `create-databases.sh`, puis réappliquer (le Job terminé s'efface au
bout de 10 minutes ; sinon `kubectl -n data delete job create-databases`).

## 6. Déployer une application (ex. undercover)

> **Pastebin est géré par Flux** dès que le runbook [`infra/flux/README.md`](flux/README.md)
> est exécuté : plus de `kubectl apply -k` pour lui, merger la PR `newTag` suffit. Les
> étapes ci-dessous valent pour les apps pas encore migrées.

Première fois :

```sh
kubectl apply -f infra/apps/undercover/base/namespace.yaml
sops decrypt infra/apps/undercover/overlays/prod/secret.enc.yaml | kubectl apply -f -
```

Puis, à chaque version :

1. Le merge sur `main` déclenche la CI, qui publie `ghcr.io/zazi2004/undercover:<sha>`
   (les 7 premiers caractères du commit).
2. Le job `bump` de la CI ouvre la PR `chore(infra): deploy undercover <sha>` qui met ce SHA
   dans `newTag` de `infra/apps/undercover/overlays/prod/kustomization.yaml` (une seule PR
   ouverte par app, mise à jour par un nouveau merge). La relire et la merger.
3. Appliquer et vérifier :

```sh
kubectl apply -k infra/apps/undercover/overlays/prod
kubectl -n undercover rollout status deployment/undercover
curl -sI https://undercover.zazi.cool/healthz
```

Revenir en arrière : remettre l'ancien SHA dans `newTag` et réappliquer.

Chaque Ingress d'application référence deux Middlewares Traefik de son namespace :
`redirect-https` (HTTP → HTTPS) et `security-headers` (HSTS) ; voir
`infra/apps/undercover/base/middleware.yaml`.

### Première fois pour pastebin (`pb.zazi.cool`)

Le secret n'est pas versionné tant que Noah ne l'a pas chiffré. Dans l'ordre :

```sh
# 1. Ajouter la clé `pastebin` (mot de passe aléatoire) dans le secret des bases
sops edit infra/data/secrets/app-db-passwords.enc.yaml
# 2. Créer la base : appliquer les secrets puis infra/data (voir §5)
# 3. Chiffrer le secret de l'app, avec le même mot de passe dans DATABASE_URL
cp infra/apps/pastebin/overlays/prod/secret.example.yaml infra/apps/pastebin/overlays/prod/secret.enc.yaml
sops encrypt --in-place infra/apps/pastebin/overlays/prod/secret.enc.yaml
# 4. Namespace et secret, puis le déploiement comme pour undercover
kubectl apply -f infra/apps/pastebin/base/namespace.yaml
sops decrypt infra/apps/pastebin/overlays/prod/secret.enc.yaml | kubectl apply -f -
```

## Vérifier un déploiement

Le workflow réutilisable `.github/workflows/verify.yml` contrôle une app depuis l'extérieur.
Il attend (polling toutes les 5 s, borné par `timeout`) que `GET /version` renvoie le SHA
attendu, puis vérifie `/healthz`, `/readyz`, la redirection HTTP → HTTPS et la validité du
certificat TLS (chaîne, nom, expiration dans plus de 7 jours). Chaque contrôle est une étape
distincte : l'étape rouge désigne la panne, et le job échoue (code non nul).

| Entrée | Rôle |
|--------|------|
| `app` | nom de l'app (affichage) |
| `url` | URL publique, sans chemin : `https://pb.zazi.cool` |
| `sha` | SHA attendu (7 caractères ou plus ; seuls les 7 premiers sont comparés) |
| `timeout` | attente maximale de `/version`, en secondes (défaut 300, max 840) |

À la main :

```sh
gh workflow run verify.yml -f app=pastebin -f url=https://pb.zazi.cool -f sha=<sha7>
gh run watch
```

Depuis un autre workflow :

```yaml
jobs:
  verify:
    uses: ./.github/workflows/verify.yml
    with:
      app: pastebin
      url: https://pb.zazi.cool
      sha: ${{ needs.image.outputs.sha }}
```

### Automatiquement après un merge

Le workflow `.github/workflows/deploy-verify.yml` se lance sur chaque push sur `main` qui
modifie `infra/apps/*/overlays/prod/kustomization.yaml`. Pour chaque app dont `newTag` a
changé (bump ou `git revert` d'un bump), il déduit le SHA du diff et l'URL du premier
`host` de `infra/apps/<app>/base/ingress.yaml`, crée un déploiement GitHub dans
l'environnement `production`, appelle `verify`, puis passe le déploiement à `success` ou
`failure`. Le résultat est visible dans Actions (`deploy-verify`) et dans Deployments.
Il ne déploie rien : c'est Flux qui réconcilie ; `verify` attend `timeout` (300 s) que
`/version` change.

Un SHA erroné donne un job rouge à l'étape « Wait for /version… » après le `timeout`.

### Rollback automatique

Si `verify` échoue, le job `rollback` de `deploy-verify` :

1. ouvre l'issue `Deploy blocked: <app> <sha> failed verify` (label `deploy-blocked:<app>`) ;
2. ouvre la PR `revert(infra): roll back <app> to <ancien sha> (<sha> failed verify)` qui
   remet l'ancien `newTag`, et active son auto-merge ;
3. après le merge, Flux réapplique l'ancienne version et `deploy-verify` la revérifie.

Garde-fous : un seul rollback par bump (branche `revert/infra-deploy-<app>-<sha>`) ; si la
vérification échoue aussi après le rollback, rien de plus n'est tenté et le job est rouge ;
si `main` a déjà un autre `newTag`, le bump est considéré comme remplacé et rien n'est fait.
Tant que l'issue `deploy-blocked:<app>` est ouverte, `bump.yml` n'ouvre plus de PR de bump
pour cette app : **la fermer** une fois le problème corrigé.

Prérequis : le secret `BUMP_TOKEN` (voir ci-dessous), sinon la PR de rollback n'a pas de
checks et ne peut pas être mergée ; l'option « Allow auto-merge » du dépôt, sinon la PR est
laissée à merger à la main.

Test de bout en bout (en production, protégé par les `readinessProbe`) : merger une PR de bump
dont `newTag` est un SHA sans image (par exemple `0000000`). Les nouveaux pods ne démarrent
pas, l'ancienne version reste en service, `verify` échoue après 300 s, puis la PR de rollback
apparaît ; vérifier qu'après son merge `/version` renvoie l'ancien SHA et que `verify` est vert.

## 7. Sauvegardes

Prérequis : un bucket Object Storage OVHcloud (S3) et un utilisateur S3 avec ses clés.

```sh
cp infra/data/backup/backup-s3.example.yaml infra/data/backup/backup-s3.enc.yaml
# remplir les valeurs, puis chiffrer avant tout commit :
sops encrypt --in-place infra/data/backup/backup-s3.enc.yaml
sops decrypt infra/data/backup/backup-s3.enc.yaml | kubectl apply -f -
kubectl apply -k infra/data/backup
```

La sauvegarde tourne chaque nuit à 3 h (heure de Paris) ; celles de plus de 30 jours sont
supprimées. Lancer une sauvegarde tout de suite :
`kubectl -n data create job --from=cronjob/postgres-backup backup-manual`.

**Test de restauration** (après la première sauvegarde, puis régulièrement) : restaure la
dernière sauvegarde dans une base jetable et affiche le nombre de lignes.

```sh
kubectl create -f infra/data/backup/restore-test-job.yaml    # affiche job.batch/restore-test-xxxxx
kubectl -n data logs -f job/restore-test-xxxxx -c restore      # attendre « RESTORE TEST OK »
```

## Dépannage

```sh
kubectl get pods -A
kubectl -n undercover logs deploy/undercover
kubectl -n undercover describe certificate undercover-tls      # certificat Let's Encrypt
kubectl -n cert-manager logs deploy/cert-manager
ssh ubuntu@$VPS 'sudo journalctl -u k3s --since "10 min ago"'
```

## PR de bump automatique

Le workflow `.github/workflows/bump.yml` est appelé par chaque workflow d'app après le push
de l'image. Réglages à faire une fois par Noah :

1. Dépôt → Settings → Actions → General → « Workflow permissions » : cocher
   **Allow GitHub Actions to create and approve pull requests**.
2. Les pushs faits avec le `GITHUB_TOKEN` ne déclenchent pas de workflows : la PR de bump
   n'aurait alors aucun check, et `ci / required` resterait en attente. Créer un PAT
   fin (ou une GitHub App) avec `contents` et `pull requests` en écriture sur ce dépôt, et
   l'enregistrer comme secret de dépôt **`BUMP_TOKEN`**. Sans ce secret, le workflow
   retombe sur `GITHUB_TOKEN` (PR créée, mais sans checks).

### Auto-merge (étape 4)

Décision : **auto-merge de la PR**, pas de commit direct de la CI. Le ruleset de `main`
exige une PR et `ci / required` (voir `docs/workflow/securite-ci.md`) ; l'auto-merge
respecte ces règles sans bypass, alors qu'un commit direct demanderait d'ajouter la CI à la
liste de contournement.

Après avoir ouvert ou mis à jour la PR, `bump.yml` vérifie le garde-fou puis exécute
`gh pr merge --auto --squash` : GitHub fusionne dès que `ci / required` est vert. Garde-fou,
sans quoi le job échoue et désactive l'auto-merge :

- la PR ne touche que `infra/apps/<app>/overlays/prod/kustomization.yaml` ;
- le diff se limite à la ligne `newTag` (une ligne retirée, une ajoutée, un SHA à 7 caractères).

Le merge déclenche `deploy-verify` (donc Flux puis `verify`) sans autre action humaine.

Réglages à faire par Noah, en plus des deux ci-dessus :

1. Settings → General → Pull Requests : cocher **Allow auto-merge**.
2. Ruleset de `main` appliqué (`docs/workflow/securite-ci.md`), avec `ci / required`
   requis : sans check requis, l'auto-merge fusionnerait immédiatement.
3. Le secret **`BUMP_TOKEN`** devient obligatoire : un merge fait avec le `GITHUB_TOKEN` ne
   déclenche ni `deploy-verify` ni les checks. Sans lui, le workflow n'active pas l'auto-merge
   et émet un avertissement.
