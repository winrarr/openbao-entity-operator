#!/usr/bin/env bash
set -euo pipefail

address=${OPENBAO_ADDR:-http://127.0.0.1:8200}
token=${OPENBAO_TOKEN:?OPENBAO_TOKEN must be set}
spec_path=${OPENBAO_OPENAPI_SPEC:-hack/openbao-openapi.json}
temporary_path="${spec_path}.tmp.$$"

mkdir -p "$(dirname "$spec_path")"
curl --fail --silent --show-error \
  --header "X-Vault-Token: ${token}" \
  --header 'Content-Type: application/json' \
  --data '{"generic_mount_paths":true}' \
  "${address%/}/v1/sys/internal/specs/openapi" > "$temporary_path"
jq -e '.openapi and .info and .paths' "$temporary_path" >/dev/null
mv "$temporary_path" "$spec_path"
echo "Updated OpenBao OpenAPI reference: $spec_path"
