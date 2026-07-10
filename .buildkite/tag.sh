#!/usr/bin/env bash
# Creates a release tag based on conventional commits since last tag.
# Runs only on main branch when not already a tag build.
# Uses GITHUB_TOKEN for authenticated git push.
set -euo pipefail

if [[ -n "${BUILDKITE_TAG:-}" || "${BUILDKITE_BRANCH:-}" != "main" ]]; then
  echo "Skipping tag creation (branch: ${BUILDKITE_BRANCH}, tag: ${BUILDKITE_TAG})"
  echo "dev" > version.txt
  buildkite-agent artifact upload version.txt
  exit 0
fi

echo "--- Creating release tag"

# Configure git identity for tag creation
git config user.email "builddeck@buildkite.com"
git config user.name "builddeck-bot"

# Get the latest tag (or v0.0.0 if none)
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

# Debug: check if token is available
if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "ERROR: GITHUB_TOKEN not set"
  exit 1
fi

# Use GITHUB_TOKEN for authenticated push
git remote set-url origin "https://${GITHUB_TOKEN}@github.com/alexhraber/builddeck.git"
git config user.email "builddeck@buildkite.com"
git config user.name "builddeck-bot"

git tag -a "${VERSION}" -m "Release ${VERSION}"
git push origin "${VERSION}"
echo "Tagged ${VERSION}"
echo "${VERSION}" > version.txt
buildkite-agent artifact upload version.txt