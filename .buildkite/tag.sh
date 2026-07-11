#!/usr/bin/env bash
# Creates and pushes a release tag based on the release type passed as argument.
# Reads the latest semver tag from git, bumps accordingly, tags the current commit,
# and uploads the version to a builddeck.tag artifact.
# Usage: tag.sh <patch|minor>
#   patch — increment PATCH (v0.1.0 → v0.1.1)
#   minor — increment MINOR, reset PATCH (v0.1.0 → v0.2.0)
set -euo pipefail

RELEASE_TYPE="${1:-}"
if [[ -z "${RELEASE_TYPE}" ]]; then
  echo "Usage: $0 <patch|minor>"
  exit 1
fi

echo "--- Creating release tag"

git config user.email "builddeck@buildkite.com"
git config user.name "builddeck-bot"

# Find the latest semver tag (vMAJOR.MINOR.PATCH)
LATEST_TAG=$(git tag -l 'v*' --sort=-v:refname | head -1 || echo "")

if [[ -z "$LATEST_TAG" ]]; then
  MAJOR=0; MINOR=0; PATCH=0
  echo "No existing tags found — starting from v0.0.0"
else
  echo "Latest tag: ${LATEST_TAG}"
  if [[ "$LATEST_TAG" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    PATCH="${BASH_REMATCH[3]}"
  else
    echo "ERROR: Could not parse version from tag: ${LATEST_TAG}"
    exit 1
  fi
fi

case "${RELEASE_TYPE}" in
  minor)
    TAG="v${MAJOR}.$((MINOR + 1)).0"
    ;;
  patch)
    TAG="v${MAJOR}.${MINOR}.$((PATCH + 1))"
    ;;
  *)
    echo "ERROR: Unknown release type: ${RELEASE_TYPE}. Use 'patch' or 'minor'."
    exit 1
    ;;
esac

echo "New tag: ${TAG}"

if git ls-remote --exit-code --tags origin "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "ERROR: Tag ${TAG} already exists at origin"
  exit 1
fi

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "ERROR: GITHUB_TOKEN not set"
  exit 1
fi

git remote set-url origin "https://alexhraber:${GITHUB_TOKEN}@github.com/alexhraber/builddeck.git"

echo "+++ Tagging ${BUILDKITE_COMMIT} with ${TAG}"
git tag -a "${TAG}" -m "Release ${TAG}"

if git push origin "${TAG}"; then
  echo "Tagged ${TAG}"
else
  echo "ERROR: Failed to push tag ${TAG}"
  exit 1
fi

echo "${TAG}" > builddeck.tag
buildkite-agent artifact upload builddeck.tag
echo "Uploaded builddeck.tag artifact"
