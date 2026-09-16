#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
OPENBAO_NAMESPACE=${OPENBAO_NAMESPACE:-openbao}
OPENBAO_TOKEN_SECRET=${OPENBAO_TOKEN_SECRET:-openbao-dev-token}
OPENBAO_TOKEN_KEY=${OPENBAO_TOKEN_KEY:-token}
TEST_NAMESPACE=${TEST_NAMESPACE:-openbao-entity-operator-e2e}
OPENBAO_DEPLOYMENT=${OPENBAO_DEPLOYMENT:-openbao}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-openbao-entity-operator-system}
operator_deployment=${OPERATOR_DEPLOYMENT:-openbao-entity-operator}
KEEP_TEST_RESOURCES=${KEEP_TEST_RESOURCES:-false}
cleanup_script=${CLEANUP_SCRIPT:-hack/cleanup-kind-e2e.sh}

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

cleanup() {
	local exit_code=$?
	if [[ "${KEEP_TEST_RESOURCES}" == true || "${exit_code}" -ne 0 ]]; then
		echo "Keeping ${TEST_NAMESPACE} for inspection (exit ${exit_code})" >&2
		exit "${exit_code}"
	fi
	if ! KUBECTL="${KUBECTL}" KUBE_CONTEXT="${KUBE_CONTEXT}" TEST_NAMESPACE="${TEST_NAMESPACE}" "${cleanup_script}"; then
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

wait_for_condition_reason() {
  local resource="$1"
  local expected_reason="$2"
  local reason
  for _ in {1..90}; do
    reason="$(kubectl_cmd -n "${TEST_NAMESPACE}" get "${resource}" -o 'jsonpath={.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null || true)"
    if [[ "${reason}" == "${expected_reason}" ]]; then
      return 0
    fi
    sleep 2
  done
  echo "${resource} did not reach Ready reason ${expected_reason}" >&2
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

openbao_cli_namespace() {
  local namespace="$1"
  shift
  kubectl_cmd -n "${OPENBAO_NAMESPACE}" exec "deployment/${OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR=http://127.0.0.1:8200 BAO_NAMESPACE="${namespace}" bao "$@"
}

openbao_list_ids() {
  local path="$1"
  openbao_cli list -format=json "${path}" 2>/dev/null | jq -r \
    'if type == "object" then .data.keys[]? else .[]? end' || true
}

reset_remote_test_objects() {
  local alias_id alias_name group_id group_name name
  local alias_ids group_ids

  alias_ids="$(openbao_list_ids identity/entity-alias/id)"
  while IFS= read -r alias_id; do
    [[ -n "${alias_id}" ]] || continue
    alias_name="$(remote_alias "${alias_id}" 2>/dev/null | jq -r '.data.name // empty' || true)"
    case "${alias_name}" in
      e2e-login|e2e-adopt-login|e2e-conflict-login|e2e-orphan-login)
        openbao_cli delete "identity/entity-alias/id/${alias_id}" >/dev/null 2>&1 || true
        ;;
    esac
  done <<< "${alias_ids}"

  group_ids="$(openbao_list_ids identity/group/id)"
  while IFS= read -r group_id; do
    [[ -n "${group_id}" ]] || continue
    group_name="$(remote_group "${group_id}" 2>/dev/null | jq -r '.data.name // empty' || true)"
    case "${group_name}" in
      e2e-group|e2e-child-group)
        openbao_cli delete "identity/group/id/${group_id}" >/dev/null 2>&1 || true
        ;;
    esac
  done <<< "${group_ids}"

  for name in \
    e2e-created e2e-adopt e2e-group-created-member e2e-unmanaged-member e2e-child-member \
    e2e-group-child-member e2e-conflict-alias-target e2e-orphan-alias-target e2e-conflict \
    e2e-orphan e2e-missing-connection e2e-missing-secret; do
    openbao_cli delete "identity/entity/name/${name}" >/dev/null 2>&1 || true
  done
}

reset_remote_test_namespaces() {
  local namespace deleted
  for namespace in e2e-namespace-a e2e-namespace-b; do
    openbao_cli namespace delete "${namespace}" >/dev/null 2>&1 || true
    deleted=false
    for _ in {1..30}; do
      if ! openbao_cli namespace lookup "${namespace}" >/dev/null 2>&1; then
        deleted=true
        break
      fi
      sleep 1
    done
    if [[ "${deleted}" != true ]]; then
      echo "OpenBao namespace ${namespace} did not finish deleting" >&2
      return 1
    fi
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
EOF

  openbao_cli write auth/kubernetes/role/e2e-operator \
    bound_service_account_names=openbao-entity-operator \
    bound_service_account_namespaces="${OPERATOR_NAMESPACE}" \
    token_policies=e2e-operator-policy \
    token_period=5m >/dev/null
}

