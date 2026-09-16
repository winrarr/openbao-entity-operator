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

cat <<'EOF' | kubectl_cmd -n "${TEST_NAMESPACE}" apply -f - >/dev/null
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

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity e2e-created e2e-adopt --wait=false >/dev/null
kubectl_cmd -n "${TEST_NAMESPACE}" wait --for=delete openbaoentity/e2e-created openbaoentity/e2e-adopt --timeout=5m >/dev/null
assert_remote_absent "${created_id}"
assert_remote_absent "${adopt_id}"

kubectl_cmd -n "${TEST_NAMESPACE}" delete openbaoentity/e2e-conflict --wait=true >/dev/null
openbao_cli delete identity/entity/name/e2e-conflict >/dev/null
echo "Kind/OpenBao integration scenarios passed"
