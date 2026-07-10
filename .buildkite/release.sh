#!/usr/bin/env bash
# Creates a GitHub Release with binary, checksum, and generated release notes.
# Requires GITHUB_TOKEN env var with repo scope.
# Usage: release.sh <version>
set -euo pipefail

VERSION="${1:-${BUILDKITE_TAG}}"
if [[ -z "${VERSION}" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "+++ Creating GitHub Release ${VERSION}"

# Download artifacts
buildkite-agent artifact download builddeck . --step build
buildkite-agent artifact download builddeck.sha256 . --step checksum 2>/dev/null || true

# Generate release notes from conventional commits since last tag
LAST_TAG=$(git tag -l 'v*' --sort=-v:refname | head -2 | tail -1)
if [[ -z "${LAST_TAG}" ]]; then
  LAST_TAG="v0.0.0"
fi

echo "Generating release notes from ${LAST_TAG}..HEAD"
RELEASE_NOTES=$(git log "${LAST_TAG}..HEAD" --pretty=format:"- %s" --reverse 2>/dev/null || echo "Initial release")

# Categorize commits
FEATURES=$(echo "$RELEASE_NOTES" | grep -E '^feat' | sed 's/^feat[\(!]*: */  - /' | sed 's/^feat!:/  - **BREAKING**: /')
FIXES=$(echo "$RELEASE_NOTES" | grep -E '^fix' | sed 's/^fix[\(!]*: */  - /')
CHANGES=$(echo "$RELEASE_NOTES" | grep -vE '^(feat|fix)' | sed 's/^/  - /')

# Build release body
BODY="## ${VERSION}

"

if [[ -n "${FEATURES}" ]]; then
  BODY="${BODY}
### Features
${FEATURES}
"
fi

if [[ -n "${FIXES}" ]]; then
  BODY="${BODY}
### Bug Fixes
${FIXES}
"
fi

if [[ -n "${CHANGES}" && "${CHANGES}" != *"  - "* ]]; then
  BODY="${BODY}
### Other Changes
${CHANGES}
"
fi

BODY="${BODY}

---

**Full Changelog**: https://github.com/alexhraber/builddeck/compare/${LAST_TAG}...${VERSION}

**Assets**:
- \`builddeck\` — Linux amd64 binary
- \`builddeck.sha256\` — SHA256 checksum"

# Create or update release
if gh release view "${VERSION}" &>/dev/null; then
  echo "Release ${VERSION} exists, updating..."
  gh release edit "${VERSION}" \
    --title "builddeck ${VERSION}" \
    --notes "${BODY}" \
    builddeck builddeck.sha256 --clobber 2>/dev/null || \
    gh release upload "${VERSION}" builddeck builddeck.sha256 --clobber
else
  echo "Creating release ${VERSION}..."
  gh release create "${VERSION}" \
    --title "builddeck ${VERSION}" \
    --notes "${BODY}" \
    builddeck builddeck.sha256
fi

echo "Release ${VERSION} complete"