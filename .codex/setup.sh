#!/bin/bash
set -euo pipefail

required_go_version="1.25.9"
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

if ! command -v golangci-lint >/dev/null 2>&1; then
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin" "${golangci_lint_version}"
fi

echo "Setup complete: $(go version)"
