#!/usr/bin/env bash
# tdd-commit-check.sh — Claude Code PreToolUse hook for commit gating.
#
# Blocks git commit when the TDD phase is not "done".
# Ensures all specs are completed through RED-GREEN-REFACTOR before committing.
#
# Exits 0 when:
#   - No .tdd-ai.json exists (not in a TDD session)
#   - Phase is "done"
#   - Command is not a git commit
#
# Exits 2 (BLOCK) when:
#   - Phase is not "done" and command is git commit
set -euo pipefail

INPUT=$(cat)

# Extract the bash command
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')

# Only gate git commit commands
if [[ "$COMMAND" != *"git commit"* && "$COMMAND" != *"git -C"*"commit"* ]]; then
  exit 0
fi

# No session file — not in a TDD session
SESSION_FILE=".tdd-ai.json"
if [[ ! -f "$SESSION_FILE" ]]; then
  exit 0
fi

# Read current phase
PHASE=$(jq -r '.phase // ""' "$SESSION_FILE")
if [[ "$PHASE" == "done" ]]; then
  exit 0
fi

# Block commit when TDD cycle is not complete
echo '{"result":"BLOCKED: TDD cycle not complete. Current phase is '"$PHASE"'. Complete all specs through RED-GREEN-REFACTOR before committing."}' >&2
exit 2
