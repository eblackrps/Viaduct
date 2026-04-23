#!/bin/bash
set -euo pipefail

required_go_version="1.25.9"
required_node_version="20.19.0"
golangci_lint_version="${GOLANGCI_LINT_VERSION:-v2.11.4}"

version_gte() {
  local actual="$1"
  local minimum="$2"
  [[ "$(printf '%s\n%s\n' "$minimum" "$actual" | sort -V | head -n1)" == "$minimum" ]]
}

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required but was not found on PATH." >&2
  exit 1
fi

go_version_output="$(go version)"
installed_go_version="$(awk '{print $3}' <<<"$go_version_output")"
installed_go_version="${installed_go_version#go}"
if [[ ! "$installed_go_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
  echo "Unable to parse Go version from: $go_version_output" >&2
  echo "Go ${required_go_version}+ is required for this repository." >&2
  exit 1
fi
if [[ "$installed_go_version" =~ ^[0-9]+\.[0-9]+$ ]]; then
  installed_go_version="${installed_go_version}.0"
fi

if ! version_gte "$installed_go_version" "$required_go_version"; then
  echo "Go ${required_go_version}+ is required for this repository, but found go${installed_go_version}." >&2
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "Node.js ${required_node_version}+ is required for dashboard and release-owner checks, but node was not found on PATH." >&2
  exit 1
fi
if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required for dashboard and Playwright checks, but was not found on PATH." >&2
  exit 1
fi

node_version_output="$(node --version)"
installed_node_version="${node_version_output#v}"
if [[ ! "$installed_node_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
  echo "Unable to parse Node.js version from: $node_version_output" >&2
  echo "Node.js ${required_node_version}+ is required for this repository." >&2
  exit 1
fi
if [[ "$installed_node_version" =~ ^[0-9]+\.[0-9]+$ ]]; then
  installed_node_version="${installed_node_version}.0"
fi
if ! version_gte "$installed_node_version" "$required_node_version"; then
  echo "Node.js ${required_node_version}+ is required for this repository, but found ${node_version_output}." >&2
  exit 1
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin" "${golangci_lint_version}"
fi

if [[ "${VIADUCT_SKIP_WEB_SETUP:-0}" != "1" ]] && [[ -f web/package-lock.json ]]; then
  (
    cd web
    npm ci
    if [[ "${VIADUCT_SKIP_PLAYWRIGHT_INSTALL:-0}" != "1" ]]; then
      if [[ "$(uname -s)" == "Linux" ]]; then
        npx playwright install --with-deps chromium
      else
        npx playwright install chromium
      fi
    fi
  )
fi

echo "Setup complete: $(go version), node ${node_version_output}"
