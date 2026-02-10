# tdd-ai CLI Improvement Plan

Use this prompt to implement 9 improvements to the tdd-ai CLI. Use the tdd-ai CLI itself to follow TDD while building these features.

## Context

tdd-ai is a Go CLI built with Cobra that acts as TDD guardrails for AI coding agents. It tracks specs (what to build), manages the red-green-refactor phase cycle, and provides structured guidance. It does NOT run tests — the AI agent runs tests itself.

These improvements come from analyzing a real conversation where an LLM used the CLI to add integration tests to an existing .NET API. The session exposed ergonomic gaps, missing guardrails, and a workflow that didn't fit the "testing existing code" scenario.

## Codebase Structure

```
main.go                              # Entry point, calls cmd.Execute()
Makefile                             # build, test, lint, install targets
go.mod                               # module: github.com/macosta/tdd-ai, go 1.25.7, cobra v1.10.2

cmd/
  root.go                            # Root command, --format flag (text|json), getWorkDir()
  init.go                            # tdd-ai init — creates .tdd-ai.json, prints next-step hint
  spec.go                            # tdd-ai spec add|list|done — manage specs
  phase.go                           # tdd-ai phase|phase next|phase set — manage phases
  guide.go                           # tdd-ai guide — phase-appropriate instructions
  status.go                          # tdd-ai status — session overview (same output as spec list)
  reset.go                           # tdd-ai reset — delete session file
  version.go                         # tdd-ai version

internal/
  types/types.go                     # Phase, Spec, Session, Guidance structs
  types/types_test.go                # Unit tests for types
  session/session.go                 # File I/O: Create, Load, Save, Exists, LoadOrFail
  session/session_test.go            # Unit tests for session
  phase/machine.go                   # State machine: Next(), CanTransition()
  phase/machine_test.go              # Unit tests for phase transitions
  guide/guide.go                     # Generate() — phase-specific instructions/rules
  guide/guide_test.go                # Unit tests for guide
  formatter/formatter.go             # FormatGuidance(), FormatStatus() — text and JSON output
  formatter/formatter_test.go        # Unit tests for formatter
```

Session state lives in `.tdd-ai.json`:
```json
{
  "phase": "red",
  "specs": [
    { "id": 1, "description": "...", "status": "active" }
  ],
  "next_id": 2
}
```

## Improvements to Implement

### 1. `spec done` should accept multiple IDs

**Problem**: Marking 7 specs done required chaining 7 commands: `tdd-ai spec done 1 && tdd-ai spec done 2 && ...`

**Change**: Update `spec done` to accept one or more IDs. Add an `--all` flag to mark every active spec as done.

```bash
# Current (keep working)
tdd-ai spec done 1

# New
tdd-ai spec done 1 2 3 4 5 6 7
tdd-ai spec done --all
```

**Files to change**: `cmd/spec.go` (change `cobra.ExactArgs(1)` to `cobra.MinimumNArgs(0)`, add `--all` flag, loop over args), `internal/types/types.go` (add `CompleteAllSpecs()` method), plus tests.

---

### 2. `spec add` should support batch input

**Problem**: Adding 7 specs required 7 separate commands, each doing a full load/save cycle.

**Change**: Accept multiple descriptions as separate args.

```bash
# Current (keep working)
tdd-ai spec add "GET /users returns 200"

# New
tdd-ai spec add "GET /users returns 200" "GET /users returns 404" "POST /users returns 201"
```

**Files to change**: `cmd/spec.go` (change `cobra.ExactArgs(1)` to `cobra.MinimumNArgs(1)`, loop over args, single load/save), plus tests.

---

### 3. `init` should support a `--retrofit` mode for testing existing code

**Problem**: The RED phase guidance says "verify ALL new tests FAIL" and "DO NOT create implementation files." But when adding tests to an already-implemented API, all tests pass immediately. The guidance was wrong for this use case and every phase was meaningless.

**Change**: Add a `--retrofit` flag to `init` that stores a `mode` field in the session (`"mode": "retrofit"` vs `"mode": "greenfield"`). The guide command reads this mode and adjusts instructions accordingly.

Retrofit RED phase guidance should say things like:
- "Write tests that verify the existing behavior of the active specs."
- "Run the project's test command to verify ALL new tests PASS against the existing implementation."
- "If tests fail, determine whether the test is wrong or the implementation has a bug."

Retrofit GREEN phase should be skipped or adjusted (implementation already exists). Retrofit REFACTOR phase stays the same.

**Session format change**:
```json
{
  "phase": "red",
  "mode": "greenfield",
  "specs": [],
  "next_id": 1
}
```

Default mode is `"greenfield"` (current behavior, backward-compatible). When `mode` is omitted from an existing file, treat it as `"greenfield"`.

**Files to change**: `internal/types/types.go` (add `Mode` field to Session), `cmd/init.go` (add `--retrofit` flag), `internal/guide/guide.go` (add retrofit variants of instructions/rules), `internal/formatter/formatter.go` (include mode in output), plus tests. The phase machine in `internal/phase/machine.go` should allow `red -> refactor` transition when mode is `"retrofit"` (skipping green since implementation exists).

---

### 4. `phase next` should validate phase expectations

**Problem**: `phase next` blindly advances with zero checks. The LLM advanced from RED to GREEN with all tests passing (RED expects failing tests). The state machine enforces order but not substance.

