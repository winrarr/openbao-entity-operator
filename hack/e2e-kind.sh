#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
OPENBAO_NAMESPACE=${OPENBAO_NAMESPACE:-openbao}
OPENBAO_DEPLOYMENT=${OPENBAO_DEPLOYMENT:-openbao}
OPENBAO_TOKEN_SECRET=${OPENBAO_TOKEN_SECRET:-openbao-dev-token}
OPENBAO_TOKEN_KEY=${OPENBAO_TOKEN_KEY:-token}
TEST_NAMESPACE=${TEST_NAMESPACE:-openbao-entity-operator-e2e}
TENANT_B_NAMESPACE=${TENANT_B_NAMESPACE:-openbao-entity-operator-e2e-b}
OUTSIDE_NAMESPACE=${OUTSIDE_NAMESPACE:-openbao-entity-operator-outside}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-openbao-entity-operator-system}
operator_deployment=${OPERATOR_DEPLOYMENT:-openbao-entity-operator}
CLEANUP_TOKEN_SECRET=${CLEANUP_TOKEN_SECRET:-e2e-cleanup-token}
KEEP_TEST_RESOURCES=${KEEP_TEST_RESOURCES:-false}
cleanup_script=${CLEANUP_SCRIPT:-hack/cleanup-kind-e2e.sh}

if [[ "${OUTSIDE_NAMESPACE}" == "${TEST_NAMESPACE}" ]]; then
  echo "OUTSIDE_NAMESPACE must differ from TEST_NAMESPACE" >&2
  exit 1
fi

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

cleanup() {
  local exit_code=$?
  if [[ "${KEEP_TEST_RESOURCES}" == true || "${exit_code}" -ne 0 ]]; then
    echo "Keeping ${TEST_NAMESPACE} and ${TENANT_B_NAMESPACE} for inspection (exit ${exit_code})" >&2
    exit "${exit_code}"
  fi
  if ! KUBECTL="${KUBECTL}" KUBE_CONTEXT="${KUBE_CONTEXT}" TEST_NAMESPACE="${TEST_NAMESPACE}" TENANT_B_NAMESPACE="${TENANT_B_NAMESPACE}" OUTSIDE_NAMESPACE="${OUTSIDE_NAMESPACE}" OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE}" "${cleanup_script}"; then
    echo "Failed to clean up ${TEST_NAMESPACE}; run make kind-e2e-clean to retry" >&2
    exit 1
  fi
  exit "${exit_code}"
}
trap cleanup EXIT

wait_ready() {
  local resource="$1"
  kubectl_cmd -n "${TEST_NAMESPACE}" wait \
    --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
    "${resource}" --timeout=5m >/dev/null
}

wait_for_jsonpath() {
  local resource="$1"
  local jsonpath="$2"
  local expected="$3"
  local actual
  for _ in {1..150}; do
    actual="$(kubectl_cmd -n "${TEST_NAMESPACE}" get "${resource}" -o "jsonpath=${jsonpath}" 2>/dev/null || true)"
    if [[ "${actual}" == "${expected}" ]]; then
      return 0
    fi
    sleep 2
  done
  echo "${resource} did not reach ${jsonpath}=${expected}" >&2
  kubectl_cmd -n "${TEST_NAMESPACE}" get "${resource}" -o yaml >&2 || true
  return 1
}

openbao_cli() {
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 bao "$@"
}

openbao_cli_stdin() {
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec -i "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 bao "$@"
}

copy_openbao_token_secret() {
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" get secret "${OPENBAO_TOKEN_SECRET}" -o json | jq \
    --arg namespace "${TEST_NAMESPACE}" \
    --arg name "${CLEANUP_TOKEN_SECRET}" \
    'del(.metadata.creationTimestamp, .metadata.managedFields, .metadata.ownerReferences, .metadata.resourceVersion, .metadata.uid) |
     .metadata.namespace = $namespace |
     .metadata.name = $name' | kubectl_cmd apply -f - >/dev/null
}

openbao_list_ids() {
  local path="$1"
  openbao_cli list -format=json "${path}" 2>/dev/null | jq -r \
    'if type == "object" then .data.keys[]? else .[]? end' || true
}

remote_alias() {
  local id="$1"
  openbao_cli read -format=json "identity/entity-alias/id/${id}"
}

remote_group() {
  local id="$1"
  openbao_cli read -format=json "identity/group/id/${id}"
}

