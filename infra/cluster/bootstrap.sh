#!/usr/bin/env bash
# Prepares a fresh Ubuntu VPS: system updates, SSH hardening, firewall, k3s.
# Run it on the VPS, as the `ubuntu` user, AFTER copying your SSH key (see infra/README.md):
#   sudo bash bootstrap.sh
# It is idempotent: running it again only re-applies the configuration.
set -euo pipefail

SSH_USER="${SUDO_USER:-ubuntu}"
# Empty means the current release of k3s' stable channel; set it to pin a version.
K3S_VERSION="${K3S_VERSION:-}"

if [[ $EUID -ne 0 ]]; then
  echo "Run with sudo." >&2
  exit 1
fi

# Disabling password login without a working key would lock us out.
keys="/home/${SSH_USER}/.ssh/authorized_keys"
if [[ ! -s "$keys" ]]; then
  echo "No SSH key in $keys: copy yours first (ssh-copy-id). Aborting." >&2
  exit 1
fi

echo "==> System updates and automatic security updates"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get upgrade -yq
apt-get install -yq ufw unattended-upgrades curl
dpkg-reconfigure -f noninteractive unattended-upgrades

echo "==> SSH: keys only, no root login"
# sshd keeps the first value it reads; 00- sorts before cloud-init's 50- file.
cat > /etc/ssh/sshd_config.d/00-zazi.conf <<'CONF'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
CONF
sshd -t
systemctl reload ssh

echo "==> Firewall: only 22, 80 and 443 are open"
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
# k3s pod and service networks must be able to reach the node (DNS, API server…).
ufw allow from 10.42.0.0/16
ufw allow from 10.43.0.0/16
ufw --force enable

echo "==> k3s"
if ! command -v k3s >/dev/null; then
  # --secrets-encryption: Kubernetes Secrets are encrypted at rest in the datastore.
  curl -sfL https://get.k3s.io |
    INSTALL_K3S_VERSION="$K3S_VERSION" sh -s - server \
      --secrets-encryption \
      --write-kubeconfig-mode 600
fi
k3s --version | head -1

echo "==> Waiting for the node to be ready"
for _ in $(seq 60); do
  if k3s kubectl get nodes 2>/dev/null | grep -q " Ready"; then
    k3s kubectl get nodes
    echo "Done. Next step: infra/README.md, section « Accès au cluster »."
    exit 0
  fi
  sleep 5
done
echo "The node is not ready after 5 minutes: check 'sudo journalctl -u k3s'." >&2
exit 1
