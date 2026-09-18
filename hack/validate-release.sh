#!/usr/bin/env bash
set -euo pipefail

release_tag=${RELEASE_TAG:-}
chart_yaml=${CHART_FILE:-charts/openbao-entity-operator/Chart.yaml}

if [[ -z "${release_tag}" ]]; then
  echo "RELEASE_TAG is required" >&2
  exit 1
fi

if [[ ! -f "${chart_yaml}" ]]; then
  echo "Chart metadata file not found: ${chart_yaml}" >&2
  exit 1
fi

semver_re='^[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$'
chart_version="$(awk '$1 == "version:" { value=$2; gsub(/^"/, "", value); gsub(/"$/, "", value); print value; exit }' "${chart_yaml}")"
app_version="$(awk '$1 == "appVersion:" { value=$2; gsub(/^"/, "", value); gsub(/"$/, "", value); print value; exit }' "${chart_yaml}")"

if [[ ! "${chart_version}" =~ ${semver_re} ]]; then
  echo "Chart.yaml version must use X.Y.Z or X.Y.Z-rc.N, got: ${chart_version}" >&2
  exit 1
fi
if [[ ! "${app_version}" =~ ${semver_re} ]]; then
  echo "Chart.yaml appVersion must use X.Y.Z or X.Y.Z-rc.N, got: ${app_version}" >&2
  exit 1
fi

case "${release_tag}" in
  v*)
    operator_version=${release_tag#v}
    if [[ "${chart_version}" != "${operator_version}" || "${app_version}" != "${operator_version}" ]]; then
      echo "Paired release tag ${release_tag} requires matching Chart.yaml version and appVersion" >&2
      exit 1
    fi
    release_kind=operator
    make_latest=true
    ;;
  chart-v*)
    chart_tag_version=${release_tag#chart-v}
    if [[ "${chart_version}" != "${chart_tag_version}" ]]; then
      echo "Chart.yaml version ${chart_version} does not match chart tag ${release_tag}" >&2
      exit 1
    fi
    operator_version=${app_version}
    release_kind=chart
    make_latest=false
    ;;
  *)
    echo "Release tag must start with v or chart-v: ${release_tag}" >&2
    exit 1
    ;;
esac

prerelease=false
if [[ "${release_tag}" == *-rc.* ]]; then
  prerelease=true
  make_latest=false
fi

printf 'release_kind=%s\n' "${release_kind}"
printf 'chart_version=%s\n' "${chart_version}"
printf 'operator_version=%s\n' "${operator_version}"
printf 'prerelease=%s\n' "${prerelease}"
printf 'make_latest=%s\n' "${make_latest}"
