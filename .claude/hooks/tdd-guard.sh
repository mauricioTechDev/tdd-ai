#!/usr/bin/env bash
# tdd-guard.sh — Claude Code PreToolUse hook for file-write gating.
#
# During the RED phase, only test files may be written.
# Non-test file writes are blocked (exit 2) to enforce TDD discipline.
#
# Exits 0 when:
#   - No .tdd-ai.json exists (not in a TDD session)
#   - Phase is not "red"
#   - The file being written matches a test pattern
#   - Tool is not a file-write tool (Write/Edit)
#
# Exits 2 (BLOCK) when:
#   - Phase is "red" and the file is not a test file
#
# Hook JSON is read from stdin (Claude Code PreToolUse format).
# The tool_input contains file_path for Write/Edit tools.
set -euo pipefail

INPUT=$(cat)

# Only gate Write and Edit tools
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""')
if [[ "$TOOL_NAME" != "Write" && "$TOOL_NAME" != "Edit" ]]; then
  exit 0
fi

# No session file — not in a TDD session
SESSION_FILE=".tdd-ai.json"
if [[ ! -f "$SESSION_FILE" ]]; then
  exit 0
fi

# Read current phase
PHASE=$(jq -r '.phase // ""' "$SESSION_FILE")
if [[ "$PHASE" != "red" ]]; then
  exit 0
fi

# Extract file path from tool input
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // ""')
if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

# Check if file matches test patterns
BASENAME=$(basename "$FILE_PATH")
if [[ "$BASENAME" == *_test.* ]] || \
   [[ "$BASENAME" == *.test.* ]] || \
   [[ "$BASENAME" == *.spec.* ]] || \
   [[ "$FILE_PATH" == */test/* ]] || \
   [[ "$FILE_PATH" == */tests/* ]]; then
  exit 0
fi

# Block non-test file writes during RED phase
echo '{"result":"BLOCKED: During the RED phase, only test files may be written. Write your failing test first, then advance to GREEN to write implementation code."}' >&2
exit 2
