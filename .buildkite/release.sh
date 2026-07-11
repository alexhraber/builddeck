#!/usr/bin/env bash
# Creates a GitHub Release linked to git tag with all artifacts and generated notes.
# Expects builddeck and builddeck.sha256 in the current directory (same-agent
# after release_build step or tag-push triggered pipeline).
# Usage: release.sh <version>
set -euo pipefail

VERSION="${1:-${BUILDKITE_TAG}}"
if [[ -z "${VERSION}" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "+++ Creating GitHub Release ${VERSION}"

cd "$(dirname "$0")/.."

echo "=== Files in workspace ==="
ls -la

# Generate release notes
LAST_TAG=$(git tag -l 'v*' --sort=-v:refname | head -2 | tail -1)
if [[ -z "${LAST_TAG}" ]]; then
  LAST_TAG="v0.0.0"
fi

echo "Generating release notes from ${LAST_TAG}..HEAD"
RELEASE_NOTES=$(git log "${LAST_TAG}..HEAD" --pretty=format:"- %s" --reverse 2>/dev/null || echo "Initial release")

FEATURES=$(echo "$RELEASE_NOTES" | grep -E '^feat' | sed 's/^feat[(!]*: */  - /' | sed 's/^feat!:/  - **BREAKING**: /')
FIXES=$(echo "$RELEASE_NOTES" | grep -E '^fix' | sed 's/^fix[(!]*: */  - /')
CHANGES=$(echo "$RELEASE_NOTES" | grep -vE '^(feat|fix)' | sed 's/^/  - /')

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

ASSETS=()
for f in builddeck*; do
  [[ -f "$f" ]] && ASSETS+=("$f")
done

echo "=== Assets to upload: ${ASSETS[@]} ==="

if gh release view "${VERSION}" &>/dev/null; then
  echo "Release ${VERSION} exists, updating..."
  gh release edit "${VERSION}" \
    --title "builddeck ${VERSION}" \
    --notes "${BODY}"
else
  echo "Creating release ${VERSION}..."
  gh release create "${VERSION}" \
    --title "builddeck ${VERSION}" \
    --notes "${BODY}"
fi

echo "Uploading assets: ${ASSETS[@]}"
gh release upload "${VERSION}" "${ASSETS[@]}" --clobber

echo "Release ${VERSION} complete"
