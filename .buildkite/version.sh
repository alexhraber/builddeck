#!/usr/bin/env bash
# Computes semver version from conventional commits since last tag.
# Outputs version string (e.g., v0.1.1) to stdout ONLY if feat/fix commits exist.
# Exits 0 with no output if no conventional commits since last tag.
# Usage: .buildkite/version.sh

set -euo pipefail

# Get the latest tag (or empty if none)
LATEST=$(git tag -l 'v*' --sort=-v:refname | head -1 || echo "")

if [[ -z "$LATEST" ]]; then
  # No tags exist - start from v0.0.0
  MAJOR=0; MINOR=0; PATCH=0
  COMMITS=$(git log --pretty=format:"%s" 2>/dev/null || echo "")
else
  if [[ "$LATEST" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    PATCH="${BASH_REMATCH[3]}"
  else
    MAJOR=0; MINOR=1; PATCH=0
  fi
  # Get commits since last tag
  COMMITS=$(git log "${LATEST}..HEAD" --pretty=format:"%s" 2>/dev/null || echo "")
fi

# Check if any conventional commits exist
if ! echo "$COMMITS" | grep -qE '^(feat|fix|refactor|perf|docs|style|test|chore)'; then
  # No conventional commits since last tag — no release needed
  exit 0
fi

# Determine bump type from conventional commits
BUMP="patch"
if echo "$COMMITS" | grep -qE '^feat(!|\()'; then
  BUMP="minor"
fi
if echo "$COMMITS" | grep -qE '^feat!:'; then
  BUMP="major"
fi

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

VERSION="v${MAJOR}.${MINOR}.${PATCH}"
echo "${VERSION}"