remote_entity() {
  local id="$1"
  openbao_cli read -format=json "identity/entity/id/${id}"
}

remote_entity_in_namespace() {
  local namespace="$1"
  local id="$2"
  openbao_cli_namespace "${namespace}" read -format=json "identity/entity/id/${id}"
}

remote_alias() {
  local id="$1"
  openbao_cli read -format=json "identity/entity-alias/id/${id}"
}

remote_group() {
  local id="$1"
  openbao_cli read -format=json "identity/group/id/${id}"
}

assert_remote_entity() {
  local id="$1"
  local expected_name="$2"
  local expected_owner="$3"
	if [[ -n "${expected_owner}" ]]; then
		remote_entity "${id}" | jq -e --arg id "${id}" --arg name "${expected_name}" --arg owner "${expected_owner}" \
			'.data.id == $id and .data.name == $name and .data.metadata.owner == $owner and (.data.policies | index("default")) != null' >/dev/null
		return
	fi
	remote_entity "${id}" | jq -e --arg id "${id}" --arg name "${expected_name}" \
		'.data.id == $id and .data.name == $name and (.data.policies | index("default")) != null' >/dev/null
}

assert_remote_entity_in_namespace() {
  local namespace="$1"
  local id="$2"
  local expected_name="$3"
  remote_entity_in_namespace "${namespace}" "${id}" | jq -e \
    --arg id "${id}" --arg name "${expected_name}" \
    '.data.id == $id and .data.name == $name and (.data.policies | index("default")) != null' >/dev/null
}

assert_remote_absent() {
  local id="$1"
  if remote_entity "${id}" >/dev/null 2>&1; then
    echo "OpenBao entity ${id} still exists" >&2
    return 1
  fi
}

assert_remote_entity_absent_in_namespace() {
  local namespace="$1"
  local id="$2"
  if remote_entity_in_namespace "${namespace}" "${id}" >/dev/null 2>&1; then
    echo "OpenBao entity ${id} still exists in namespace ${namespace}" >&2
    return 1
  fi
}

assert_remote_alias() {
  local id="$1"
  local expected_name="$2"
  local expected_accessor="$3"
  local expected_canonical_id="$4"
  remote_alias "${id}" | jq -e \
    --arg id "${id}" --arg name "${expected_name}" --arg accessor "${expected_accessor}" --arg canonical_id "${expected_canonical_id}" \
    '.data.id == $id and .data.name == $name and .data.mount_accessor == $accessor and .data.canonical_id == $canonical_id' >/dev/null
}

assert_remote_alias_absent() {
  local id="$1"
  if remote_alias "${id}" >/dev/null 2>&1; then
    echo "OpenBao entity alias ${id} still exists" >&2
    return 1
  fi
}

assert_remote_group_member() {
  local id="$1"
  local member_type="$2"
  local member_id="$3"
  local field
  case "${member_type}" in
    entity) field=member_entity_ids ;;
    group) field=member_group_ids ;;
    *) echo "unsupported group member type ${member_type}" >&2; return 1 ;;
  esac
  remote_group "${id}" | jq -e --arg id "${member_id}" --arg field "${field}" '.data[$field] | index($id) != null' >/dev/null
}

assert_remote_group_member_absent() {
  local id="$1"
  local member_type="$2"
  local member_id="$3"
  local field
  case "${member_type}" in
    entity) field=member_entity_ids ;;
    group) field=member_group_ids ;;
    *) echo "unsupported group member type ${member_type}" >&2; return 1 ;;
  esac
  remote_group "${id}" | jq -e --arg id "${member_id}" --arg field "${field}" '(.data[$field] // []) | index($id) == null' >/dev/null
}

wait_for_remote_group_member() {
  local id="$1"
  local member_type="$2"
  local member_id="$3"
  local expected="$4"
  for _ in {1..90}; do
    if [[ "${expected}" == present ]] && assert_remote_group_member "${id}" "${member_type}" "${member_id}"; then
      return 0
    fi
    if [[ "${expected}" == absent ]] && assert_remote_group_member_absent "${id}" "${member_type}" "${member_id}"; then
      return 0
    fi
    sleep 2
  done
  echo "OpenBao group ${id} member ${member_type}/${member_id} did not become ${expected}" >&2
  remote_group "${id}" >&2 || true
  return 1
}