**Change**: Add an optional `--test-result` flag to `phase next` that accepts `pass` or `fail`. When provided, validate it matches the phase expectation:
- RED phase: expects `--test-result fail` (tests should be failing)
- GREEN phase: expects `--test-result pass` (tests should be passing)
- REFACTOR phase: expects `--test-result pass` (tests should still pass)

When `--test-result` is NOT provided, print a warning reminding the agent what it should have verified, but still allow the transition (don't break backward compatibility).

In retrofit mode, RED phase expects `--test-result pass` instead of `fail`.

```bash
# Current (keep working, but now prints a warning)
tdd-ai phase next

# New (validates)
tdd-ai phase next --test-result fail   # in red phase — OK
tdd-ai phase next --test-result pass   # in red phase — ERROR
```

**Files to change**: `cmd/phase.go` (add `--test-result` flag, validation logic), `internal/phase/machine.go` (add `ExpectedTestResult(phase, mode)` function), plus tests.

---

### 5. Help text should include examples

**Problem**: Running `--help` on any command returned almost nothing useful. The LLM had to trial-and-error its way through the CLI.

**Change**: Add Cobra `Example` fields to every command. Also add a `Long` description where missing.

Examples of what to add:

```go
// spec add
Example: `  tdd-ai spec add "User can login with email and password"
  tdd-ai spec add "Returns 404 when not found" "Returns 400 for invalid input"`,

// spec done
Example: `  tdd-ai spec done 1
  tdd-ai spec done 1 2 3
  tdd-ai spec done --all`,

// phase next
Example: `  tdd-ai phase next
  tdd-ai phase next --test-result fail`,

// init
Example: `  tdd-ai init
  tdd-ai init --retrofit`,

// guide
Example: `  tdd-ai guide
  tdd-ai guide --format json`,
```

**Files to change**: All files in `cmd/` — add `Example` and improve `Long` on each `cobra.Command`.

---

### 6. `status` should be richer than `spec list`

**Problem**: `status` and `spec list` produce identical output via the same `FormatStatus` function. There's no reason for two commands.

**Change**: Make `status` a richer overview that includes:
- Current phase
- Session mode (greenfield/retrofit)
- Spec summary (total, active, done)
- All specs with status
- The recommended next action (e.g., "Next: run 'tdd-ai guide --format json' to get instructions")

Keep `spec list` as the simpler spec-only view.

**Files to change**: `cmd/status.go`, `internal/formatter/formatter.go` (add a `FormatFullStatus` or enhance `FormatStatus` with a `verbose` option), plus tests.

---

### 7. Every command should hint the next action

**Problem**: `init` says "Next: add specs with ..." but most other commands give no hint. After `spec add`, the LLM didn't know to run `guide` next. The breadcrumb trail was broken.

**Change**: Add a "Next:" line to the text output of every mutating command:

| Command | Next hint |
|---------|-----------|
| `init` | `Next: add specs with 'tdd-ai spec add "description"'` (already exists) |
| `spec add` | `Next: run 'tdd-ai guide --format json' for phase instructions` |
| `spec done` | `Next: add more specs or run 'tdd-ai reset' to start over` (if all done) |
| `phase next` | `Next: run 'tdd-ai guide --format json' for phase instructions` (if not done) |
| `phase next` to done | `Next: mark completed specs with 'tdd-ai spec done <id>'` |
| `phase set` | `Next: run 'tdd-ai guide --format json' for phase instructions` |

Only add hints in text format, not JSON (agents using JSON parse structured output, they don't need hints).

**Files to change**: `cmd/spec.go`, `cmd/phase.go` — add `fmt.Fprintln` after the main output line.

---

### 8. `spec list` should sort specs by ID

**Problem**: In the observed session, specs appeared in a non-sequential order in the status output (5, 6, 7 instead of 1-7). Whether from the session file or display, the output should always be sorted by ID for readability.

**Change**: Sort specs by ID before displaying in both `FormatStatus` and `FormatGuidance`.

**Files to change**: `internal/formatter/formatter.go` (sort the specs slice by ID before rendering), plus tests.

---

### 9. `phase next` should require at least one active spec

**Problem**: The LLM could advance through phases with no specs at all. The TDD cycle is meaningless without specs to implement.

**Change**: `phase next` should return an error if there are no active specs when advancing from RED to GREEN. It's valid to have no active specs when moving from REFACTOR to DONE (you might have marked them all done already).

**Files to change**: `cmd/phase.go` (add spec count check before transition from red), plus tests.

---

## Implementation Order

Use `tdd-ai` itself to TDD these changes. Suggested order (simplest first, dependencies respected):

1. **Improvement 8** — Sort specs by ID (isolated formatter change)
2. **Improvement 5** — Help text examples (no logic changes, just strings)
3. **Improvement 1** — `spec done` accepts multiple IDs + `--all`
4. **Improvement 2** — `spec add` accepts multiple descriptions
5. **Improvement 7** — Next-action hints on all commands
6. **Improvement 6** — Richer `status` command
7. **Improvement 9** — `phase next` requires active specs
8. **Improvement 3** — Retrofit mode (largest change, touches most files)
9. **Improvement 4** — `phase next --test-result` validation (depends on retrofit mode for full behavior)

## Build & Test

```bash
make test          # Run all tests
make build         # Build binary to bin/tdd-ai
make lint          # go vet
```

After all changes, bump the version in the Makefile from `0.1.0` to `0.2.0`, update the README to document the new features, and do a final `make test && make build`.
