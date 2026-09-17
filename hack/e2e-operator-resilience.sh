#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KUBE_CONTEXT=${KUBE_CONTEXT:-kind-openbao-entity-operator}
OPENBAO_IMAGE=${OPENBAO_IMAGE:-openbao/openbao:2.6.2}
PERSISTENT_OPENBAO_NAMESPACE=${PERSISTENT_OPENBAO_NAMESPACE:-openbao-entity-operator-persistent}
PERSISTENT_OPENBAO_DEPLOYMENT=${PERSISTENT_OPENBAO_DEPLOYMENT:-openbao-persistent}
PERSISTENT_OPENBAO_SERVICE=${PERSISTENT_OPENBAO_SERVICE:-openbao-persistent}
PERSISTENT_OPENBAO_TLS_SECRET=${PERSISTENT_OPENBAO_TLS_SECRET:-openbao-persistent-tls}
PERSISTENT_OPENBAO_BOOTSTRAP_SECRET=${PERSISTENT_OPENBAO_BOOTSTRAP_SECRET:-openbao-persistent-bootstrap}
PERSISTENT_E2E_NAMESPACE=${PERSISTENT_E2E_NAMESPACE:-openbao-entity-operator-resilience}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-openbao-entity-operator-system}
OPERATOR_DEPLOYMENT=${OPERATOR_DEPLOYMENT:-openbao-entity-operator}
OPERATOR_POLICY=${OPERATOR_POLICY:-e2e-operator-policy}
OPERATOR_TOKEN_SECRET=${OPERATOR_TOKEN_SECRET:-e2e-operator-token}
OPENBAO_CA_SECRET=${OPENBAO_CA_SECRET:-openbao-persistent-ca}
KEEP_TEST_RESOURCES=${KEEP_TEST_RESOURCES:-false}
MANIFEST=${MANIFEST:-hack/kind-openbao-persistent.yaml}

kubectl_cmd() {
  "${KUBECTL}" --context="${KUBE_CONTEXT}" "$@"
}

persistent_cli() {
  kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" exec "deployment/${PERSISTENT_OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR="https://${PERSISTENT_OPENBAO_SERVICE}.${PERSISTENT_OPENBAO_NAMESPACE}.svc.cluster.local:8200" \
    BAO_CACERT=/openbao/tls/ca.crt BAO_TOKEN="${root_token}" bao "$@"
}

persistent_cli_stdin() {
  kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" exec -i "deployment/${PERSISTENT_OPENBAO_DEPLOYMENT}" -- \
    env BAO_ADDR="https://${PERSISTENT_OPENBAO_SERVICE}.${PERSISTENT_OPENBAO_NAMESPACE}.svc.cluster.local:8200" \
    BAO_CACERT=/openbao/tls/ca.crt BAO_TOKEN="${root_token}" bao "$@"
}

wait_for_jsonpath() {
  local namespace="$1"
  local resource="$2"
  local jsonpath="$3"
  local expected="$4"
  local actual
  for _ in {1..150}; do
    actual="$(kubectl_cmd -n "${namespace}" get "${resource}" -o "jsonpath=${jsonpath}" 2>/dev/null || true)"
    if [[ "${actual}" == "${expected}" ]]; then
      return 0
    fi
    sleep 2
  done
  echo "${resource} in ${namespace} did not reach ${jsonpath}=${expected}" >&2
  kubectl_cmd -n "${namespace}" get "${resource}" -o yaml >&2 || true
  return 1
}

wait_ready() {
  local namespace="$1"
  local resource="$2"
  kubectl_cmd -n "${namespace}" wait \
    --for='jsonpath={.status.conditions[?(@.type=="Ready")].status}=True' \
    "${resource}" --timeout=5m >/dev/null
}

wait_unsealed() {
  for attempt in {1..150}; do
    if unseal_fixture >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "persistent OpenBao did not become unsealed" >&2
  persistent_cli status -format=json >&2 || true
  return 1
}

read_status() {
  persistent_cli status -format=json 2>/dev/null | sed '/^command terminated with exit code /d' || true
}

unseal_fixture() {
  local output sealed
  output="$(persistent_cli operator unseal "${unseal_key}" 2>/dev/null || true)"
  sealed="$(awk '$1 == "Sealed" { print $2 }' <<<"${output}" | tail -1)"
  if [[ "${sealed}" != false ]]; then
    echo "resilience: unseal response did not report Sealed=false (${sealed:-unknown})" >&2
    return 1
  fi
}

wait_for_status() {
  local status_json attempt
  for attempt in {1..150}; do
    status_json="$(read_status)"
    if jq -e 'has("initialized") and has("sealed")' <<<"${status_json}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "persistent OpenBao API did not become ready" >&2
  return 1
}

create_tls_material() {
  local tls_dir="$1"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "${tls_dir}/ca.key" -out "${tls_dir}/ca.crt" \
    -subj '/CN=openbao-entity-operator resilience test CA' -days 2 >/dev/null 2>&1
  openssl req -newkey rsa:2048 -nodes \
    -keyout "${tls_dir}/tls.key" -out "${tls_dir}/tls.csr" \
    -subj '/CN=openbao-persistent' >/dev/null 2>&1
  openssl x509 -req -in "${tls_dir}/tls.csr" \
    -CA "${tls_dir}/ca.crt" -CAkey "${tls_dir}/ca.key" -CAcreateserial \
    -out "${tls_dir}/tls.crt" -days 2 -sha256 \
    -extfile <(printf 'subjectAltName=DNS:%s.%s.svc.cluster.local,DNS:%s.%s.svc,DNS:%s,IP:127.0.0.1' \
      "${PERSISTENT_OPENBAO_SERVICE}" "${PERSISTENT_OPENBAO_NAMESPACE}" \
      "${PERSISTENT_OPENBAO_SERVICE}" "${PERSISTENT_OPENBAO_NAMESPACE}" \
      "${PERSISTENT_OPENBAO_SERVICE}") >/dev/null 2>&1
}

apply_tls_secret() {
  local tls_dir="$1"
  kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" create secret generic "${PERSISTENT_OPENBAO_TLS_SECRET}" \
    --from-file=tls.crt="${tls_dir}/tls.crt" \
    --from-file=tls.key="${tls_dir}/tls.key" \
    --from-file=ca.crt="${tls_dir}/ca.crt" \
    --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

read_bootstrap_secret() {
  root_token="$(kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" get secret "${PERSISTENT_OPENBAO_BOOTSTRAP_SECRET}" -o jsonpath='{.data.root-token}' | base64 --decode)"
  unseal_key="$(kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" get secret "${PERSISTENT_OPENBAO_BOOTSTRAP_SECRET}" -o jsonpath='{.data.unseal-key}' | base64 --decode)"
}