assert_remote_group_absent() {
  local id="$1"
  if remote_group "${id}" >/dev/null 2>&1; then
    echo "OpenBao group ${id} still exists" >&2
    return 1
  fi
}

wait_for_remote_alias_canonical_id() {
  local id="$1"
  local expected_canonical_id="$2"
  for _ in {1..90}; do
    if remote_alias "${id}" | jq -e --arg canonical_id "${expected_canonical_id}" \
      '.data.canonical_id == $canonical_id' >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "OpenBao entity alias ${id} did not reach canonical ID ${expected_canonical_id}" >&2
  remote_alias "${id}" >&2 || true
  return 1
}

echo "Using Kind context ${KUBE_CONTEXT}"
kubectl_cmd -n "${OPENBAO_NAMESPACE}" wait --for=condition=available \
  "deployment/${OPENBAO_DEPLOYMENT}" --timeout=5m >/dev/null
kubectl_cmd -n "${OPERATOR_NAMESPACE}" wait --for=condition=available \
  "deployment/${operator_deployment}" --timeout=5m >/dev/null
reset_remote_test_objects
reset_remote_test_namespaces
configure_kubernetes_auth

kubectl_cmd create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
kubectl_cmd -n "${OPENBAO_NAMESPACE}" get secret "${OPENBAO_TOKEN_SECRET}" -o json \
  | jq 'del(.metadata.namespace, .metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.managedFields)' \
  | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null

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
kind: OpenBaoEntity
metadata:
  name: e2e-created
spec:
  connectionRef:
    name: openbao
  metadata:
    owner: platform
  policies:
    - default
  driftDetectionInterval: 5s
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-created
created_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-created -o jsonpath='{.status.id}')"
[[ -n "${created_id}" ]] || { echo "created entity did not publish an ID" >&2; exit 1; }
assert_remote_entity "${created_id}" e2e-created platform

kubectl_cmd -n "${TEST_NAMESPACE}" patch openbaoentity/e2e-created --type=merge \
  -p '{"spec":{"metadata":{"owner":"security"},"policies":["default","security"],"disabled":true}}' >/dev/null
wait_for_jsonpath openbaoentity/e2e-created '{.status.metadata.owner}' security
assert_remote_entity "${created_id}" e2e-created security
kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-created -o json \
  | jq -e '.status.disabled == true and (.status.policies | index("security")) != null' >/dev/null

openbao_cli delete "identity/entity/id/${created_id}" >/dev/null
new_created_id=""
for _ in {1..150}; do
	new_created_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-created -o jsonpath='{.status.id}' 2>/dev/null || true)"
	if [[ -n "${new_created_id}" && "${new_created_id}" != "${created_id}" ]] && remote_entity "${new_created_id}" >/dev/null 2>&1; then
		break
	fi
	sleep 2
done
[[ -n "${new_created_id}" && "${new_created_id}" != "${created_id}" ]] || {
	echo "OpenBaoEntity did not reacquire a new external entity after drift" >&2
	kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-created -o yaml >&2 || true
	exit 1
}
created_id="${new_created_id}"
assert_remote_entity "${created_id}" e2e-created security

openbao_cli write identity/entity name=e2e-adopt metadata=owner=external policies=default >/dev/null
adopt_id="$(openbao_cli read -format=json identity/entity/name/e2e-adopt | jq -er '.data.id')"
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-adopt
spec:
  connectionRef:
    name: openbao
  creationPolicy: Adopt
  metadata:
    owner: adopted
  policies:
    - default
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-adopt
kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-adopt -o json \
  | jq -e --arg id "${adopt_id}" '.status.id == $id' >/dev/null
assert_remote_entity "${adopt_id}" e2e-adopt adopted

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroup
metadata:
  name: e2e-group
spec:
  connectionRef:
    name: openbao
  metadata:
    owner: platform
  policies:
    - default
  driftDetectionInterval: 5s
  deletionPolicy: Delete
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupMembership
metadata:
  name: e2e-group-created-member
spec:
  groupRef:
    name: e2e-group
  entityRef:
    name: e2e-created
