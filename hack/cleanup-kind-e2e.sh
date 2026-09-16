#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
TEST_NAMESPACE=${TEST_NAMESPACE:-openbao-entity-operator-e2e}
DELETE_TIMEOUT=${DELETE_TIMEOUT:-5m}

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

kubectl_cmd cluster-info >/dev/null
if ! kubectl_cmd get namespace "${TEST_NAMESPACE}" >/dev/null 2>&1; then
  echo "Namespace ${TEST_NAMESPACE} does not exist; cleanup is complete"
  exit 0
fi

for resource in openbaogroupmemberships openbaoentityaliases openbaogroups openbaoentities openbaoconnections; do
  echo "Deleting ${resource} in ${TEST_NAMESPACE}"
  kubectl_cmd -n "${TEST_NAMESPACE}" delete "${resource}" --all --ignore-not-found=true --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
done

echo "Deleting namespace ${TEST_NAMESPACE}"
kubectl_cmd delete namespace "${TEST_NAMESPACE}" --ignore-not-found=true --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
