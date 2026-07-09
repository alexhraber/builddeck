#!/usr/bin/env bash
set -euo pipefail

GO_VERSION="1.25.8"
CACHE_KEY="go-${GO_VERSION}"
INSTALL_DIR="/tmp/go-install"

if ! command -v go &>/dev/null && [[ ! -x "${INSTALL_DIR}/go/bin/go" ]]; then
  if ! buildkite-agent cache get "${CACHE_KEY}" "${INSTALL_DIR}/go" 2>/dev/null; then
    echo "+++ Downloading Go ${GO_VERSION}"
    mkdir -p "${INSTALL_DIR}"
    curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C "${INSTALL_DIR}" -xz
    buildkite-agent cache set "${CACHE_KEY}" "${INSTALL_DIR}/go"
  fi
fi

export PATH="${INSTALL_DIR}/go/bin:${PATH}"
echo "go $(go version)"

exec "$@"
