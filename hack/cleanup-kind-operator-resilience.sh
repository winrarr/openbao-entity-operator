#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
PERSISTENT_OPENBAO_NAMESPACE=${PERSISTENT_OPENBAO_NAMESPACE:-openbao-entity-operator-persistent}
PERSISTENT_E2E_NAMESPACE=${PERSISTENT_E2E_NAMESPACE:-openbao-entity-operator-resilience}
DELETE_TIMEOUT=${DELETE_TIMEOUT:-5m}

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

cleanup_namespace() {
  local namespace="$1"
  if ! kubectl_cmd get namespace "${namespace}" >/dev/null 2>&1; then
    return 0
  fi
  for resource in openbaogroupmemberships openbaoentityaliases openbaopolicies openbaogroups openbaoentities openbaoconnections; do
    kubectl_cmd -n "${namespace}" delete "${resource}" --all --ignore-not-found=true \
      --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
  done
  kubectl_cmd delete namespace "${namespace}" --ignore-not-found=true \
    --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
}

kubectl_cmd cluster-info >/dev/null
cleanup_namespace "${PERSISTENT_E2E_NAMESPACE}"
cleanup_namespace "${PERSISTENT_OPENBAO_NAMESPACE}"
echo "Cleaned operator resilience fixture resources"
