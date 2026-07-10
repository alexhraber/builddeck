#!/usr/bin/env bash
# Uploads builddeck binary to existing GitHub release.
# Requires GITHUB_TOKEN env var with repo scope.
# Usage: release.sh <version>
set -euo pipefail

VERSION="${1:-${BUILDKITE_TAG}}"
if [[ -z "${VERSION}" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "+++ Uploading builddeck binary to release ${VERSION}"

buildkite-agent artifact download builddeck . --step build

gh release upload "${VERSION}" builddeck --clobber
echo "Release ${VERSION} updated with new binary"