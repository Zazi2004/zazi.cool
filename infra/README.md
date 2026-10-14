# Infrastructure — zazi.cool

Un VPS OVHcloud (Ubuntu) avec k3s mono-nœud : Traefik (inclus dans k3s), cert-manager,
PostgreSQL partagé et une application par namespace. Choix et principes : voir le
[`CLAUDE.md` racine](../CLAUDE.md), §2 et §8.

```
infra/
├── cluster/   # bootstrap du VPS, accès kubectl, Traefik, cert-manager, ClusterIssuers
├── data/      # PostgreSQL, bases par application, sauvegardes
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

Première fois :

```sh
kubectl apply -f infra/apps/undercover/base/namespace.yaml
sops decrypt infra/apps/undercover/overlays/prod/secret.enc.yaml | kubectl apply -f -
```

Puis, à chaque version :

1. Le merge sur `main` déclenche la CI, qui publie `ghcr.io/zazi2004/undercover:<sha>`
   (les 7 premiers caractères du commit).
2. Mettre ce SHA dans `newTag` de `infra/apps/undercover/overlays/prod/kustomization.yaml`
   et committer (`chore(infra): deploy undercover <sha>`).
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
