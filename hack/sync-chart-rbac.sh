#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src_file="$repo_root/config/rbac/role.yaml"
dst_file="$repo_root/charts/openbao-entity-operator/templates/clusterrole.yaml"
helpers_dir="$repo_root/config/rbac"
helpers_file="$repo_root/charts/openbao-entity-operator/templates/rbac-helpers.yaml"

grep -q '^rules:$' "$src_file"

{
  cat <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "openbao-entity-operator.fullname" . }}-manager
  labels:
{{ include "openbao-entity-operator.labels" . | indent 4 }}
EOF
  sed -n '/^rules:$/,$p' "$src_file"
} >"$dst_file"

echo "Synced manager RBAC from $src_file to $dst_file"

: >"$helpers_file"
for helper_source in "$helpers_dir"/openbao_*_role.yaml; do
  [[ -f "$helper_source" ]] || continue
  helper_name="$(sed -n 's/^  name: //p' "$helper_source" | head -n 1)"
  [[ -n "$helper_name" ]] || { echo "No ClusterRole name found in $helper_source" >&2; exit 1; }
  if [[ -s "$helpers_file" ]]; then
    printf '%s\n' '---' >>"$helpers_file"
  fi
  {
    printf '%s\n' 'apiVersion: rbac.authorization.k8s.io/v1'
    printf '%s\n' 'kind: ClusterRole'
    printf '%s\n' 'metadata:'
    printf '%s\n' '  name: {{ include "openbao-entity-operator.fullname" . }}-'"$helper_name"
    printf '%s\n' '  labels:'
    printf '%s\n' '{{ include "openbao-entity-operator.labels" . | nindent 4 }}'
    sed -n '/^rules:$/,$p' "$helper_source"
  } >>"$helpers_file"
done

echo "Synced generated API RBAC helpers from $helpers_dir to $helpers_file"
