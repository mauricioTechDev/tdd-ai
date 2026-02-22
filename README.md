# tdd-ai

A CLI tool that acts as TDD guardrails for AI coding agents.

## Why This Exists

LLMs are excellent at writing tests and code. What they lack is **discipline**. Without guardrails, AI agents will:

- Write tests and implementation simultaneously instead of tests first
- Modify tests to match broken implementation instead of fixing the code
- Skip the red phase and never verify that tests actually fail before passing
- Add features nobody asked for instead of implementing the minimum to pass

`tdd-ai` fixes this. It is a TDD state machine that tracks what to build, enforces the red-green-refactor cycle, and surfaces structured state at each phase. The AI agent calls the CLI, reads the current state, decides what to do, does the work, then checks back in.

**It does not run tests.** The AI agent runs tests itself using whatever framework the project uses (`npm test`, `go test`, `pytest`, `cargo test`, etc.). This keeps the tool completely framework-agnostic -- it works with any language, any test runner, any AI agent.

## Install

### Option 1: npm (recommended)

```bash
npm install -g tdd-ai
```

### Option 2: Download Binary (no Node.js required)

Pre-built binaries are available for the following platforms:

| OS      | Architecture | Archive       |
|---------|-------------|---------------|
| Linux   | amd64 (x86-64) | `.tar.gz`  |
| Linux   | arm64 (Apple Silicon / ARM servers) | `.tar.gz` |
| macOS   | amd64 (Intel) | `.tar.gz`  |
| macOS   | arm64 (Apple Silicon) | `.tar.gz` |
| Windows | amd64 (x86-64) | `.zip`    |
| Windows | arm64       | `.zip`        |

