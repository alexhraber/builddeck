#!/usr/bin/env bash
# Creates a release tag based on conventional commits.
# Runs only on main branch when not already a tag build.
# Outputs version to tag.txt artifact for downstream consumption.
# Exits silently if no conventional commits (feat/fix) since last tag.
set -euo pipefail

if [[ -n "${BUILDKITE_TAG:-}" || "${BUILDKITE_BRANCH:-}" != "main" ]]; then
  echo "Skipping tag creation (branch: ${BUILDKITE_BRANCH}, tag: ${BUILDKITE_TAG})"
  exit 0
fi

echo "--- Creating release tag"

git config user.email "builddeck@buildkite.com"
git config user.name "builddeck-bot"

VERSION=$(.buildkite/version.sh)

# No conventional commits since last tag — silently exit (no tag, no release)
if [[ -z "${VERSION}" ]]; then
  echo "No conventional commits (feat/fix) since last tag — skipping release"
  exit 0
fi

echo "Version: ${VERSION}"

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "ERROR: GITHUB_TOKEN not set"
  exit 1
fi

git remote set-url origin "https://alexhraber:${GITHUB_TOKEN}@github.com/alexhraber/builddeck.git"

# Create tag
git tag -a "${VERSION}" -m "Release ${VERSION}"
echo "Tag created locally: ${VERSION}"

# Push tag with explicit error handling
if git push origin "${VERSION}"; then
  echo "Tagged ${VERSION}"
else
  echo "ERROR: Failed to push tag ${VERSION}"
  exit 1
fi

# Output version to artifact for TUI consumption
echo "${VERSION}" > builddeck.tag
buildkite-agent artifact upload builddeck.tag
echo "Uploaded builddeck.tag artifact"