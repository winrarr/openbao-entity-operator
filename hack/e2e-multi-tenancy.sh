#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
OPENBAO_NAMESPACE=${OPENBAO_NAMESPACE:-openbao}
OPENBAO_DEPLOYMENT=${OPENBAO_DEPLOYMENT:-openbao}
TEST_NAMESPACE=${TEST_NAMESPACE:-openbao-entity-operator-e2e}
TENANT_B_NAMESPACE=${TENANT_B_NAMESPACE:-openbao-entity-operator-e2e-b}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-openbao-entity-operator-system}
OPERATOR_DEPLOYMENT=${OPERATOR_DEPLOYMENT:-openbao-entity-operator}
OPERATOR_SERVICE_ACCOUNT=${OPERATOR_SERVICE_ACCOUNT:-${OPERATOR_DEPLOYMENT}}
TENANT_AUTHOR_ROLE=${TENANT_AUTHOR_ROLE:-${OPERATOR_DEPLOYMENT}-tenant-author-role}
TENANT_A_OPENBAO_NAMESPACE=${TENANT_A_OPENBAO_NAMESPACE:-openbao-entity-operator-tenant-a}
TENANT_B_OPENBAO_NAMESPACE=${TENANT_B_OPENBAO_NAMESPACE:-openbao-entity-operator-tenant-b}
TENANT_A_AUTH_ROLE=${TENANT_A_AUTH_ROLE:-openbao-entity-operator}
TENANT_B_AUTH_ROLE=${TENANT_B_AUTH_ROLE:-openbao-entity-operator}
DELETE_TIMEOUT=${DELETE_TIMEOUT:-5m}

if [[ "${TEST_NAMESPACE}" == "${TENANT_B_NAMESPACE}" ]]; then
  echo "TENANT_B_NAMESPACE must differ from TEST_NAMESPACE" >&2
  exit 1
fi
if [[ "${TENANT_A_OPENBAO_NAMESPACE}" == "${TENANT_B_OPENBAO_NAMESPACE}" ]]; then
  echo "tenant OpenBao namespaces must differ" >&2
  exit 1
fi

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

openbao_cli() {
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 bao "$@"
}

openbao_cli_namespace() {
  local namespace="$1"
  shift
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 BAO_NAMESPACE="${namespace}" bao "$@"
}

openbao_cli_namespace_stdin() {
  local namespace="$1"
  shift
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec -i "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 BAO_NAMESPACE="${namespace}" bao "$@"
}

openbao_cli_namespace_with_token() {
  local namespace="$1"
  local token="$2"
  shift 2
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 BAO_NAMESPACE="${namespace}" BAO_TOKEN="${token}" bao "$@"
}

service_account_principal() {
  printf 'system:serviceaccount:%s:%s\n' "$1" "$2"
}

assert_can() {
  local principal="$1"
  local namespace="$2"
  local verb="$3"
  local resource="$4"
  local result
  result="$(kubectl_cmd auth can-i --as="${principal}" -n "${namespace}" "${verb}" "${resource}" || true)"
  if [[ "${result}" != yes ]]; then
    echo "${principal} cannot ${verb} ${resource} in ${namespace}" >&2
    return 1
  fi
  return 0
}

assert_cannot() {
  local principal="$1"
  local namespace="$2"
  local verb="$3"
  local resource="$4"
  local result
  result="$(kubectl_cmd auth can-i --as="${principal}" -n "${namespace}" "${verb}" "${resource}" || true)"
  if [[ "${result}" == yes ]]; then
    echo "${principal} unexpectedly can ${verb} ${resource} in ${namespace}" >&2
    return 1
  fi
  return 0
}

assert_command_denied() {
  if "$@" >/dev/null 2>&1; then
    echo "command unexpectedly succeeded: $*" >&2
    return 1
  fi
  return 0
}

apply_as() {
  local namespace="$1"
  local service_account="$2"
  kubectl_cmd --as="$(service_account_principal "${namespace}" "${service_account}")" -n "${namespace}" apply -f -
}

wait_ready_as() {
  local namespace="$1"
  local service_account="$2"
  local resource="$3"
  kubectl_cmd --as="$(service_account_principal "${namespace}" "${service_account}")" -n "${namespace}" wait \
    --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
    "${resource}" --timeout=5m >/dev/null
}

ensure_openbao_namespace() {
  local namespace="$1"
  if ! openbao_cli namespace lookup "${namespace}" >/dev/null 2>&1; then
    openbao_cli namespace create "${namespace}" >/dev/null
  fi
}