EOF
wait_ready openbaogroup/e2e-group
wait_ready openbaogroupmembership/e2e-group-created-member
group_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaogroup/e2e-group -o jsonpath='{.status.id}')"
[[ -n "${group_id}" ]] || { echo "created group did not publish an ID" >&2; exit 1; }
assert_remote_group_member "${group_id}" entity "${created_id}"

openbao_cli write identity/entity name=e2e-unmanaged-member policies=default >/dev/null
unmanaged_member_id="$(openbao_cli read -format=json identity/entity/name/e2e-unmanaged-member | jq -er '.data.id')"
openbao_cli write "identity/group/id/${group_id}" member_entity_ids="${created_id},${unmanaged_member_id}" member_group_ids= >/dev/null
wait_for_remote_group_member "${group_id}" entity "${unmanaged_member_id}" present
wait_for_remote_group_member "${group_id}" entity "${created_id}" present

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroup
metadata:
  name: e2e-child-group
spec:
  connectionRef:
    name: openbao
  deletionPolicy: Delete
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupMembership
metadata:
  name: e2e-child-member
spec:
  groupRef:
    name: e2e-child-group
  entityRef:
    name: e2e-adopt
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoGroupMembership
metadata:
  name: e2e-group-child-member
spec:
  groupRef:
    name: e2e-group
  memberGroupRef:
    name: e2e-child-group
EOF
wait_ready openbaogroup/e2e-child-group
wait_ready openbaogroupmembership/e2e-child-member
wait_ready openbaogroupmembership/e2e-group-child-member
child_group_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaogroup/e2e-child-group -o jsonpath='{.status.id}')"
assert_remote_group_member "${group_id}" group "${child_group_id}"
assert_remote_group_member "${child_group_id}" entity "${adopt_id}"

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaogroupmembership/e2e-group-created-member --wait=true >/dev/null
wait_for_remote_group_member "${group_id}" entity "${created_id}" absent
assert_remote_group_member "${group_id}" entity "${unmanaged_member_id}"

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaogroupmembership e2e-group-child-member e2e-child-member --wait=false >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete \
  openbaogroupmembership/e2e-group-child-member openbaogroupmembership/e2e-child-member --timeout=5m >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaogroup e2e-group e2e-child-group --wait=false >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete openbaogroup/e2e-group openbaogroup/e2e-child-group --timeout=5m >/dev/null
assert_remote_group_absent "${group_id}"
assert_remote_group_absent "${child_group_id}"
openbao_cli delete "identity/entity/id/${unmanaged_member_id}" >/dev/null

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
    name: e2e-created
  mountAccessor: ${auth_accessor}
  name: e2e-login
  driftDetectionInterval: 5s
  deletionPolicy: Delete
EOF
wait_ready openbaoentityalias/e2e-alias
alias_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentityalias/e2e-alias -o jsonpath='{.status.id}')"
[[ -n "${alias_id}" ]] || { echo "created entity alias did not publish an ID" >&2; exit 1; }
assert_remote_alias "${alias_id}" e2e-login "${auth_accessor}" "${created_id}"

openbao_cli write "identity/entity-alias/id/${alias_id}" canonical_id="${adopt_id}" >/dev/null
wait_for_remote_alias_canonical_id "${alias_id}" "${adopt_id}"
wait_for_remote_alias_canonical_id "${alias_id}" "${created_id}"
wait_for_jsonpath openbaoentityalias/e2e-alias '{.status.canonicalID}' "${created_id}"
assert_remote_alias "${alias_id}" e2e-login "${auth_accessor}" "${created_id}"

adopt_alias_id="$(openbao_cli write -format=json identity/entity-alias name=e2e-adopt-login mount_accessor="${auth_accessor}" canonical_id="${adopt_id}" | jq -er '.data.id')"
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntityAlias
metadata:
  name: e2e-adopt-alias
spec:
  connectionRef:
    name: openbao
  entityRef:
    name: e2e-adopt
  mountAccessor: ${auth_accessor}
  name: e2e-adopt-login
  creationPolicy: Adopt
  deletionPolicy: Delete
EOF
wait_ready openbaoentityalias/e2e-adopt-alias
kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentityalias/e2e-adopt-alias -o json \
  | jq -e --arg id "${adopt_alias_id}" '.status.id == $id' >/dev/null

openbao_cli write identity/entity name=e2e-conflict-alias-target policies=default >/dev/null
conflict_alias_target_id="$(openbao_cli read -format=json identity/entity/name/e2e-conflict-alias-target | jq -er '.data.id')"
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-conflict-alias-target
spec:
  connectionRef:
    name: openbao
  creationPolicy: Adopt
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-conflict-alias-target

