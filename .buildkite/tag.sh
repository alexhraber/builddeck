#!/usr/bin/env bash
# Determines semver bump from conventional commits and creates/pushes tag
# Uploads version.txt artifact for build step
set -euo pipefail

echo "--- Determining version bump from conventional commits"

LATEST=$(git tag -l 'v*' --sort=-v:refname | head -1 || echo "v0.0.0")
if [[ "$LATEST" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  MAJOR="${BASH_REMATCH[1]}"
  MINOR="${BASH_REMATCH[2]}"
  PATCH="${BASH_REMATCH[3]}"
else
  MAJOR=0; MINOR=1; PATCH=0
fi

# Get commits since last tag
COMMITS=$(git log "${LATEST}..HEAD" --pretty=format:"%s" 2>/dev/null || echo "")

# Determine bump type from conventional commits
BUMP="patch"
if echo "$COMMITS" | grep -qE '^feat(\!|\(|:)'; then
  BUMP="minor"
fi
if echo "$COMMITS" | grep -qE '^BREAKING CHANGE:|^feat\!\:'; then
  BUMP="major"
fi

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

VERSION="v${MAJOR}.${MINOR}.${PATCH}"
echo "Previous: ${LATEST} -> Next: ${VERSION} (${BUMP})"

git tag -a "${VERSION}" -m "Release ${VERSION}"
git push origin "${VERSION}"
echo "Tagged ${VERSION}"
echo "${VERSION}" > version.txt
buildkite-agent artifact upload version.txt