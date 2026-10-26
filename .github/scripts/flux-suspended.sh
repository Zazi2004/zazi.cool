#!/usr/bin/env bash
# Usage: flux-suspended.sh <app> [apps.yaml]
# Exit status: 0 if the Flux Kustomization of <app> is suspended, 1 if it is active,
# 2 if <app> has no Kustomization in the file (or on a usage error).
#
# A suspended app is not reconciled by Flux: bumping its newTag deploys nothing, so
# bump.yml and deploy-verify.yml skip it instead of running verify (and a rollback) that
# can only fail. Plain awk rather than yq so that the test runs anywhere; it relies on the
# layout of infra/flux/clusters/prod/apps.yaml: one document per app separated by `---`,
# `metadata.name` and `spec.suspend` indented by two spaces.
set -euo pipefail

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: $0 <app> [apps.yaml]" >&2
  exit 2
fi
app="$1"
file="${2:-infra/flux/clusters/prod/apps.yaml}"

state="$(awk -v app="$app" '
  function flush() {
    if (kind && name == app) { found = 1; result = (suspend == "true") ? "suspended" : "active" }
    kind = 0; name = ""; suspend = ""; section = ""
  }
  { sub(/[ \t]+#.*$/, ""); gsub(/["\047]/, "") }
  /^---/ { flush(); next }
  /^[a-zA-Z]/ { section = $1 }
  /^kind: *Kustomization$/ { kind = 1 }
  section == "metadata:" && /^  name:/ { name = $2 }
  section == "spec:" && /^  suspend:/ { suspend = $2 }
  END { flush(); print found ? result : "missing" }
' "$file")"

case "$state" in
  suspended) exit 0 ;;
  active) exit 1 ;;
  *)
    echo "no Flux Kustomization named $app in $file" >&2
    exit 2
    ;;
esac
