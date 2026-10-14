#!/usr/bin/env bash
# Opens an SSH tunnel to the Kubernetes API: local port 16443 → VPS 127.0.0.1:6443.
# Keep it running while using kubectl. Usage: infra/cluster/tunnel.sh <vps-host>
set -euo pipefail
host="${1:?usage: $0 <vps-host>}"
echo "Tunnel to the Kubernetes API of ${host} open on 127.0.0.1:16443 (Ctrl+C to close)."
exec ssh -N -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 \
  -L 16443:127.0.0.1:6443 "ubuntu@${host}"