bootstrap_persistent_openbao() {
  local init_output status_json initialized sealed
  if kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" get secret "${PERSISTENT_OPENBAO_BOOTSTRAP_SECRET}" >/dev/null 2>&1; then
    read_bootstrap_secret
  else
    init_output="$(persistent_cli operator init -key-shares=1 -key-threshold=1 -format=json)"
    unseal_key="$(jq -er '.unseal_keys_b64[0]' <<<"${init_output}")"
    root_token="$(jq -er '.root_token' <<<"${init_output}")"
    kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" create secret generic "${PERSISTENT_OPENBAO_BOOTSTRAP_SECRET}" \
      --from-literal=root-token="${root_token}" --from-literal=unseal-key="${unseal_key}" \
      --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
  fi

  wait_for_status
  status_json="$(read_status)"
  initialized="$(jq -er '.initialized' <<<"${status_json}")"
  sealed="$(jq -er '.sealed' <<<"${status_json}")"
  if [[ "${initialized}" != true ]]; then
    echo "persistent OpenBao is not initialized after bootstrap" >&2
    return 1
  fi
  if [[ "${sealed}" == true ]]; then
    unseal_fixture || true
  fi
  wait_unsealed
}

write_operator_policy() {
  persistent_cli_stdin policy write "${OPERATOR_POLICY}" - >/dev/null <<'EOF'
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
EOF
}

create_token() {
  persistent_cli token create -format=json -policy="${OPERATOR_POLICY}" -period=1h
}

