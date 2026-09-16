#!/usr/bin/env bash
set -euo pipefail

spec_path=${OPENBAO_OPENAPI_SPEC:-hack/openbao-openapi.json}

test -s "$spec_path" || { echo "OpenBao OpenAPI reference is missing: $spec_path" >&2; exit 1; }
jq -e '.openapi == "3.0.2" and .info.title == "OpenBao API" and (.paths | type == "object")' "$spec_path" >/dev/null
jq -e '(.paths | has("/identity/entity")) and (.paths | has("/identity/entity/id/{id}")) and (.paths | has("/identity/entity/name/{name}"))' "$spec_path" >/dev/null
jq -e '(.paths | has("/identity/entity-alias")) and (.paths | has("/identity/entity-alias/id")) and (.paths | has("/identity/entity-alias/id/{id}"))' "$spec_path" >/dev/null
echo "Validated OpenBao OpenAPI reference: $spec_path"
