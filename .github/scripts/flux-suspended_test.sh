#!/usr/bin/env bash
# Tests for flux-suspended.sh. Run: .github/scripts/flux-suspended_test.sh
# Regression: ONBO-76, home was bumped and verified while its Kustomization was suspended.
set -euo pipefail

dir="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/apps.yaml" <<'YAML'
# Header comment.
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: active
  namespace: flux-system
spec:
  interval: 10m
  sourceRef:
    kind: GitRepository
    name: suspended
---
# Suspended until the first deployment.
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: suspended
  namespace: flux-system
spec:
  suspend: true
  interval: 10m
  sourceRef:
    kind: GitRepository
    name: flux-system
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: "quoted"
spec:
  suspend: "true" # comment
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: resumed
spec:
  suspend: false
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: notflux
spec:
  suspend: true
YAML

failures=0
check() {
  local app="$1" want="$2" got=0
  "$dir/flux-suspended.sh" "$app" "$tmp/apps.yaml" 2>/dev/null || got=$?
  if [ "$got" = "$want" ]; then
    echo "ok   $app: exit $got"
  else
    echo "FAIL $app: exit $got, want $want"
    failures=$((failures + 1))
  fi
}

check suspended 0
check quoted 0
check active 1
check resumed 1
# A sourceRef name is not a metadata name.
check flux-system 2
# Only Flux Kustomizations count.
check notflux 2
check missing 2

# The real file must parse: every app the CI deploys has a Kustomization.
for app in $(ls "$dir/../../infra/apps"); do
  got=0
  "$dir/flux-suspended.sh" "$app" "$dir/../../infra/flux/clusters/prod/apps.yaml" || got=$?
  if [ "$got" = 2 ]; then
    echo "FAIL $app: no Kustomization in infra/flux/clusters/prod/apps.yaml"
    failures=$((failures + 1))
  else
    echo "ok   $app: exit $got (real apps.yaml)"
  fi
done

[ "$failures" = 0 ]