configure_openbao_tenant() {
  local namespace="$1"
  local auth_role="$2"

  ensure_openbao_namespace "${namespace}"
  if ! openbao_cli_namespace "${namespace}" auth list -format=json | jq -e 'has("kubernetes/")' >/dev/null 2>&1; then
    openbao_cli_namespace "${namespace}" auth enable kubernetes >/dev/null
  fi

  openbao_cli_namespace "${namespace}" write auth/kubernetes/config \
    kubernetes_host=https://kubernetes.default.svc:443 \
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
    token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token >/dev/null

  openbao_cli_namespace_stdin "${namespace}" policy write "${auth_role}" - >/dev/null <<'EOF'
path "auth/token/lookup-self" {
  capabilities = ["read"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}

path "identity/entity" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/entity/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/entity-alias" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/entity-alias/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/group" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/group/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "sys/policies/acl/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "sys/policies/acl" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "auth/kubernetes/role" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "auth/kubernetes/role/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF

  openbao_cli_namespace "${namespace}" write "auth/kubernetes/role/${auth_role}" \
    bound_service_account_names="${OPERATOR_SERVICE_ACCOUNT}" \
    bound_service_account_namespaces="${OPERATOR_NAMESPACE}" \
    token_policies="${auth_role}" token_period=5m >/dev/null
}

create_tenant_authorization() {
  local namespace="$1"
  local service_account="$2"
  kubectl_cmd -n "${namespace}" create serviceaccount "${service_account}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
  kubectl_cmd -n "${namespace}" create rolebinding "${service_account}-author" \
    --clusterrole="${TENANT_AUTHOR_ROLE}" \
    --serviceaccount="${namespace}:${service_account}" \
    --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

create_platform_connection() {
  local namespace="$1"
  local name="$2"
  local bao_namespace="$3"
  local auth_role="$4"
  cat <<EOF | kubectl_cmd -n "${namespace}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: ${name}
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  namespace: ${bao_namespace}
  kubernetesAuth:
    mountPath: kubernetes
    role: ${auth_role}
EOF
}

create_platform_fixture_secret() {
  local namespace="$1"
  kubectl_cmd -n "${namespace}" create secret generic platform-credential-fixture \
    --from-literal=token=not-a-real-openbao-credential --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

cleanup_success() {
  for namespace in "${TEST_NAMESPACE}" "${TENANT_B_NAMESPACE}"; do
    for resource in openbaogroupmemberships openbaoentityaliases openbaokubernetesauthroles openbaopolicies openbaogroups openbaoentities openbaoconnections; do
      kubectl_cmd -n "${namespace}" delete "${resource}" --all --ignore-not-found=true --wait=true --timeout="${DELETE_TIMEOUT}" >/dev/null
    done
  done
  openbao_cli namespace delete "${TENANT_A_OPENBAO_NAMESPACE}" >/dev/null 2>&1 || true
  openbao_cli namespace delete "${TENANT_B_OPENBAO_NAMESPACE}" >/dev/null 2>&1 || true
}

cleanup() {
  local exit_code=$?
  if [[ "${exit_code}" -ne 0 ]]; then
    echo "Keeping multi-tenancy fixtures for inspection (exit ${exit_code})" >&2
    exit "${exit_code}"
  fi
  cleanup_success
  exit 0
}
trap cleanup EXIT

echo "Testing Kubernetes and OpenBao tenant isolation"
kubectl_cmd -n "${OPERATOR_NAMESPACE}" wait --for=condition=available \
  "deployment/${OPERATOR_DEPLOYMENT}" --timeout=5m >/dev/null

for namespace in "${TEST_NAMESPACE}" "${TENANT_B_NAMESPACE}"; do
  kubectl_cmd get namespace "${namespace}" >/dev/null
done

create_tenant_authorization "${TEST_NAMESPACE}" tenant-a-user
create_tenant_authorization "${TENANT_B_NAMESPACE}" tenant-b-user

tenant_a_principal="$(service_account_principal "${TEST_NAMESPACE}" tenant-a-user)"
tenant_b_principal="$(service_account_principal "${TENANT_B_NAMESPACE}" tenant-b-user)"

assert_can "${tenant_a_principal}" "${TEST_NAMESPACE}" create openbaoentities
assert_can "${tenant_b_principal}" "${TENANT_B_NAMESPACE}" create openbaoentities
assert_can "${tenant_a_principal}" "${TEST_NAMESPACE}" create openbaokubernetesauthroles
assert_can "${tenant_b_principal}" "${TENANT_B_NAMESPACE}" create openbaokubernetesauthroles
for principal in "${tenant_a_principal}" "${tenant_b_principal}"; do
  namespace="${TEST_NAMESPACE}"
  [[ "${principal}" == "${tenant_b_principal}" ]] && namespace="${TENANT_B_NAMESPACE}"
  assert_cannot "${principal}" "${namespace}" get openbaoconnections
  assert_cannot "${principal}" "${namespace}" create openbaoconnections
  assert_cannot "${principal}" "${namespace}" get secrets
done
assert_cannot "${tenant_a_principal}" "${TENANT_B_NAMESPACE}" get openbaoentities
assert_cannot "${tenant_a_principal}" "${TENANT_B_NAMESPACE}" create openbaoentities
assert_cannot "${tenant_b_principal}" "${TEST_NAMESPACE}" get openbaoentities
assert_cannot "${tenant_b_principal}" "${TEST_NAMESPACE}" create openbaoentities
assert_cannot "${tenant_a_principal}" "${TENANT_B_NAMESPACE}" get openbaokubernetesauthroles
assert_cannot "${tenant_b_principal}" "${TEST_NAMESPACE}" get openbaokubernetesauthroles

configure_openbao_tenant "${TENANT_A_OPENBAO_NAMESPACE}" "${TENANT_A_AUTH_ROLE}"
configure_openbao_tenant "${TENANT_B_OPENBAO_NAMESPACE}" "${TENANT_B_AUTH_ROLE}"
create_platform_connection "${TEST_NAMESPACE}" platform-connection "${TENANT_A_OPENBAO_NAMESPACE}" "${TENANT_A_AUTH_ROLE}"
create_platform_connection "${TENANT_B_NAMESPACE}" platform-connection "${TENANT_B_OPENBAO_NAMESPACE}" "${TENANT_B_AUTH_ROLE}"
create_platform_fixture_secret "${TEST_NAMESPACE}"
create_platform_fixture_secret "${TENANT_B_NAMESPACE}"
for fixture in tenant-a-workload tenant-b-workload; do
  namespace="${TEST_NAMESPACE}"
  [[ "${fixture}" == tenant-b-workload ]] && namespace="${TENANT_B_NAMESPACE}"
  kubectl_cmd -n "${namespace}" create serviceaccount "${fixture}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
done

kubectl_cmd -n "${TEST_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  openbaoconnection/platform-connection --timeout=5m >/dev/null
kubectl_cmd -n "${TENANT_B_NAMESPACE}" wait --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
  openbaoconnection/platform-connection --timeout=5m >/dev/null

assert_command_denied kubectl_cmd --as="${tenant_a_principal}" -n "${TEST_NAMESPACE}" \
  patch openbaoconnection/platform-connection --type=merge -p '{"spec":{"namespace":"'"${TENANT_B_OPENBAO_NAMESPACE}"'"}}'
assert_command_denied kubectl_cmd --as="${tenant_b_principal}" -n "${TENANT_B_NAMESPACE}" \
  delete openbaoconnection/platform-connection
assert_command_denied kubectl_cmd --as="${tenant_a_principal}" -n "${TEST_NAMESPACE}" get secret platform-credential-fixture
assert_command_denied kubectl_cmd --as="${tenant_b_principal}" -n "${TENANT_B_NAMESPACE}" get secret platform-credential-fixture

cat <<EOF | apply_as "${TEST_NAMESPACE}" tenant-a-user >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: tenant-a-policy
spec:
  connectionRef:
    name: platform-connection
  rules: |
    path "identity/entity/name/tenant-a-entity" {
      capabilities = ["read"]
    }
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: tenant-a-entity
spec:
  connectionRef:
    name: platform-connection
  policies:
    - tenant-a-policy
EOF
cat <<EOF | apply_as "${TENANT_B_NAMESPACE}" tenant-b-user >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: tenant-b-policy
spec:
  connectionRef:
    name: platform-connection
  rules: |
    path "identity/entity/name/tenant-b-entity" {
      capabilities = ["read"]
    }
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: tenant-b-entity
spec:
  connectionRef:
    name: platform-connection
  policies:
    - tenant-b-policy
EOF

cat <<EOF | apply_as "${TEST_NAMESPACE}" tenant-a-user >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoKubernetesAuthRole
metadata:
  name: tenant-a-workload
spec:
  connectionRef:
    name: platform-connection
  boundServiceAccountNames:
    - tenant-a-workload
  boundServiceAccountNamespaces:
    - ${TEST_NAMESPACE}
  tokenPolicies:
    - ${TENANT_A_AUTH_ROLE}
  tokenPeriod: 5m
EOF
cat <<EOF | apply_as "${TENANT_B_NAMESPACE}" tenant-b-user >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoKubernetesAuthRole
metadata:
  name: tenant-b-workload
spec:
  connectionRef:
    name: platform-connection
  boundServiceAccountNames:
    - tenant-b-workload
  boundServiceAccountNamespaces:
    - ${TENANT_B_NAMESPACE}
  tokenPolicies:
    - ${TENANT_B_AUTH_ROLE}
  tokenPeriod: 5m
EOF

wait_ready_as "${TEST_NAMESPACE}" tenant-a-user openbaopolicy/tenant-a-policy
wait_ready_as "${TEST_NAMESPACE}" tenant-a-user openbaoentity/tenant-a-entity
wait_ready_as "${TENANT_B_NAMESPACE}" tenant-b-user openbaopolicy/tenant-b-policy
wait_ready_as "${TENANT_B_NAMESPACE}" tenant-b-user openbaoentity/tenant-b-entity
wait_ready_as "${TEST_NAMESPACE}" tenant-a-user openbaokubernetesauthrole/tenant-a-workload
wait_ready_as "${TENANT_B_NAMESPACE}" tenant-b-user openbaokubernetesauthrole/tenant-b-workload

openbao_cli_namespace "${TENANT_A_OPENBAO_NAMESPACE}" read identity/entity/name/tenant-a-entity >/dev/null
openbao_cli_namespace "${TENANT_B_OPENBAO_NAMESPACE}" read identity/entity/name/tenant-b-entity >/dev/null
if openbao_cli_namespace "${TENANT_A_OPENBAO_NAMESPACE}" read identity/entity/name/tenant-b-entity >/dev/null 2>&1; then
  echo "tenant A OpenBao namespace unexpectedly contains tenant B entity" >&2
  exit 1
fi
if openbao_cli_namespace "${TENANT_B_OPENBAO_NAMESPACE}" read identity/entity/name/tenant-a-entity >/dev/null 2>&1; then
  echo "tenant B OpenBao namespace unexpectedly contains tenant A entity" >&2
  exit 1
fi

operator_jwt="$(kubectl_cmd -n "${OPERATOR_NAMESPACE}" create token "${OPERATOR_SERVICE_ACCOUNT}" --duration=10m)"
tenant_a_token="$(openbao_cli_namespace "${TENANT_A_OPENBAO_NAMESPACE}" write -field=token auth/kubernetes/login role="${TENANT_A_AUTH_ROLE}" jwt="${operator_jwt}")"
tenant_b_token="$(openbao_cli_namespace "${TENANT_B_OPENBAO_NAMESPACE}" write -field=token auth/kubernetes/login role="${TENANT_B_AUTH_ROLE}" jwt="${operator_jwt}")"
openbao_cli_namespace_with_token "${TENANT_A_OPENBAO_NAMESPACE}" "${tenant_a_token}" read identity/entity/name/tenant-a-entity >/dev/null
openbao_cli_namespace_with_token "${TENANT_B_OPENBAO_NAMESPACE}" "${tenant_b_token}" read identity/entity/name/tenant-b-entity >/dev/null
assert_command_denied openbao_cli_namespace_with_token "${TENANT_B_OPENBAO_NAMESPACE}" "${tenant_a_token}" \
  write identity/entity/name/tenant-b-escape name=tenant-b-escape
assert_command_denied openbao_cli_namespace_with_token "${TENANT_A_OPENBAO_NAMESPACE}" "${tenant_b_token}" \
  write identity/entity/name/tenant-a-escape name=tenant-a-escape

tenant_a_workload_jwt="$(kubectl_cmd -n "${TEST_NAMESPACE}" create token tenant-a-workload --duration=10m)"
tenant_b_workload_jwt="$(kubectl_cmd -n "${TENANT_B_NAMESPACE}" create token tenant-b-workload --duration=10m)"
tenant_a_workload_token="$(openbao_cli_namespace "${TENANT_A_OPENBAO_NAMESPACE}" write -field=token auth/kubernetes/login role=tenant-a-workload jwt="${tenant_a_workload_jwt}")"
tenant_b_workload_token="$(openbao_cli_namespace "${TENANT_B_OPENBAO_NAMESPACE}" write -field=token auth/kubernetes/login role=tenant-b-workload jwt="${tenant_b_workload_jwt}")"
openbao_cli_namespace_with_token "${TENANT_A_OPENBAO_NAMESPACE}" "${tenant_a_workload_token}" read auth/token/lookup-self >/dev/null
openbao_cli_namespace_with_token "${TENANT_B_OPENBAO_NAMESPACE}" "${tenant_b_workload_token}" read auth/token/lookup-self >/dev/null
if openbao_cli_namespace "${TENANT_A_OPENBAO_NAMESPACE}" write -field=token auth/kubernetes/login role=tenant-a-workload jwt="${tenant_b_workload_jwt}" >/dev/null 2>&1; then
  echo "tenant A Kubernetes Auth role accepted tenant B's ServiceAccount" >&2
  exit 1
fi

echo "Kubernetes and OpenBao tenant isolation scenarios passed"
