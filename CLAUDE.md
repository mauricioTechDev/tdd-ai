# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build        # Build binary to bin/tdd-ai (injects version via ldflags)
make test         # Run all tests: go test ./... -v
make test-short   # Run tests without I/O: go test ./... -short
make lint         # Static analysis: go vet ./...
make install      # Copy binary to ~/go/bin/
make clean        # Remove bin/ directory
```

Run a single package's tests:
```bash
go test ./internal/phase/ -v -run TestNext
```

## Architecture

**tdd-ai** is a Go CLI tool that enforces TDD discipline for AI coding agents. It does NOT run tests itself — the AI agent runs tests using whatever framework the project uses. The tool provides a state machine, phase-appropriate guidance, and structured output.

### Package Layout

- `main.go` — Entry point, calls `cmd.Execute()`
- `cmd/` — Cobra CLI commands. Each command follows: load session → validate → mutate → save → output
- `internal/types/` — Core data structures: `Phase`, `Mode`, `Spec`, `Session`, `Guidance`, `Event`
- `internal/session/` — Session persistence (read/write `.tdd-ai.json` in working directory)
- `internal/phase/` — State machine: `Next()`, `NextWithMode()`, `ExpectedTestResult()`, `CanTransition()`
- `internal/guide/` — Generates phase-specific instructions and rules based on current phase and mode
- `internal/formatter/` — Formats output as text or JSON (`FormatGuidance`, `FormatStatus`, `FormatFullStatus`)

### Key Concepts

**TDD Phase State Machine:**
- Greenfield: `red → green → refactor → done`
- Retrofit (testing existing code): `red → refactor → done` (skips green)

**Two Modes:**
- `greenfield` (default) — New code; RED expects tests to fail
- `retrofit` (`--retrofit`) — Existing code; RED expects tests to pass, skips GREEN

**Session File:** `.tdd-ai.json` in the working directory stores phase, mode, specs, test command, last test result, and event history.

**Output Format:** All commands support `--format json` for machine-readable output. Format auto-detects: JSON when piped, text when in a terminal.

### Dependencies

- `github.com/spf13/cobra` — CLI framework
- `golang.org/x/term` — Terminal detection for auto-format
- No external databases, APIs, or services
