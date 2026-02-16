# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build        # Build binary to bin/tdd-ai (injects version via ldflags)
make test         # Run all tests: go test ./... -v
make test-short   # Run tests without I/O: go test ./... -short
make test-race    # Run tests with race detector: go test -race ./...
make lint         # Lint with golangci-lint (errcheck, staticcheck, gocritic, revive, etc.)
make ci           # Run full CI suite locally: lint + test-race + build
make coverage     # Run tests with race detector and print per-function coverage
make install      # Copy binary to ~/go/bin/
make clean        # Remove bin/ and coverage.out
```

Run a single package's tests:
```bash
go test ./internal/phase/ -v -run TestNext
```

## Pre-Commit Checklist

Before committing or pushing code, ALWAYS run:

1. `gofmt -w .` — Auto-format all Go files
2. `make ci` — Run lint, tests (with race detector), and build

Do NOT commit or push if `make ci` fails. Fix issues first.

## Architecture

**tdd-ai** is a Go CLI tool that enforces TDD discipline for AI coding agents. It does NOT run tests itself — the AI agent runs tests using whatever framework the project uses. The tool provides a state machine, phase-appropriate guidance, and structured output.

### Package Layout

- `main.go` — Entry point, calls `cmd.Execute()`
- `cmd/` — Cobra CLI commands. Each command follows: load session → validate → mutate → save → output
- `internal/types/` — Core data structures: `Phase`, `Mode`, `Spec`, `Session`, `Guidance`, `Event`
- `internal/session/` — Session persistence (read/write `.tdd-ai.json` in working directory)
- `internal/phase/` — State machine: `Next()`, `NextWithMode()`, `NextInLoop()`, `ExpectedTestResult()`, `CanTransition()`
- `internal/guide/` — Generates phase-specific instructions and rules based on current phase and mode
- `internal/reflection/` — Default reflection questions and answer validation for the refactor phase
- `internal/formatter/` — Formats output as text or JSON (`FormatGuidance`, `FormatStatus`, `FormatFullStatus`)

### Key Concepts

**Per-Spec TDD Loop (Canonical TDD):**
The CLI follows Kent Beck's canonical TDD tight loop. Instead of batch-processing all specs through phases, it works on **one spec at a time**:

1. `tdd-ai spec add "desc1" "desc2" ...` — Add specs to the backlog
2. `tdd-ai spec pick <id>` — Pick ONE spec to work on
3. RED → GREEN → REFACTOR for that spec
4. On `phase next` from REFACTOR: auto-completes the current spec, loops back to RED if specs remain, or advances to DONE if the backlog is empty
5. `tdd-ai complete` — Escape hatch to batch-finish all remaining specs at once

**TDD Phase State Machine:**
- Greenfield: `red → green → refactor → [red (loop) | done]`
- Retrofit (testing existing code): `red → refactor → [red (loop) | done]` (skips green)

**Two Modes:**
- `greenfield` (default) — New code; RED expects tests to fail
- `retrofit` (`--retrofit`) — Existing code; RED expects tests to pass, skips GREEN

**Spec Pick:** During the RED phase, agents must pick a spec with `tdd-ai spec pick <id>` before advancing. This focuses each iteration on a single requirement.

**Refactor Reflections:** During the refactor phase, 7 structured reflection questions are loaded (including a test-discovery question). Agents must answer all questions (min 5 words each) before advancing. Use `tdd-ai refactor reflect <n> --answer "..."` to answer and `tdd-ai refactor status` to view progress.

**Session File:** `.tdd-ai.json` in the working directory stores phase, mode, specs, test command, last test result, current spec ID, iteration count, reflections, and event history.

**Output Format:** All commands support `--format json` for machine-readable output. Format auto-detects: JSON when piped, text when in a terminal.

### Dependencies

- `github.com/spf13/cobra` — CLI framework
- `golang.org/x/term` — Terminal detection for auto-format
- No external databases, APIs, or services
