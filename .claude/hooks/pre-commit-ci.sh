#!/usr/bin/env bash
set -euo pipefail

COMMAND=$(jq -r '.tool_input.command // ""')

# Fast path: only gate git commit commands
if [[ "$COMMAND" != *"git commit"* && "$COMMAND" != *"git -C"*"commit"* ]]; then
  exit 0
fi

echo "Pre-commit hook: running gofmt and make ci..." >&2

gofmt -w .

if ! make ci 2>&1; then
  echo "BLOCKED: make ci failed. Fix the issues above before committing." >&2
  exit 2
fi

echo "Pre-commit hook: all checks passed." >&2
exit 0