reset_remote_test_objects() {
  local alias_id alias_name group_id group_name name

  openbao_cli delete auth/approle/role/e2e-approle >/dev/null 2>&1 || true

  while IFS= read -r alias_id; do
    [[ -n "${alias_id}" ]] || continue
    alias_name="$(remote_alias "${alias_id}" 2>/dev/null | jq -r '.data.name // empty' || true)"
    case "${alias_name}" in
      e2e-login|e2e-adopt-login|e2e-conflict-login|e2e-orphan-login)
        openbao_cli delete "identity/entity-alias/id/${alias_id}" >/dev/null 2>&1 || true
        ;;
    esac
  done < <(openbao_list_ids identity/entity-alias/id)

  while IFS= read -r group_id; do
    [[ -n "${group_id}" ]] || continue
    group_name="$(remote_group "${group_id}" 2>/dev/null | jq -r '.data.name // empty' || true)"
    case "${group_name}" in
      e2e-group|e2e-child-group)
        openbao_cli delete "identity/group/id/${group_id}" >/dev/null 2>&1 || true
        ;;
    esac
  done < <(openbao_list_ids identity/group/id)

  for name in \
    e2e-entity e2e-created e2e-adopt e2e-group-created-member e2e-unmanaged-member \
    e2e-child-member e2e-group-child-member e2e-conflict-alias-target \
    e2e-orphan-alias-target e2e-conflict e2e-orphan e2e-cleanup; do
    openbao_cli delete "identity/entity/name/${name}" >/dev/null 2>&1 || true
  done

  openbao_cli delete auth/kubernetes/role/e2e-managed-role >/dev/null 2>&1 || true

  for name in e2e-policy e2e-policy-conflict e2e-policy-adopt e2e-policy-orphan e2e-policy-delete; do
    openbao_cli delete "sys/policies/acl/${name}" >/dev/null 2>&1 || true
  done
}

