#!/usr/bin/env bash
# -------------------------------------------------------------------
# .buildkite/trigger.sh — Fire a Buildkite build from the command line
#
# Usage:
#   export BUILDKITE_API_TOKEN="xxx"
#   export BUILDKITE_ORG="your-org-slug"
#   export BUILDKITE_PIPELINE="builddeck"
#
#   ./trigger.sh                                          # trigger main
#   ./trigger.sh -b develop                               # trigger a branch
#   ./trigger.sh -b feat/foo -c "trigger: deploy preview" # custom message
#   ./trigger.sh --pipeline builddeck                     # specify pipeline
# -------------------------------------------------------------------
set -euo pipefail

TOKEN="${BUILDKITE_API_TOKEN:-}"
ORG="${BUILDKITE_ORG:-}"
PIPELINE="${BUILDKITE_PIPELINE:-}"
BRANCH="main"
MESSAGE="Triggered from trigger.sh"
COMMIT="HEAD"

usage() {
  sed -n 's/^# \{0,1\}//p' "$0" | sed '1,2d'
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -b|--branch) BRANCH="$2";  shift 2 ;;
    -m|--message) MESSAGE="$2"; shift 2 ;;
    -p|--pipeline) PIPELINE="$2"; shift 2 ;;
    -h|--help) usage ;;
    *) echo "unknown: $1"; usage ;;
  esac
done

: "${TOKEN:?BUILDKITE_API_TOKEN not set}"
: "${ORG:?BUILDKITE_ORG not set}"
: "${PIPELINE:?BUILDKITE_PIPELINE not set}"

echo "→ Triggering ${ORG}/${PIPELINE} on ${BRANCH} …"

curl -sS -X POST "https://api.buildkite.com/v2/organizations/${ORG}/pipelines/${PIPELINE}/builds" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "$(cat <<EOF
{
  "commit": "${COMMIT}",
  "branch": "${BRANCH}",
  "message": "${MESSAGE}",
  "meta_data": {
    "triggered_by": "trigger.sh"
  }
}
EOF
)" | python3 -m json.tool 2>/dev/null || cat

echo "✓ done"