conflict_alias_id="$(openbao_cli write -format=json identity/entity-alias name=e2e-conflict-login mount_accessor="${auth_accessor}" canonical_id="${conflict_alias_target_id}" | jq -er '.data.id')"
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntityAlias
metadata:
  name: e2e-conflict-alias
spec:
  connectionRef:
    name: openbao
  entityRef:
    name: e2e-conflict-alias-target
  mountAccessor: ${auth_accessor}
  name: e2e-conflict-login
EOF
wait_for_condition_reason openbaoentityalias/e2e-conflict-alias AliasAcquireFailed
kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentityalias/e2e-conflict-alias -o json \
  | jq -e '.status.conditions[] | select(.type == "Ready") | select(.message | contains("already exists"))' >/dev/null

openbao_cli write identity/entity name=e2e-orphan-alias-target policies=default >/dev/null
orphan_alias_target_id="$(openbao_cli read -format=json identity/entity/name/e2e-orphan-alias-target | jq -er '.data.id')"
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-orphan-alias-target
spec:
  connectionRef:
    name: openbao
  creationPolicy: Adopt
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-orphan-alias-target

orphan_alias_id="$(openbao_cli write -format=json identity/entity-alias name=e2e-orphan-login mount_accessor="${auth_accessor}" canonical_id="${orphan_alias_target_id}" | jq -er '.data.id')"
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntityAlias
metadata:
  name: e2e-orphan-alias
spec:
  connectionRef:
    name: openbao
  entityRef:
    name: e2e-orphan-alias-target
  mountAccessor: ${auth_accessor}
  name: e2e-orphan-login
  creationPolicy: Adopt
  deletionPolicy: Orphan
EOF
wait_ready openbaoentityalias/e2e-orphan-alias
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentityalias/e2e-orphan-alias --wait=true >/dev/null
assert_remote_alias "${orphan_alias_id}" e2e-orphan-login "${auth_accessor}" "${orphan_alias_target_id}"
openbao_cli delete "identity/entity-alias/id/${orphan_alias_id}" >/dev/null

openbao_cli write identity/entity name=e2e-conflict metadata=owner=external policies=default >/dev/null
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-conflict
spec:
  connectionRef:
    name: openbao
EOF
wait_for_condition_reason openbaoentity/e2e-conflict EntityAcquireFailed
kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-conflict -o json \
  | jq -e '.status.conditions[] | select(.type == "Ready") | select(.message | contains("already exists"))' >/dev/null

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-orphan
spec:
  connectionRef:
    name: openbao
  policies:
    - default
  deletionPolicy: Orphan
EOF
wait_ready openbaoentity/e2e-orphan
orphan_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-orphan -o jsonpath='{.status.id}')"
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-orphan --wait=true >/dev/null
assert_remote_entity "${orphan_id}" e2e-orphan ""
openbao_cli delete "identity/entity/id/${orphan_id}" >/dev/null

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentityalias e2e-alias e2e-adopt-alias --wait=false >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete openbaoentityalias/e2e-alias openbaoentityalias/e2e-adopt-alias --timeout=5m >/dev/null
assert_remote_alias_absent "${alias_id}"
assert_remote_alias_absent "${adopt_alias_id}"

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentityalias/e2e-conflict-alias --wait=true >/dev/null
openbao_cli delete "identity/entity-alias/id/${conflict_alias_id}" >/dev/null

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity e2e-created e2e-adopt e2e-conflict-alias-target e2e-orphan-alias-target --wait=false >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete \
  openbaoentity/e2e-created openbaoentity/e2e-adopt \
  openbaoentity/e2e-conflict-alias-target openbaoentity/e2e-orphan-alias-target --timeout=5m >/dev/null
assert_remote_absent "${created_id}"
assert_remote_absent "${adopt_id}"
assert_remote_absent "${conflict_alias_target_id}"
assert_remote_absent "${orphan_alias_target_id}"

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-conflict --wait=true >/dev/null
openbao_cli delete identity/entity/name/e2e-conflict >/dev/null

cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: e2e-missing-connection
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  tokenSecretRef:
    name: ${OPENBAO_TOKEN_SECRET}
    key: ${OPENBAO_TOKEN_KEY}
