#!/bin/bash
set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required but was not found on PATH." >&2
  exit 1
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin" v1.57.2
fi

echo "Setup complete: $(go version)"