configure_kubernetes_auth() {
  if ! openbao_cli auth list -format=json | jq -e 'has("kubernetes/")' >/dev/null 2>&1; then
    openbao_cli auth enable kubernetes >/dev/null
  fi

  openbao_cli write auth/kubernetes/config \
    kubernetes_host=https://kubernetes.default.svc:443 \
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
    token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token >/dev/null

  openbao_cli_stdin policy write e2e-operator-policy - >/dev/null <<'EOF'
path "sys/health" {
  capabilities = ["read"]
}

path "auth/token/lookup-self" {
  capabilities = ["read"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}

path "identity/entity/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/entity" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/entity-alias/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/entity-alias" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/group/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "identity/group" {
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

  openbao_cli write auth/kubernetes/role/e2e-operator \
    bound_service_account_names=openbao-entity-operator \
    bound_service_account_namespaces="${OPERATOR_NAMESPACE}" \
    token_policies=e2e-operator-policy \
    token_period=5m >/dev/null
}

configure_approle_auth() {
  if ! openbao_cli auth list -format=json | jq -e 'has("approle/")' >/dev/null 2>&1; then
    openbao_cli auth enable approle >/dev/null
  fi

  openbao_cli write auth/approle/role/e2e-approle \
    token_policies=e2e-operator-policy \
    token_period=5m >/dev/null

  local role_id secret_id
  role_id="$(openbao_cli read -field=role_id auth/approle/role/e2e-approle/role-id)"
  secret_id="$(openbao_cli write -f -field=secret_id auth/approle/role/e2e-approle/secret-id)"

  kubectl_cmd -n "${TEST_NAMESPACE}" create secret generic e2e-approle-role \
    --from-literal=role-id="${role_id}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
  kubectl_cmd -n "${TEST_NAMESPACE}" create secret generic e2e-approle-secret \
    --from-literal=secret-id="${secret_id}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

assert_remote_policy() {
  local name="$1"
  local expected_rules="$2"
  openbao_cli read -format=json "sys/policies/acl/${name}" | jq -e \
    --arg name "${name}" --arg rules "${expected_rules}" \
    '.data.name == $name and (.data.policy | rtrimstr("\n")) == ($rules | rtrimstr("\n"))' >/dev/null
}

echo "Using Kind context ${KUBE_CONTEXT}"
kubectl_cmd -n "${OPENBAO_NAMESPACE}" wait --for=condition=available \
  "deployment/${OPENBAO_DEPLOYMENT}" --timeout=5m >/dev/null
kubectl_cmd -n "${OPERATOR_NAMESPACE}" wait --for=condition=available \
  "deployment/${operator_deployment}" --timeout=5m >/dev/null

for crd in openbaoconnections openbaopolicies openbaokubernetesauthroles openbaoentities openbaoentityaliases openbaogroups openbaogroupmemberships; do
  kubectl_cmd get crd "${crd}.openbao.openbao-operator.io" >/dev/null
done
kubectl_cmd -n "${TEST_NAMESPACE}" get rolebinding "${operator_deployment}-manager" >/dev/null
kubectl_cmd -n "${TENANT_B_NAMESPACE}" get rolebinding "${operator_deployment}-manager" >/dev/null
if kubectl_cmd get clusterrolebinding "${operator_deployment}-manager" >/dev/null 2>&1; then
  echo "scoped installation unexpectedly created a manager ClusterRoleBinding" >&2
  exit 1
fi

reset_remote_test_objects
configure_kubernetes_auth
kubectl_cmd create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
kubectl_cmd create namespace "${OUTSIDE_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
configure_approle_auth

kubectl_cmd -n "${OUTSIDE_NAMESPACE}" delete openbaoconnection outside-scope --ignore-not-found=true >/dev/null
cat <<EOF | kubectl_cmd -n "${OUTSIDE_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: outside-scope
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  kubernetesAuth:
    mountPath: kubernetes
    role: e2e-operator
EOF
for _ in {1..20}; do
  if [[ -n "$(kubectl_cmd -n "${OUTSIDE_NAMESPACE}" get openbaoconnection/outside-scope -o 'jsonpath={.status.conditions[0].type}' 2>/dev/null || true)" ]]; then
    echo "operator reconciled a resource outside its watch namespace allowlist" >&2
    kubectl_cmd -n "${OUTSIDE_NAMESPACE}" get openbaoconnection/outside-scope -o yaml >&2 || true
    exit 1
  fi
  sleep 1
done

cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  kubernetesAuth:
    mountPath: kubernetes
    role: e2e-operator
EOF
wait_ready openbaoconnection/openbao

cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: openbao-approle
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  appRole:
    mountPath: approle
    roleIDSecretRef:
      name: e2e-approle-role
    secretIDSecretRef:
      name: e2e-approle-secret
EOF
wait_ready openbaoconnection/openbao-approle

copy_openbao_token_secret
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: e2e-cleanup-token
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  tokenSecretRef:
    name: ${CLEANUP_TOKEN_SECRET}
    key: ${OPENBAO_TOKEN_KEY}
EOF
wait_ready openbaoconnection/e2e-cleanup-token

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: e2e-policy
spec:
  connectionRef:
    name: openbao
  deletionPolicy: Delete
  rules: |
    path "identity/entity/name/e2e-entity" {
      capabilities = ["read"]
    }
EOF
wait_ready openbaopolicy/e2e-policy
policy_hash="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaopolicy/e2e-policy -o jsonpath='{.status.rulesHash}')"
policy_version="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaopolicy/e2e-policy -o jsonpath='{.status.version}')"
[[ -n "${policy_hash}" && "${policy_version}" =~ ^[1-9][0-9]*$ ]] || {
  echo "policy status did not publish a hash and positive version" >&2
  exit 1
}
assert_remote_policy e2e-policy 'path "identity/entity/name/e2e-entity" {
  capabilities = ["read"]
}'

kubectl_cmd -n "${TEST_NAMESPACE}" create serviceaccount e2e-workload --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoKubernetesAuthRole
metadata:
  name: e2e-managed-role
spec:
  connectionRef:
    name: openbao
  boundServiceAccountNames:
    - e2e-workload
  boundServiceAccountNamespaces:
    - ${TEST_NAMESPACE}
  tokenPolicies:
    - e2e-policy
  tokenPeriod: 5m
EOF
wait_ready openbaokubernetesauthrole/e2e-managed-role
role_hash="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaokubernetesauthrole/e2e-managed-role -o jsonpath='{.status.configHash}')"
[[ -n "${role_hash}" ]] || { echo "Kubernetes Auth role did not publish a configuration hash" >&2; exit 1; }
openbao_cli read -format=json auth/kubernetes/role/e2e-managed-role | jq -e \
  --arg namespace "${TEST_NAMESPACE}" \
  '.data as $data | ($data.bound_service_account_names | index("e2e-workload")) != null and ($data.bound_service_account_namespaces | index($namespace)) != null and ($data.token_policies | index("e2e-policy")) != null' >/dev/null
workload_jwt="$(kubectl_cmd -n "${TEST_NAMESPACE}" create token e2e-workload --duration=10m)"
openbao_cli write -format=json auth/kubernetes/login role=e2e-managed-role jwt="${workload_jwt}" | jq -e \
  '.auth.policies | index("e2e-policy") != null' >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" create serviceaccount e2e-other --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
other_jwt="$(kubectl_cmd -n "${TEST_NAMESPACE}" create token e2e-other --duration=10m)"
if openbao_cli write -format=json auth/kubernetes/login role=e2e-managed-role jwt="${other_jwt}" >/dev/null 2>&1; then
  echo "Kubernetes Auth role accepted an unbound ServiceAccount" >&2
  exit 1
fi

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-entity
spec:
  connectionRef:
    name: openbao
  policies:
    - e2e-policy
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-entity
entity_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-entity -o jsonpath='{.status.id}')"
[[ -n "${entity_id}" ]] || { echo "entity did not publish an ID" >&2; exit 1; }
openbao_cli read -format=json "identity/entity/id/${entity_id}" | jq -e \
  --arg id "${entity_id}" '.data.id == $id and .data.name == "e2e-entity" and (.data.policies | index("e2e-policy")) != null' >/dev/null