Download the latest release from the [GitHub Releases page](https://github.com/mauricioTechDev/tdd-ai/releases/latest).

**macOS / Linux:**

```bash
# Replace <version>, <os>, and <arch> with the values for your system
# Example: tdd-ai_0.5.0_darwin_arm64.tar.gz (macOS Apple Silicon)
# Example: tdd-ai_0.5.0_linux_amd64.tar.gz  (Linux x86-64)
curl -L https://github.com/mauricioTechDev/tdd-ai/releases/latest/download/tdd-ai_<version>_<os>_<arch>.tar.gz | tar xz

# Move to a directory on your PATH
sudo mv tdd-ai /usr/local/bin/
```

**Windows:**

Download the `.zip` file from the [Releases page](https://github.com/mauricioTechDev/tdd-ai/releases/latest), extract it, and add the directory to your `PATH`.

### Option 3: Build from Source (requires Go 1.22+)

```bash
git clone https://github.com/mauricioTechDev/tdd-ai.git
cd tdd-ai
make build

# Binary is at bin/tdd-ai
# Optionally copy to your PATH:
cp bin/tdd-ai ~/go/bin/
```

### Verify Installation

```bash
tdd-ai version
```

## Quick Start

```bash
# 1. Start a TDD session (optionally configure the test command)
tdd-ai init --test-cmd "go test ./..."

# 2. Define what to build (supports multiple specs at once)
tdd-ai spec add "Calculate shipping cost based on weight" "Free shipping over $50"

# 3. Pick a spec to work on
tdd-ai spec pick 1

# 4. Check current state and blockers (starts at red)
tdd-ai guide --format json
tdd-ai blockers --format json

# 5. After writing tests and confirming they fail, advance
tdd-ai phase next --test-result fail

# 6. Check state — now in green phase
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

## Commands

| Command | Description |
|---------|-------------|
| `tdd-ai init` | Start a new TDD session (greenfield mode) |
| `tdd-ai init --retrofit` | Start a session for testing existing code |
| `tdd-ai init --test-cmd "cmd"` | Start a session with a configured test command |
| `tdd-ai spec add "desc" [...]` | Add one or more specs |
| `tdd-ai spec list` | List all specs with status |
| `tdd-ai spec pick <id>` | Pick a spec to work on in the current iteration |
| `tdd-ai spec done <id> [id...]` | Mark one or more specs as completed |
| `tdd-ai spec done --all` | Mark all active specs as completed |
| `tdd-ai phase` | Show current phase |
| `tdd-ai phase next` | Advance to next phase |
| `tdd-ai phase next --test-result pass\|fail` | Advance with test result validation |
| `tdd-ai phase set <phase>` | Manually set phase (red/green/refactor/done) |
| `tdd-ai blockers` | Show what's preventing phase advancement |
| `tdd-ai guide` | Get current phase state and context |
| `tdd-ai test` | Run configured test command and record result |
| `tdd-ai refactor` | Show refactor reflection status |
| `tdd-ai refactor reflect <n> --answer "..."` | Answer a reflection question |
| `tdd-ai refactor status` | Show all reflection questions with status |
| `tdd-ai complete` | Finish TDD cycle (advance to done + mark specs complete) |
| `tdd-ai status` | Full session overview (phase, mode, specs, next action) |
| `tdd-ai reset` | Clear session and start over |
| `tdd-ai version` | Print version |

All commands support `--format json` for machine-readable output.

### Batch Operations

Add multiple specs in a single command:

```bash
tdd-ai spec add "GET /users returns 200" "POST /users returns 201" "GET /users/999 returns 404"
```

Mark multiple specs done at once:

```bash
tdd-ai spec done 1 2 3
tdd-ai spec done --all
```

### Retrofit Mode

Use `--retrofit` when adding tests to existing code. In retrofit mode:

- The RED phase expects tests to **pass** (since implementation already exists)
- The GREEN phase is skipped (red goes directly to refactor)
- After refactor, loops back to RED for the next spec (or DONE if all specs are complete)
- Guidance tells the agent to verify existing behavior, not create new implementations

```bash
tdd-ai init --retrofit --test-cmd "dotnet test MyProject.Tests"
tdd-ai spec add "GET /users returns 200" "POST /users validates input"
tdd-ai guide --format json   # Instructions say: verify existing behavior
```

### Test Command Integration

Configure a test command during init to enable automatic test running:

```bash
# Configure during init
tdd-ai init --test-cmd "go test ./..."
tdd-ai init --retrofit --test-cmd "npm test"

# Run tests and record the result
tdd-ai test

# The result is automatically used by 'phase next'
tdd-ai phase next   # No --test-result needed!
```

The `tdd-ai test` command runs the configured test command, captures the exit code (0 = pass, non-zero = fail), stores the result in the session, and prints the test output. When `phase next` is called without `--test-result`, it automatically reads the stored result.

### Quick Completion

When you're done with a TDD cycle, use `complete` to wrap up in one command:

```bash
# Runs tests (if configured), advances to done, marks all specs complete
tdd-ai complete

# Or with explicit test result
tdd-ai complete --test-result pass
```

This replaces the ceremony of running `phase next` multiple times plus `spec done --all`.

### Test Result Validation

Use `--test-result` with `phase next` to validate test state before advancing:

```bash
# In red phase (greenfield): tests should be failing
tdd-ai phase next --test-result fail

# In green phase: tests should be passing
tdd-ai phase next --test-result pass

# In red phase (retrofit): tests should be passing
tdd-ai phase next --test-result pass
```

When `--test-result` is omitted and no stored test result exists, a warning is printed but the transition still proceeds.

## Using tdd-ai with AI Agents

The core idea: instead of hoping the AI follows TDD, you give it a CLI that **enforces it through state and blockers**. The AI checks the current phase state, decides what to do, does the work, then advances the phase. The CLI guarantees correctness — if tests didn't fail in RED, `phase next` won't succeed.

### The Prompt

The simplest way to use `tdd-ai` is to include it in your prompt when asking an AI agent to implement something:

```
Implement a URL shortener service with create, redirect, and stats endpoints.

Use the tdd-ai CLI to follow strict TDD. Here is the workflow:

1. Run: tdd-ai init
2. Run: tdd-ai spec add "<requirement>" for each distinct requirement
3. Run: tdd-ai spec pick <id> to pick ONE spec to work on
4. Run: tdd-ai guide --format json to check current phase state
5. Do the work appropriate for the current phase
6. If unsure what's blocking you, run: tdd-ai blockers --format json
7. When the phase is complete, run: tdd-ai phase next
8. Run: tdd-ai guide --format json again and repeat
9. After REFACTOR, the tool auto-completes the current spec and loops back to RED for the next spec
10. Continue until tdd-ai phase shows "done" (all specs complete)
```

This works with any AI agent that has terminal access: Cursor, Claude Code, Windsurf, Codex CLI, GitHub Copilot in the terminal, or any other tool.

### Agent Rules (Persistent Configuration)

Re-typing the prompt every time is tedious. Most AI coding tools support persistent rules that are loaded automatically. Create a rule file once and the AI always knows the workflow.

**Cursor** (`.cursor/rules/tdd-workflow.mdc`):

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
4. Run `tdd-ai guide --format json` to read the current phase state
5. Do the work appropriate for the phase (red: write failing tests, green: write minimal impl, refactor: clean up)
6. Run `tdd-ai blockers --format json` to check what's blocking advancement
7. Run `tdd-ai phase next` when all blockers are resolved
8. Repeat from step 4 until phase reaches "done"

Key rules:
- In red phase: pick a spec first, then write tests only, verify they fail
- In green phase: write minimal implementation, do NOT modify tests
- In refactor phase: improve code quality, answer all 7 reflection questions with `tdd-ai refactor reflect`
- After refactor: the current spec is auto-completed, loop continues with the next spec
```

**GitHub Copilot / Codex** (`.github/copilot-instructions.md` or `AGENTS.md`):

```markdown
## TDD Workflow

This project uses the `tdd-ai` CLI to enforce test-driven development.
Before implementing any feature, follow this workflow:

1. Run `tdd-ai init` if no session exists
2. Add specs with `tdd-ai spec add "<requirement>"`
3. Pick a spec with `tdd-ai spec pick <id>`
4. Check `tdd-ai guide --format json` for current phase state
5. Do the work for the current phase — do not skip phases
6. Check `tdd-ai blockers --format json` if unsure what's blocking you
7. Advance with `tdd-ai phase next` when blockers are resolved
```

**Windsurf** (`.windsurfrules`):

```
When implementing features, use the tdd-ai CLI for TDD workflow.
Run `tdd-ai guide --format json` to check current phase state.
Run `tdd-ai blockers --format json` to see what's blocking advancement.
Advance with `tdd-ai phase next` when blockers are resolved.
Do not skip phases or write implementation before tests fail.
```

**Any agent with a system prompt**: Include the workflow instructions from "The Prompt" section above in your system prompt or custom instructions.

### Hooks (Automated Enforcement)

Some AI tools support hooks -- scripts that run automatically before or after the agent acts. You can use hooks to enforce the TDD workflow without relying on the AI to remember.

**Cursor hooks** (`.cursor/hooks.json`):

```json
{
  "version": 1,
  "hooks": {
    "stop": [
      {
        "command": "bash .cursor/hooks/tdd-check.sh"
      }
    ]
  }
}
```

```bash
#!/bin/bash
# .cursor/hooks/tdd-check.sh
# Runs when the agent finishes a turn. If the TDD cycle is not complete,
# sends a follow-up message telling the agent to continue.

if [ ! -f .tdd-ai.json ]; then
  exit 0
fi

PHASE=$(tdd-ai phase 2>/dev/null)

if [ "$PHASE" = "done" ]; then
  echo '{}'
else
  GUIDE=$(tdd-ai guide --format json)
  cat <<EOF
{
  "followup_message": "TDD cycle is not complete. Current phase: $PHASE. Run tdd-ai guide --format json and follow the instructions."
}
EOF
fi
```

This creates an automated loop: the agent works, the hook checks if TDD is done, and if not, it sends the agent back to keep going.

### CI/CD (Pull Request Checks)

You can also use `tdd-ai` in CI to verify that pull requests followed the TDD workflow:

```yaml
# .github/workflows/tdd-check.yml
name: TDD Check
on: [pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Check TDD session
        run: |
          if [ -f .tdd-ai.json ]; then
            PHASE=$(./bin/tdd-ai phase)
            if [ "$PHASE" != "done" ]; then
              echo "TDD session is not complete (phase: $PHASE)"
              exit 1
            fi
          fi
```

## How the Agent Flow Works

Here is the full sequence of what happens when an AI agent uses `tdd-ai`:

```
Developer: "Implement a rate limiter with sliding window"
                    |
                    v
     Agent runs: tdd-ai init
     Agent runs: tdd-ai spec add "Rate limiter with sliding window algorithm"
     Agent runs: tdd-ai spec add "Return 429 when limit exceeded"
                    |
                    v
  ┌─────────────────────────────────────────────┐
  │  Per-Spec Loop (one spec at a time)         │
  │                                             │
  │  Agent runs: tdd-ai spec pick 1             │
  │  Agent runs: tdd-ai guide --format json     │
  │                    |                        │
  │                    v                        │
  │          Phase: RED                         │
  │          expected_test_result: "fail"       │
  │          blockers: ["No test result..."]    │
  │                    |                        │
  │                    v                        │
  │  Agent writes test for spec [1]             │
  │  Agent runs: npm test → confirms tests fail │
  │  Agent runs: tdd-ai phase next --test-result fail
  │  Agent runs: tdd-ai guide --format json     │
  │                    |                        │
  │                    v                        │
  │          Phase: GREEN                       │
  │          expected_test_result: "pass"       │
  │                    |                        │
  │                    v                        │
  │  Agent writes minimal implementation        │
  │  Tests fail? → Agent fixes (NOT the tests)  │
  │  Tests pass? → Agent runs: tdd-ai phase next --test-result pass
  │  Agent runs: tdd-ai guide --format json     │
  │                    |                        │
  │                    v                        │
  │          Phase: REFACTOR                    │
  │          expected_test_result: "pass"       │
  │                    |                        │
  │                    v                        │
  │  Agent refactors                            │
  │  Agent runs: npm test after each change     │
  │  Agent answers all 7 reflection questions   │
  │  Agent runs: tdd-ai phase next --test-result pass
  │                    |                        │
  │                    v                        │
  │  Spec [1] auto-completed                    │
  │  Specs remaining? → Loop back to RED        │
  │  All specs done?  → Advance to DONE         │
  └─────────────────────────────────────────────┘
                    |
                    v
              Done.
```

### JSON Output Examples

```bash
tdd-ai guide --format json
```

```json
{
  "phase": "red",
  "mode": "greenfield",
  "next_phase": "green",
  "test_cmd": "go test ./...",
  "specs": [
    {
      "id": 1,
      "description": "Rate limiter with sliding window algorithm",
      "status": "active"
    },
    {
      "id": 2,
      "description": "Return 429 when limit exceeded",
      "status": "active"
    }
  ],
  "current_spec": {
    "id": 1,
    "description": "Rate limiter with sliding window algorithm",
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

The `expected_test_result` field tells the agent what outcome tests should produce in the current phase. The `blockers` list tells the agent what must be resolved before `phase next` will succeed. Use `tdd-ai blockers --format json` to check blockers at any time:

```bash
tdd-ai blockers --format json
```

```json
{
  "phase": "red",
  "blockers": ["No spec selected", "No test result recorded"],
  "can_advance": false
}
```

The `current_spec`, `iteration`, and `total_specs` fields show which spec the agent is working on and how far through the backlog it is. The `mode`, `next_phase`, and `test_cmd` fields help AI agents understand the full context without needing to call multiple commands.

## Development

```bash
# Run tests
make test

# Build
make build

# Lint
make lint
```

## Design Principles

- **The CLI does not run tests.** The AI agent already knows how to do that. This tool provides structure, not execution.
- **State over instructions.** The CLI surfaces state (current phase, expected test result, blockers) and lets the agent decide what to do. Constraints are enforced by the state machine, not by telling the agent what to do.
- **Blockers over hints.** Instead of telling the agent "pick a spec first," `phase next` fails and `blockers` explains why. Enforcement beats aspiration.
- **Framework-agnostic.** Works with any language, any test runner, any AI agent.
- **Agent-agnostic.** Works with Cursor, Claude Code, Codex, GitHub Copilot, Windsurf, or any tool with terminal access.
- **Structured output.** `--format json` lets AI agents parse state programmatically.
- **Discipline over intelligence.** LLMs are smart but undisciplined. This tool enforces the red-green-refactor cycle.
