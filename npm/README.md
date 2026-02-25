# tdd-ai

**TDD guardrails for AI coding agents.** A CLI state machine that enforces red-green-refactor discipline for any LLM with terminal access.

[![npm version](https://img.shields.io/npm/v/tdd-ai.svg)](https://www.npmjs.com/package/tdd-ai)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/mauricioTechDev/tdd-ai/blob/main/LICENSE)

## The Problem

LLMs are excellent at writing tests and code. What they lack is **discipline**. Without guardrails, AI agents will:

- Write tests and implementation simultaneously instead of tests first
- Modify tests to match broken implementation instead of fixing the code
- Skip the red phase and never verify that tests actually fail before passing
- Add features nobody asked for instead of implementing the minimum to pass

## The Solution

`tdd-ai` is a TDD state machine that tracks what to build, enforces the red-green-refactor cycle, and surfaces structured state at each phase. The AI agent calls the CLI, reads the current state, decides what to do, does the work, then checks back in.

**It does not run tests.** The AI agent runs tests itself using whatever framework the project uses (`npm test`, `go test`, `pytest`, `cargo test`, etc.). This keeps the tool completely framework-agnostic.

## Install

```bash
npm install -g tdd-ai
```

## Quick Start

```bash
# 1. Start a TDD session
tdd-ai init --test-cmd "npm test"

# 2. Define what to build
tdd-ai spec add "Calculate shipping cost based on weight" "Free shipping over $50"

# 3. Pick a spec to work on
tdd-ai spec pick 1

# 4. Check current phase state (starts at red)
tdd-ai guide --format json
tdd-ai blockers --format json

# 5. After writing tests and confirming they fail, advance
tdd-ai phase next --test-result fail

# 6. Check state — now in green phase, write minimal implementation
tdd-ai guide --format json

# 7. After implementation passes all tests, advance
tdd-ai phase next --test-result pass

# 8. In refactor phase, answer reflection questions
tdd-ai refactor status
tdd-ai refactor reflect 1 --answer "Tests are already descriptive and clear enough"
# ... answer all 7 questions ...

# 9. Advance — loops back to RED for the next spec, or DONE if all specs are complete
tdd-ai phase next --test-result pass
```

## Works With Any AI Coding Tool

`tdd-ai` works with any AI agent that has terminal access. Add a rules file once and the AI always follows TDD:

### Cursor

Create `.cursor/rules/tdd-workflow.mdc`:

```markdown
---
description: TDD workflow using the tdd-ai CLI
globs: *
alwaysApply: true
---

When implementing any feature, use the tdd-ai CLI to follow strict TDD:

1. Run `tdd-ai init` to start a session (skip if .tdd-ai.json already exists)
2. Run `tdd-ai spec add "<requirement>"` for each requirement
3. Run `tdd-ai spec pick <id>` to pick ONE spec to work on
4. Run `tdd-ai guide --format json` to read current phase state
5. Do the work appropriate for the phase (red: failing tests, green: minimal impl, refactor: clean up)
6. Run `tdd-ai blockers --format json` to see what's blocking advancement
7. Run `tdd-ai phase next` when blockers are resolved
8. Repeat from step 4 — after REFACTOR, the current spec is auto-completed and the loop continues
```

### GitHub Copilot

Add to `AGENTS.md` or `.github/copilot-instructions.md`:

```markdown
## TDD Workflow

This project uses the `tdd-ai` CLI to enforce test-driven development.
Before implementing any feature, follow this workflow:

1. Run `tdd-ai init` if no session exists
2. Add specs with `tdd-ai spec add "<requirement>"`
3. Pick a spec with `tdd-ai spec pick <id>`
4. Check `tdd-ai guide --format json` for current phase state
5. Do the work for the current phase — do not skip phases
6. Run `tdd-ai blockers --format json` to see what's blocking advancement
7. Advance with `tdd-ai phase next` when blockers are resolved
```

### Claude Code

Add to `CLAUDE.md`:

```markdown
## TDD Workflow

Use the `tdd-ai` CLI to enforce strict TDD when implementing features.
Run `tdd-ai guide --format json` to check current phase state.
Run `tdd-ai blockers --format json` to see what's blocking advancement.
Advance with `tdd-ai phase next` when blockers are resolved.
Do not skip phases or write implementation before tests fail.
```

### Windsurf

Add to `.windsurfrules`:

```
When implementing features, use the tdd-ai CLI for TDD workflow.
Run `tdd-ai guide --format json` to check current phase state.
Run `tdd-ai blockers --format json` to see what's blocking advancement.
Advance with `tdd-ai phase next` when blockers are resolved.
Do not skip phases or write implementation before tests fail.
```

### Augment Code, Codex CLI, and Others

Any AI tool with terminal access works. Include the workflow instructions in your system prompt or custom instructions file.

## How It Works

The CLI provides structured state at each TDD phase. The agent reads the state and decides what to do:

```json
{
  "phase": "red",
  "mode": "greenfield",
  "next_phase": "green",
  "test_cmd": "npm test",
  "specs": [
    {
      "id": 1,
      "description": "Calculate shipping cost based on weight",
      "status": "active"
    }
  ],
  "current_spec": {
    "id": 1,
    "description": "Calculate shipping cost based on weight",
    "status": "active"
  },
  "iteration": 1,
  "total_specs": 2,
  "expected_test_result": "fail",
  "blockers": [
    "No test result recorded"
  ]
}
```

The `expected_test_result` tells the agent what outcome tests should produce. The `blockers` list tells the agent what must be resolved before `phase next` will succeed. Use `tdd-ai blockers --format json` to check blockers at any time:

```json
{
  "phase": "red",
  "blockers": ["No spec selected", "No test result recorded"],
  "can_advance": false
}
```

The agent reads the state, does the work, checks blockers, then advances to the next phase. The state machine enforces correctness — if blockers remain, `phase next` fails. The cycle continues until all phases are complete.

## Two Modes

**Greenfield** (default) — Building new features from scratch:
```
red (tests fail) -> green (tests pass) -> refactor -> [red (loop) | done]
```

**Retrofit** — Adding tests to existing code:
```bash
tdd-ai init --retrofit
# red (tests pass, verifying existing behavior) -> refactor -> [red (loop) | done]
```

## Agent Mode

Use `--agent` for stricter enforcement when AI agents are driving the workflow:

```bash
tdd-ai init --agent --test-cmd "npm test"
```

- `phase set` is **disabled entirely** — the agent must use `phase next`
- `complete` requires `--force` to prevent bypassing the TDD cycle
- Backward compatible — existing sessions default to non-agent mode

## TDD Compliance Verification

After completing specs, verify the session followed proper TDD discipline:

```bash
tdd-ai verify
# TDD Compliance: 100%
# Specs verified: 3, compliant: 3
# No violations found.
```

Checks for: spec_picked events, failing tests during RED phase, and no `phase_set` usage. Returns exit code 1 on violations — useful in CI. The compliance score also appears in `tdd-ai status` output.

## Hooks (Automated Enforcement)

tdd-ai provides two example hook scripts that enforce TDD discipline automatically. These scripts read JSON from stdin and check the `.tdd-ai.json` session file, so they work with any AI coding agent that supports a hook system — not just Claude Code.

1. **File-write gating** (`tdd-guard.sh`) — During the RED phase, blocks writes to non-test files. Only test patterns (`*_test.*`, `*.test.*`, `*.spec.*`, `*/test/*`, `*/tests/*`) are allowed.

2. **Commit gating** (`tdd-commit-check.sh`) — Blocks `git commit` when the TDD phase is not `done`. Ensures all specs complete RED-GREEN-REFACTOR before committing.

### Claude Code Setup

**Step 1.** Create the hooks directory:

```bash
mkdir -p .claude/hooks
```

**Step 2.** Create `.claude/hooks/tdd-guard.sh`:

```bash
#!/usr/bin/env bash
# tdd-guard.sh — Claude Code PreToolUse hook for file-write gating.
#
# During the RED phase, only test files may be written.
# Non-test file writes are blocked (exit 2) to enforce TDD discipline.
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
```

**Step 3.** Create `.claude/hooks/tdd-commit-check.sh`:

```bash
#!/usr/bin/env bash
# tdd-commit-check.sh — Claude Code PreToolUse hook for commit gating.
#
# Blocks git commit when the TDD phase is not "done".
# Ensures all specs are completed through RED-GREEN-REFACTOR before committing.
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
```

**Step 4.** Make both scripts executable:

```bash
chmod +x .claude/hooks/tdd-guard.sh .claude/hooks/tdd-commit-check.sh
```

**Step 5.** Add the hook configuration to `.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/tdd-guard.sh"
          }
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/tdd-commit-check.sh"
          }
        ]
      }
    ]
  }
}
```

See the [full documentation](https://github.com/mauricioTechDev/tdd-ai#hooks-automated-enforcement) for additional hook examples (Cursor, CI/CD).

## Commands

| Command | Description |
|---------|-------------|
| `tdd-ai init` | Start a new TDD session |
| `tdd-ai init --retrofit` | Start a session for testing existing code |
| `tdd-ai init --agent` | Start a session with stricter agent mode enforcement |
| `tdd-ai spec add "desc" [...]` | Add one or more specs |
| `tdd-ai spec list` | List all specs with status |
| `tdd-ai spec pick <id>` | Pick a spec to work on in the current iteration |
| `tdd-ai spec done <id> [...]` | Mark specs as completed |
| `tdd-ai blockers` | Show what's preventing phase advancement |
| `tdd-ai guide` | Get current phase state and context |
| `tdd-ai phase next` | Advance to next phase |
| `tdd-ai phase set <phase> --force` | Manually set phase (requires --force; disabled in agent mode) |
| `tdd-ai test` | Run configured test command |
| `tdd-ai refactor` | Show refactor reflection status |
| `tdd-ai refactor reflect <n> --answer "..."` | Answer a reflection question |
| `tdd-ai refactor status` | Show all reflection questions with status |
| `tdd-ai complete` | Finish TDD cycle (requires --force in agent mode) |
| `tdd-ai verify` | Check TDD compliance (exit 1 on violations) |
| `tdd-ai status` | Full session overview (includes compliance score) |

All commands support `--format json` for machine-readable output.

## Links

- [GitHub Repository](https://github.com/mauricioTechDev/tdd-ai) — Full documentation, design principles, and development guide
- [MIT License](https://github.com/mauricioTechDev/tdd-ai/blob/main/LICENSE)
