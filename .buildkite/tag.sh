#!/usr/bin/env bash
# Creates a release tag based on conventional commits.
# Runs only on main branch when not already a tag build.
set -euo pipefail

if [[ -n "${BUILDKITE_TAG:-}" || "${BUILDKITE_BRANCH:-}" != "main" ]]; then
  echo "Skipping tag creation (branch: ${BUILDKITE_BRANCH}, tag: ${BUILDKITE_TAG})"
  exit 0
fi

echo "--- Creating release tag"

git config user.email "builddeck@buildkite.com"
git config user.name "builddeck-bot"

VERSION=$(.buildkite/version.sh)
echo "Version: ${VERSION}"

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "ERROR: GITHUB_TOKEN not set"
  exit 1
fi

git remote set-url origin "https://alexhraber:${GITHUB_TOKEN}@github.com/alexhraber/builddeck.git"

git tag -a "${VERSION}" -m "Release ${VERSION}"
git push origin "${VERSION}"
echo "Tagged ${VERSION}"