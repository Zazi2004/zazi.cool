#!/usr/bin/env bash
# Copies the k3s kubeconfig from the VPS to ~/.kube/zazi.yaml. The API (port 6443) is not
# exposed: kubectl goes through an SSH tunnel on local port 16443 (see `tunnel.sh`).
# Usage: infra/cluster/fetch-kubeconfig.sh <vps-host>
set -euo pipefail
host="${1:?usage: $0 <vps-host>}"
out="$HOME/.kube/zazi.yaml"

mkdir -p "$HOME/.kube"
ssh "ubuntu@${host}" sudo cat /etc/rancher/k3s/k3s.yaml |
  sed -E -e 's#https://127.0.0.1:6443#https://127.0.0.1:16443#' \
    -e 's/(name|cluster|user|current-context): default$/\1: zazi/' > "$out"
chmod 600 "$out"
echo "Kubeconfig written to $out (context: zazi)."
echo "Use it with: export KUBECONFIG=$out"