auth_accessor="$(openbao_cli auth list -format=json | jq -er '."token/".accessor')"
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntityAlias
metadata:
  name: e2e-alias
spec:
  connectionRef:
    name: openbao
  entityRef:
    name: e2e-entity
  mountAccessor: ${auth_accessor}
  name: e2e-login
  deletionPolicy: Delete
EOF
wait_ready openbaoentityalias/e2e-alias
alias_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentityalias/e2e-alias -o jsonpath='{.status.id}')"
[[ -n "${alias_id}" ]] || { echo "entity alias did not publish an ID" >&2; exit 1; }
remote_alias "${alias_id}" | jq -e \
  --arg id "${alias_id}" --arg accessor "${auth_accessor}" --arg entity_id "${entity_id}" \
  '.data.id == $id and .data.name == "e2e-login" and .data.mount_accessor == $accessor and .data.canonical_id == $entity_id' >/dev/null

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroup
metadata:
  name: e2e-group
spec:
  connectionRef:
    name: openbao
  deletionPolicy: Delete
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupMembership
metadata:
  name: e2e-group-membership
spec:
  groupRef:
    name: e2e-group
  entityRef:
    name: e2e-entity
EOF
wait_ready openbaogroup/e2e-group
wait_ready openbaogroupmembership/e2e-group-membership
group_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaogroup/e2e-group -o jsonpath='{.status.id}')"
membership_member_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaogroupmembership/e2e-group-membership -o jsonpath='{.status.memberID}')"
[[ "${membership_member_id}" == "${entity_id}" ]] || {
  echo "membership status member ID ${membership_member_id} did not match entity ID ${entity_id}" >&2
  exit 1
}
remote_group "${group_id}" | jq -e --arg entity_id "${entity_id}" \
  '(.data.member_entity_ids // []) | index($entity_id) != null' >/dev/null

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-cleanup
spec:
  connectionRef:
    name: e2e-cleanup-token
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-cleanup
cleanup_entity_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-cleanup -o jsonpath='{.status.id}')"
[[ -n "${cleanup_entity_id}" ]] || { echo "cleanup entity did not publish an ID" >&2; exit 1; }

kubectl_cmd -n "${TEST_NAMESPACE}" delete secret "${CLEANUP_TOKEN_SECRET}" --wait=true >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-cleanup --wait=false >/dev/null
wait_for_jsonpath openbaoentity/e2e-cleanup '{.status.conditions[?(@.type=="CleanupRequired")].status}' True
finalizer="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-cleanup -o jsonpath='{.metadata.finalizers[0]}')"
[[ -n "${finalizer}" ]] || { echo "cleanup entity lost its finalizer while its credential was unavailable" >&2; exit 1; }

copy_openbao_token_secret
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete openbaoentity/e2e-cleanup --timeout=5m >/dev/null
if openbao_cli read "identity/entity/id/${cleanup_entity_id}" >/dev/null 2>&1; then
  echo "cleanup entity still exists in OpenBao after credential recovery" >&2
  exit 1
fi

echo "Kind/OpenBao smoke and integration scenarios passed"
