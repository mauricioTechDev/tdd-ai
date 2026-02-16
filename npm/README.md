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

`tdd-ai` is a TDD state machine that tracks what to build, enforces the red-green-refactor cycle, and gives the AI structured instructions at each phase. The AI agent calls the CLI, reads the guidance, does the work, then checks back in.

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

# 4. Get instructions for the current phase (starts at red)
tdd-ai guide --format json

# 5. After writing tests and confirming they fail, advance
tdd-ai phase next --test-result fail

# 6. Get green phase instructions, write minimal implementation
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
4. Run `tdd-ai guide --format json` to get phase instructions
5. Follow the instructions and rules EXACTLY
6. Run `tdd-ai phase next` when the current phase is complete
7. Repeat steps 4-6 — after REFACTOR, the current spec is auto-completed and the loop continues
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
4. Check `tdd-ai guide --format json` for phase-specific instructions
5. Follow the instructions exactly -- do not skip phases
6. Advance with `tdd-ai phase next` when each phase is complete
```

### Claude Code

Add to `CLAUDE.md`:

```markdown
## TDD Workflow

Use the `tdd-ai` CLI to enforce strict TDD when implementing features.
Run `tdd-ai guide --format json` before writing any code.
Follow the phase instructions exactly. Advance with `tdd-ai phase next`.
Do not skip phases or write implementation before tests fail.
```

### Windsurf

Add to `.windsurfrules`:

```
When implementing features, use the tdd-ai CLI for TDD workflow.
Run `tdd-ai guide --format json` before writing any code.
Follow the phase instructions exactly. Advance with `tdd-ai phase next`.
Do not skip phases or write implementation before tests fail.
```

### Augment Code, Codex CLI, and Others

Any AI tool with terminal access works. Include the workflow instructions in your system prompt or custom instructions file.

## How It Works

The CLI guides the AI through each TDD phase with structured JSON output:

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
  "instructions": [
    "Write a failing test for spec [1]: Calculate shipping cost based on weight",
    "Cover happy path, edge cases, and error conditions for this spec.",
    "Run the project's test command to verify the new test FAILS.",
    "Do NOT write any implementation code yet.",
    "When the test is written and confirmed failing, run: tdd-ai test && tdd-ai phase next (test result is stored and auto-used)"
  ],
  "rules": [
    "DO NOT create implementation files.",
    "DO NOT write skeleton or stub implementations.",
    "Tests must assert specific expected values, not just 'does not throw'."
  ]
}
```

The AI reads the `instructions` and `rules`, does the work, then advances to the next phase. The cycle continues until all phases are complete.

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

## Commands

| Command | Description |
|---------|-------------|
| `tdd-ai init` | Start a new TDD session |
| `tdd-ai init --retrofit` | Start a session for testing existing code |
| `tdd-ai spec add "desc" [...]` | Add one or more specs |
| `tdd-ai spec list` | List all specs with status |
| `tdd-ai spec pick <id>` | Pick a spec to work on in the current iteration |
| `tdd-ai spec done <id> [...]` | Mark specs as completed |
| `tdd-ai guide` | Get phase-appropriate instructions |
| `tdd-ai phase next` | Advance to next phase |
| `tdd-ai test` | Run configured test command |
| `tdd-ai refactor` | Show refactor reflection status |
| `tdd-ai refactor reflect <n> --answer "..."` | Answer a reflection question |
| `tdd-ai refactor status` | Show all reflection questions with status |
| `tdd-ai complete` | Finish TDD cycle |
| `tdd-ai status` | Full session overview |

All commands support `--format json` for machine-readable output.

## Links

- [GitHub Repository](https://github.com/mauricioTechDev/tdd-ai) — Full documentation, design principles, and development guide
- [MIT License](https://github.com/mauricioTechDev/tdd-ai/blob/main/LICENSE)
