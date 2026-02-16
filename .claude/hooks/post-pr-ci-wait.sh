#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')
STDOUT=$(echo "$INPUT" | jq -r '.stdout // ""')

PR_NUMBER=""

# Case 1: gh pr create — PR URL is in stdout
if [[ "$COMMAND" == *"gh pr create"* ]]; then
  PR_NUMBER=$(echo "$STDOUT" | grep -oE '/pull/[0-9]+' | grep -oE '[0-9]+' | tail -1)
fi

# Case 2: git push — check if current branch has an open PR
if [[ "$COMMAND" == *"git push"* && -z "$PR_NUMBER" ]]; then
  PR_NUMBER=$(gh pr view --json number -q '.number' 2>/dev/null || true)
fi

# No PR found — nothing to do
if [[ -z "$PR_NUMBER" ]]; then
  exit 0
fi

echo "Waiting for CI on PR #${PR_NUMBER}..." >&2

# Wait for checks to complete (timeout after 5 minutes)
if gh pr checks "$PR_NUMBER" --watch --fail-fast 2>/dev/null; then
  # All checks passed
  jq -n --arg pr "$PR_NUMBER" '{
    "systemMessage": ("CI passed on PR #" + $pr + ". All checks green. Report success to the user.")
  }'
else
  # Some check failed — grab the failed run logs
  FAILED_RUN=$(gh run list --branch "$(git branch --show-current)" --status failure --limit 1 --json databaseId -q '.[0].databaseId' 2>/dev/null || true)
  LOGS=""
  if [[ -n "$FAILED_RUN" ]]; then
    LOGS=$(gh run view "$FAILED_RUN" --log-failed 2>/dev/null | tail -30 || true)
  fi

  jq -n --arg pr "$PR_NUMBER" --arg logs "$LOGS" '{
    "systemMessage": ("CI FAILED on PR #" + $pr + ". Investigate the failure, fix if possible, or report to the user. Failed logs:\n" + $logs)
  }'
fi
