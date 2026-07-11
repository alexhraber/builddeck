#!/usr/bin/env bash
# Validates that commits since the last release tag (or the last 10 commits
# on branches without tags) follow conventional commits format.
# This ensures the auto-tagging in tag.sh + version.sh works correctly.
# Exits 0 if all commits are valid, 1 with a report of invalid commits otherwise.
set -euo pipefail

CONVENTIONAL_COMMITS_PATTERN='^(feat|fix|refactor|perf|docs|style|test|chore|build|ci|revert)(\([a-z_]+\))?!?:\s.+'
MAX_COMMITS=10

if ! command -v git &>/dev/null; then
  echo "commitlint: git not found — skipping"
  exit 0
fi

LATEST_TAG=$(git tag -l 'v*' --sort=-v:refname | head -1 || echo "")

if [[ -n "${LATEST_TAG}" ]]; then
  COMMITS=$(git log "${LATEST_TAG}..HEAD" --pretty=format:"%s" 2>/dev/null || echo "")
else
  COMMITS=$(git log --max-count=${MAX_COMMITS} --pretty=format:"%s" 2>/dev/null || echo "")
fi

if [[ -z "${COMMITS}" ]]; then
  echo "commitlint: no commits to check"
  exit 0
fi

INVALID=()
while IFS= read -r msg; do
  [[ -z "${msg}" ]] && continue
  if ! echo "${msg}" | grep -qE "${CONVENTIONAL_COMMITS_PATTERN}" && ! echo "${msg}" | grep -qE '^(Merge|Revert)'; then
    INVALID+=("${msg}")
  fi
done <<< "${COMMITS}"

if [[ ${#INVALID[@]} -eq 0 ]]; then
  echo "commitlint: all commits follow conventional commits format ✓"
  exit 0
fi

echo "^^^ +++"
echo "commitlint: ${#INVALID[@]} commit(s) do not follow conventional commits format:"
echo ""
printf '  • %s\n' "${INVALID[@]}"
echo ""
echo "Expected format: type(scope)!: description"
echo "  types: feat, fix, refactor, perf, docs, style, test, chore, build, ci, revert"
echo "  scope: optional (e.g. feat(api):, fix(tui):)"
echo "  breaking: add ! after scope (e.g. feat!:, fix(api)!:)"
echo ""
echo "Examples:"
echo "  feat: add new endpoint"
echo "  fix(tui): handle nil pointer in log view"
echo "  feat(api)!: change response format (breaking)"
exit 1
