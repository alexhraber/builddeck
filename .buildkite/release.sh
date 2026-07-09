#!/usr/bin/env bash
# Creates a GitHub release with the builddeck binary attached.
# Requires GITHUB_TOKEN env var with repo scope.
set -euo pipefail

VERSION="${BUILDKITE_TAG:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="v$(date +%Y%m%d).${BUILDKITE_BUILD_NUMBER}"
fi

echo "+++ Creating GitHub release ${VERSION}"

buildkite-agent artifact download builddeck . --step build

if ! gh release view "${VERSION}" &>/dev/null; then
  gh release create "${VERSION}" \
    --title "builddeck ${VERSION}" \
    --notes "$(cat <<EOF
## builddeck ${VERSION}

Binary for Linux amd64. Download and run:

\`\`\`
chmod +x builddeck
export BUILDKITE_API_TOKEN="your-token"
./builddeck
\`\`\`
EOF
)" \
    builddeck
  echo "Release ${VERSION} created"
else
  gh release upload "${VERSION}" builddeck --clobber
  echo "Release ${VERSION} updated with new binary"
fi
