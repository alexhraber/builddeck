#!/usr/bin/env bash
# Installs Go if missing, restores from build artifact if available,
# then exec's the supplied command with Go on PATH.
set -euo pipefail

GO_VERSION="1.25.8"
INSTALL_DIR="/tmp/go-install"
ARCHIVE="/tmp/go-${GO_VERSION}.tar.gz"

if ! command -v go &>/dev/null && [[ ! -x "${INSTALL_DIR}/go/bin/go" ]]; then
  mkdir -p "${INSTALL_DIR}"

  if ! buildkite-agent artifact download "go-${GO_VERSION}.tar.gz" /tmp 2>/dev/null; then
    echo "+++ Downloading Go ${GO_VERSION}"
    curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o "${ARCHIVE}"
    buildkite-agent artifact upload "${ARCHIVE}"
  fi

  tar xzf "${ARCHIVE}" -C "${INSTALL_DIR}"
fi

export PATH="${INSTALL_DIR}/go/bin:${PATH}"
echo "go $(go version)"

exec "$@"
