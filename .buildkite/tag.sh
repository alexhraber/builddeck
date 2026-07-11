#!/usr/bin/env bash
# Creates a release tag based on conventional commits since the last tag.
# Runs only on main branch when not already a tag build.
# Computes the next semver via version.sh and pushes the tag.
# Uploads builddeck.tag artifact for downstream steps.
set -euo pipefail

if [[ -n "${BUILDKITE_TAG:-}" || "${BUILDKITE_BRANCH:-}" != "main" ]]; then
  echo "Skipping tag creation (branch: ${BUILDKITE_BRANCH}, tag: ${BUILDKITE_TAG})"
  exit 0
fi

echo "--- Checking conventional commits for release tag"

git config user.email "builddeck@buildkite.com"
git config user.name "builddeck-bot"

VERSION=$(cd "$(dirname "$0")" && ./version.sh)

if [[ -z "${VERSION}" ]]; then
  echo "No conventional commits (feat/fix) since last tag — skipping release"
  exit 0
fi

echo "Version: ${VERSION}"

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "ERROR: GITHUB_TOKEN not set"
  exit 1
fi

# Check if tag already exists at origin before trying to create it
if git ls-remote --exit-code --tags origin "refs/tags/${VERSION}" >/dev/null 2>&1; then
  echo "Tag ${VERSION} already exists at origin — skipping"
  exit 0
fi

git remote set-url origin "https://alexhraber:${GITHUB_TOKEN}@github.com/alexhraber/builddeck.git"

git tag -a "${VERSION}" -m "Release ${VERSION}"
echo "Tag created locally: ${VERSION}"

if git push origin "${VERSION}"; then
  echo "+++ Tagged ${VERSION}"
else
  echo "ERROR: Failed to push tag ${VERSION}"
  exit 1
fi

echo "${VERSION}" > builddeck.tag
buildkite-agent artifact upload builddeck.tag
echo "Uploaded builddeck.tag artifact"