apply_operator_token_secret() {
  local token="$1"
  kubectl_cmd -n "${PERSISTENT_E2E_NAMESPACE}" create secret generic "${OPERATOR_TOKEN_SECRET}" \
    --from-literal=token="${token}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

apply_connection() {
  kubectl_cmd -n "${PERSISTENT_E2E_NAMESPACE}" apply -f - >/dev/null <<EOF
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoConnection
metadata:
  name: persistent
spec:
  address: https://${PERSISTENT_OPENBAO_SERVICE}.${PERSISTENT_OPENBAO_NAMESPACE}.svc.cluster.local:8200
  tokenSecretRef:
    name: ${OPERATOR_TOKEN_SECRET}
  caBundleSecretRef:
    name: ${OPENBAO_CA_SECRET}
EOF
}

apply_policy() {
  local name="$1"
  local path="$2"
  kubectl_cmd -n "${PERSISTENT_E2E_NAMESPACE}" apply -f - >/dev/null <<EOF
apiVersion: openbao.openbao-operator.io/v1alpha1
kind: OpenBaoPolicy
metadata:
  name: ${name}
spec:
  connectionRef:
    name: persistent
  deletionPolicy: Delete
  rules: |
    path "${path}" {
      capabilities = ["read"]
    }
EOF
  wait_ready "${PERSISTENT_E2E_NAMESPACE}" "openbaopolicy/${name}"
}

assert_remote_policy() {
  local name="$1"
  local path="$2"
  persistent_cli read -format=json "sys/policies/acl/${name}" | jq -e \
    --arg name "${name}" --arg path "${path}" \
    '.data.name == $name and (.data.policy | contains(("path \"" + $path + "\"")))' >/dev/null
}

revoke_and_wait_for_connection_failure() {
  local token="$1"
  persistent_cli token revoke "${token}" >/dev/null
  kubectl_cmd -n "${PERSISTENT_E2E_NAMESPACE}" annotate openbaoconnection/persistent \
    resilience.openbao-entity-operator.io/reconcile="$(date +%s)" --overwrite >/dev/null
  wait_for_jsonpath "${PERSISTENT_E2E_NAMESPACE}" openbaoconnection/persistent \
    '{.status.conditions[?(@.type=="Ready")].status}' False
}

cleanup_on_exit() {
  local exit_code=$?
  if [[ -n "${tls_dir:-}" && -d "${tls_dir}" ]]; then
    rm -rf -- "${tls_dir}"
  fi
  if [[ "${KEEP_TEST_RESOURCES}" == true || "${exit_code}" -ne 0 ]]; then
    echo "Keeping ${PERSISTENT_E2E_NAMESPACE} and ${PERSISTENT_OPENBAO_NAMESPACE} for inspection (exit ${exit_code})" >&2
  fi
  exit "${exit_code}"
}
trap cleanup_on_exit EXIT

tls_dir="$(mktemp -d)"
root_token=''
unseal_key=''
create_tls_material "${tls_dir}"

echo "resilience: applying TLS fixture"
kubectl_cmd cluster-info >/dev/null
kubectl_cmd create namespace "${PERSISTENT_OPENBAO_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
apply_tls_secret "${tls_dir}"
PERSISTENT_OPENBAO_NAMESPACE="${PERSISTENT_OPENBAO_NAMESPACE}" \
PERSISTENT_OPENBAO_DEPLOYMENT="${PERSISTENT_OPENBAO_DEPLOYMENT}" \
PERSISTENT_OPENBAO_SERVICE="${PERSISTENT_OPENBAO_SERVICE}" \
PERSISTENT_OPENBAO_TLS_SECRET="${PERSISTENT_OPENBAO_TLS_SECRET}" \
OPENBAO_IMAGE="${OPENBAO_IMAGE}" \
  envsubst < "${MANIFEST}" | kubectl_cmd apply -f - >/dev/null
kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" rollout status \
  "deployment/${PERSISTENT_OPENBAO_DEPLOYMENT}" --timeout=5m >/dev/null
kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" delete pod \
  -l app.kubernetes.io/name=openbao-persistent --wait=true >/dev/null
kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" rollout status \
  "deployment/${PERSISTENT_OPENBAO_DEPLOYMENT}" --timeout=5m >/dev/null

bootstrap_persistent_openbao
echo "resilience: bootstrapped TLS fixture"
write_operator_policy
echo "resilience: wrote operator policy"
token_a_json="$(create_token)"
token_a="$(jq -er '.auth.client_token' <<<"${token_a_json}")"
jq -e --arg policy "${OPERATOR_POLICY}" '(.auth.policies // []) | index("root") == null and index($policy) != null' <<<"${token_a_json}" >/dev/null

kubectl_cmd -n "${PERSISTENT_E2E_NAMESPACE}" create namespace "${PERSISTENT_E2E_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
echo "resilience: applying operator credentials"
kubectl_cmd -n "${PERSISTENT_E2E_NAMESPACE}" create secret generic "${OPENBAO_CA_SECRET}" \
  --from-file=ca.crt="${tls_dir}/ca.crt" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
apply_operator_token_secret "${token_a}"
apply_connection
echo "resilience: waiting for connection"
wait_ready "${PERSISTENT_E2E_NAMESPACE}" openbaoconnection/persistent

echo "resilience: testing initial reconciliation"
apply_policy resilience-before-restart identity/entity/name/resilience-before-restart
assert_remote_policy resilience-before-restart identity/entity/name/resilience-before-restart

kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" delete pod \
  -l app.kubernetes.io/name=openbao-persistent --wait=true >/dev/null
kubectl_cmd -n "${PERSISTENT_OPENBAO_NAMESPACE}" rollout status \
  "deployment/${PERSISTENT_OPENBAO_DEPLOYMENT}" --timeout=5m >/dev/null
wait_unsealed
echo "resilience: testing restart recovery"
apply_policy resilience-after-restart identity/entity/name/resilience-after-restart
assert_remote_policy resilience-after-restart identity/entity/name/resilience-after-restart

revoke_and_wait_for_connection_failure "${token_a}"
echo "resilience: rotating credentials"
token_b_json="$(create_token)"
token_b="$(jq -er '.auth.client_token' <<<"${token_b_json}")"
jq -e --arg policy "${OPERATOR_POLICY}" '(.auth.policies // []) | index("root") == null and index($policy) != null' <<<"${token_b_json}" >/dev/null
apply_operator_token_secret "${token_b}"
wait_ready "${PERSISTENT_E2E_NAMESPACE}" openbaoconnection/persistent
echo "resilience: testing post-rotation reconciliation"
apply_policy resilience-after-rotation identity/entity/name/resilience-after-rotation
assert_remote_policy resilience-after-rotation identity/entity/name/resilience-after-rotation

echo "Kind/OpenBao operator resilience scenarios passed"
