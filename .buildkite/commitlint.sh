#!/usr/bin/env bash
# Validates that commits unique to the current branch follow conventional
# commits format. On main, checks commits since the last release tag.
# On feature branches, checks commits not in origin/main.
# Exits 0 if all commits are valid, 1 with a report of invalid commits otherwise.
set -euo pipefail

PATTERN='^(feat|fix|refactor|perf|docs|style|test|chore|build|ci|revert)(\([a-z_]+\))?!?:\s.+'

if ! command -v git &>/dev/null; then
  echo "commitlint: git not found — skipping"
  exit 0
fi

BRANCH="${BUILDKITE_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}"

if [[ "${BRANCH}" == "main" ]]; then
  LATEST_TAG=$(git tag -l 'v*' --sort=-v:refname | head -1 || echo "")
  if [[ -z "${LATEST_TAG}" ]]; then
    echo "commitlint: no tags found — skipping"
    exit 0
  fi
  RANGE="${LATEST_TAG}..HEAD"
else
  if git cat-file -e origin/main 2>/dev/null; then
    RANGE="origin/main..HEAD"
  else
    echo "commitlint: origin/main not available on feature branch — skipping"
    exit 0
  fi
fi

COMMITS=$(git log "${RANGE}" --pretty=format:"%s" 2>/dev/null || echo "")
if [[ -z "${COMMITS}" ]]; then
  echo "commitlint: no commits in range ${RANGE}"
  exit 0
fi

INVALID=()
while IFS= read -r msg; do
  [[ -z "${msg}" ]] && continue
  if ! echo "${msg}" | grep -qE "${PATTERN}" && ! echo "${msg}" | grep -qE '^(Merge|Revert)'; then
    INVALID+=("${msg}")
  fi
done <<< "${COMMITS}"

if [[ ${#INVALID[@]} -eq 0 ]]; then
  echo "commitlint: all ${BRANCH} commits follow conventional commits format ✓"
  exit 0
fi

echo "^^^ +++"
echo "commitlint: ${#INVALID[@]} commit(s) in range ${RANGE} do not follow conventional commits format:"
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
