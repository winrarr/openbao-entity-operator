#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
TEST_NAMESPACE=${TEST_NAMESPACE:-openbao-entity-operator-e2e}
TENANT_B_NAMESPACE=${TENANT_B_NAMESPACE:-openbao-entity-operator-e2e-b}
OUTSIDE_NAMESPACE=${OUTSIDE_NAMESPACE:-openbao-entity-operator-outside}
OPENBAO_NAMESPACE=${OPENBAO_NAMESPACE:-openbao}
TENANT_A_OPENBAO_NAMESPACE=${TENANT_A_OPENBAO_NAMESPACE:-openbao-entity-operator-tenant-a}
TENANT_B_OPENBAO_NAMESPACE=${TENANT_B_OPENBAO_NAMESPACE:-openbao-entity-operator-tenant-b}
OPENBAO_DEPLOYMENT=${OPENBAO_DEPLOYMENT:-openbao}
DELETE_TIMEOUT=${DELETE_TIMEOUT:-5m}

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

kubectl_cmd cluster-info >/dev/null

cleanup_namespace() {
  local namespace="$1"
  if ! kubectl_cmd get namespace "${namespace}" >/dev/null 2>&1; then
    return 0
  fi

  for resource in openbaogroupmemberships openbaoentityaliases openbaokubernetesauthroles openbaopolicies openbaogroups openbaoentities openbaoconnections; do
    echo "Deleting ${resource} in ${namespace}"
    kubectl_cmd -n "${namespace}" delete "${resource}" --all --ignore-not-found=true --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
  done

  echo "Deleting namespace ${namespace}"
  kubectl_cmd delete namespace "${namespace}" --ignore-not-found=true --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
}

cleanup_namespace "${TEST_NAMESPACE}"
if [[ "${TENANT_B_NAMESPACE}" != "${TEST_NAMESPACE}" ]]; then
  cleanup_namespace "${TENANT_B_NAMESPACE}"
fi

if kubectl_cmd get namespace "${OUTSIDE_NAMESPACE}" >/dev/null 2>&1; then
	if [[ "${OUTSIDE_NAMESPACE}" == "${TEST_NAMESPACE}" ]]; then
		echo "OUTSIDE_NAMESPACE must differ from TEST_NAMESPACE" >&2
		exit 1
	fi
	echo "Deleting namespace ${OUTSIDE_NAMESPACE}"
	kubectl_cmd delete namespace "${OUTSIDE_NAMESPACE}" --ignore-not-found=true --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
fi

openbao_cli() {
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 bao "$@"
}

if kubectl_cmd -n "${OPENBAO_NAMESPACE}" get deployment "${OPENBAO_DEPLOYMENT}" >/dev/null 2>&1; then
  for namespace in "${TENANT_A_OPENBAO_NAMESPACE}" "${TENANT_B_OPENBAO_NAMESPACE}"; do
    openbao_cli namespace delete "${namespace}" >/dev/null 2>&1 || true
  done
fi
