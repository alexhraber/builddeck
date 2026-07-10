#!/usr/bin/env bash
# Ensures Go is available, then exec's the supplied command.
set -euo pipefail

GO_VERSION="1.25.8"
INSTALL_DIR="/tmp/go-install"

if ! command -v go &>/dev/null && [[ ! -x "${INSTALL_DIR}/go/bin/go" ]]; then
  echo "+++ Downloading Go ${GO_VERSION}"
  mkdir -p "${INSTALL_DIR}"
  curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C "${INSTALL_DIR}" -xz
fi

export PATH="${INSTALL_DIR}/go/bin:${PATH}"
echo "go $(go version)"
echo "+++ $*"

exec "$@"