EOF
wait_ready openbaoconnection/e2e-missing-connection
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-missing-connection
spec:
  connectionRef:
    name: e2e-missing-connection
  policies:
    - default
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-missing-connection
missing_connection_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-missing-connection -o jsonpath='{.status.id}')"
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoconnection/e2e-missing-connection --wait=true >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-missing-connection --wait=true >/dev/null
assert_remote_entity "${missing_connection_id}" e2e-missing-connection ""
openbao_cli delete "identity/entity/id/${missing_connection_id}" >/dev/null

kubectl_cmd -n "${TEST_NAMESPACE}" get secret "${OPENBAO_TOKEN_SECRET}" -o json \
  | jq 'del(.metadata.namespace, .metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.managedFields) | .metadata.name = "e2e-missing-secret-token"' \
  | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: e2e-missing-secret
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  tokenSecretRef:
    name: e2e-missing-secret-token
    key: ${OPENBAO_TOKEN_KEY}
EOF
wait_ready openbaoconnection/e2e-missing-secret
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-missing-secret
spec:
  connectionRef:
    name: e2e-missing-secret
  policies:
    - default
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-missing-secret
missing_secret_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-missing-secret -o jsonpath='{.status.id}')"
kubectl_cmd -n "${TEST_NAMESPACE}" delete secret/e2e-missing-secret-token --wait=true >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-missing-secret --wait=true >/dev/null
assert_remote_entity "${missing_secret_id}" e2e-missing-secret ""
openbao_cli delete "identity/entity/id/${missing_secret_id}" >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoconnection/e2e-missing-secret --wait=true >/dev/null

for namespace in e2e-namespace-a e2e-namespace-b; do
  openbao_cli namespace create "${namespace}" >/dev/null
done
cat <<EOF | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: e2e-namespace-a
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  namespace: e2e-namespace-a
  tokenSecretRef:
    name: ${OPENBAO_TOKEN_SECRET}
    key: ${OPENBAO_TOKEN_KEY}
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: e2e-namespace-b
spec:
  address: http://openbao.${OPENBAO_NAMESPACE}.svc.cluster.local:8200
  namespace: e2e-namespace-b
  tokenSecretRef:
    name: ${OPENBAO_TOKEN_SECRET}
    key: ${OPENBAO_TOKEN_KEY}
EOF
wait_ready openbaoconnection/e2e-namespace-a
wait_ready openbaoconnection/e2e-namespace-b
cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-namespace-a-entity
spec:
  connectionRef:
    name: e2e-namespace-a
  policies:
    - default
  deletionPolicy: Delete
---
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoEntity
metadata:
  name: e2e-namespace-b-entity
spec:
  connectionRef:
    name: e2e-namespace-b
  policies:
    - default
  deletionPolicy: Delete
EOF
wait_ready openbaoentity/e2e-namespace-a-entity
wait_ready openbaoentity/e2e-namespace-b-entity
namespace_a_entity_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-namespace-a-entity -o jsonpath='{.status.id}')"
namespace_b_entity_id="$(kubectl_cmd -n "${TEST_NAMESPACE}" get openbaoentity/e2e-namespace-b-entity -o jsonpath='{.status.id}')"
[[ "${namespace_a_entity_id}" != "${namespace_b_entity_id}" ]] || {
  echo "namespace entities unexpectedly share an ID" >&2
  exit 1
}
assert_remote_entity_in_namespace e2e-namespace-a "${namespace_a_entity_id}" e2e-namespace-a-entity
assert_remote_entity_in_namespace e2e-namespace-b "${namespace_b_entity_id}" e2e-namespace-b-entity
if remote_entity_in_namespace e2e-namespace-b "${namespace_a_entity_id}" >/dev/null 2>&1; then
  echo "namespace B can read namespace A's entity" >&2
  exit 1
fi
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-namespace-a-entity openbaoentity/e2e-namespace-b-entity --wait=false >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete \
  openbaoentity/e2e-namespace-a-entity openbaoentity/e2e-namespace-b-entity --timeout=5m >/dev/null
assert_remote_entity_absent_in_namespace e2e-namespace-a "${namespace_a_entity_id}"
assert_remote_entity_absent_in_namespace e2e-namespace-b "${namespace_b_entity_id}"
kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoconnection/e2e-namespace-a openbaoconnection/e2e-namespace-b --wait=true >/dev/null
for namespace in e2e-namespace-a e2e-namespace-b; do
  openbao_cli namespace delete "${namespace}" >/dev/null
done

echo "Kind/OpenBao integration scenarios passed"
