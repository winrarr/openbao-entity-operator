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
operator_deployment=${OPERATOR_DEPLOYMENT:-openbao-entity-operator-controller-manager}
KEEP_TEST_RESOURCES=${KEEP_TEST_RESOURCES:-false}

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

cleanup() {
	local exit_code=$?
	if [[ "${KEEP_TEST_RESOURCES}" == true || "${exit_code}" -ne 0 ]]; then
		echo "Keeping ${TEST_NAMESPACE} for inspection (exit ${exit_code})" >&2
		exit "${exit_code}"
	fi
	kubectl_cmd delete namespace "${TEST_NAMESPACE}" --ignore-not-found --wait=true --timeout=5m >/dev/null 2>&1 || true
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

remote_entity() {
  local id="$1"
  openbao_cli read -format=json "identity/entity/id/${id}"
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

assert_remote_absent() {
  local id="$1"
  if remote_entity "${id}" >/dev/null 2>&1; then
    echo "OpenBao entity ${id} still exists" >&2
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
  tokenSecretRef:
    name: ${OPENBAO_TOKEN_SECRET}
    key: ${OPENBAO_TOKEN_KEY}
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
echo "Kind/OpenBao integration scenarios passed"
