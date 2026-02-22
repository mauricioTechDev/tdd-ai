# Actionable Recommendations: Enforcing TDD Discipline on AI Agents

Recommendations derived from the [Pac-Man TDD Process Analysis](./tdd-process-analysis-pacman.md). Each item is scoped for implementation as a standalone change to the tdd-ai CLI or its ecosystem.

---

## CLI Changes

### 1. Add `tdd-ai verify` command

**Priority:** High
**Effort:** Medium

A post-hoc compliance command that analyzes the event history and reports TDD violations.

```
$ tdd-ai verify
✗ VIOLATION: No RED phase recorded for spec 3
✗ VIOLATION: phase_set used 2 times (should be 0)
✗ VIOLATION: No test_run events between phase transitions
✓ 5/8 specs followed proper RED-GREEN-REFACTOR
Score: 62% TDD compliance
```

**What to check:**
- Every spec has a `spec_picked` event
- A `test_run` with result `fail` exists after each `spec_picked` (RED observed)
- A `phase_next` from red->green exists after each failing test run
- A `test_run` with result `pass` exists after entering GREEN
- A `phase_next` from green->refactor exists after passing test
- All 7 reflection questions answered before leaving REFACTOR
- No `phase_set` usage (flag as violation)
- Timestamp ordering is correct (no out-of-sequence events)

**Exit codes:** 0 = compliant, 1 = violations found. This lets CI hooks consume the result.

---

### 2. Restrict `phase set` with `--force` flag

**Priority:** High
**Effort:** Low

Currently `phase set` bypasses all blocker validation. Add friction:

```go
// In cmd/phase.go phaseSetCmd
if !forceFlag {
    return fmt.Errorf(
        "phase set bypasses TDD guardrails. Use --force to override.\n" +
        "Prefer 'tdd-ai phase next' for normal phase advancement.")
}
// Log as violation in event history
s.AddEvent(types.Event{
    Action: "phase_set",
    From:   string(s.Phase),
    To:     string(p),
    Result: "forced_override",
})
```

**Files to modify:**
- `cmd/phase.go` — add `--force` flag, update `phaseSetCmd`
- `internal/types/types.go` — no changes needed (event already supports this)

---

### 3. Add agent mode (`tdd-ai init --agent`)

**Priority:** Medium
**Effort:** Medium

When initialized with `--agent`, apply stricter enforcement:

- `phase set` is disabled entirely (returns error)
- `complete` requires `--force` flag
- All outputs include explicit "next action" instructions
- Session file records `"agent_mode": true`

```bash
$ tdd-ai init --test-cmd "npm test" --agent
Initialized in agent mode. phase set is disabled.
```

**Files to modify:**
- `cmd/init.go` — add `--agent` flag
- `internal/types/types.go` — add `AgentMode bool` to Session
- `cmd/phase.go` — check `AgentMode` in `phaseSetCmd`
- `cmd/complete.go` — check `AgentMode` for `--force` requirement

---

### 4. Add compliance score to `tdd-ai status`

**Priority:** Medium
**Effort:** Low

Show a TDD compliance percentage in `tdd-ai status` output based on event history analysis. Reuse the logic from `tdd-ai verify`.

```
$ tdd-ai status
Phase:      refactor
Mode:       greenfield
Iteration:  3
Specs:      2/5 done
Compliance: 100% (3/3 specs followed RED-GREEN-REFACTOR)
```

**Files to modify:**
- `internal/formatter/formatter.go` — add compliance calculation
- `cmd/status.go` — include in output

---

### 5. Ship a `tdd-ai generate claude-md` command

**Priority:** Medium
**Effort:** Low

Generate a CLAUDE.md template with all enforcement instructions pre-written:

```bash
$ tdd-ai generate claude-md > CLAUDE.md
```

Output includes:
- Mandatory tdd-ai initialization block
- Anti-batching instructions
- One-spec-at-a-time constraint
- Verification instructions
- Explicit prohibitions (no `phase set`, no impl during RED)

**Template content:**

```markdown
## MANDATORY: TDD Workflow (Non-Negotiable)

Before writing ANY code (implementation OR test), you MUST:

1. Run `tdd-ai init --test-cmd "<test-command>"`
2. Run `tdd-ai spec add` for each module
3. For EACH spec, follow this EXACT sequence:
   a. `tdd-ai spec pick <id>`
   b. Write the TEST file ONLY — no implementation
   c. Run tests — verify the test FAILS (RED)
   d. `tdd-ai test --result fail`
   e. `tdd-ai phase next`
   f. Write MINIMUM implementation to pass
   g. Run tests — verify it PASSES (GREEN)
   h. `tdd-ai test --result pass`
   i. `tdd-ai phase next`
   j. Refactor, answer all reflections
   k. `tdd-ai phase next`

NEVER write implementation and test files in the same step.
NEVER batch multiple modules together.
NEVER use `tdd-ai phase set` — only use `tdd-ai phase next`.

## ONE SPEC AT A TIME

Complete the full RED-GREEN-REFACTOR cycle for ONE spec before
starting the next. Do NOT:
- Write multiple test files before running any
- Write implementation alongside tests
- Optimize for speed over discipline

## Verification

After completing all specs, run `tdd-ai status --format json`.
The output MUST show:
- Every spec status = "done"
- Event history contains phase_next transitions (NOT phase_set)
- Iteration count matches number of completed specs
```

**Files to add:**
- `cmd/generate.go` — new command with `claude-md` subcommand
- `internal/templates/claude-md.go` — template content

---

## Hook / CI Integration

### 6. Ship a Claude Code hook template for file-write gating

**Priority:** High
**Effort:** Medium

Provide a `.claude/hooks/tdd-guard.sh` that checks the current tdd-ai phase before file writes and blocks implementation writes during RED.

```bash
#!/bin/bash
# .claude/hooks/tdd-guard.sh
# Block implementation file writes during RED phase

# Only check Write/Edit tool calls
SESSION=".tdd-ai.json"
[ ! -f "$SESSION" ] && exit 0

PHASE=$(jq -r '.phase' "$SESSION")
FILE_PATH="$1"

# During RED phase, only test files should be written
if [ "$PHASE" = "red" ]; then
  if [[ "$FILE_PATH" != *test* && "$FILE_PATH" != *spec* && "$FILE_PATH" != *.test.* ]]; then
    echo "BLOCKED: Cannot write implementation files during RED phase."
    echo "Current phase: RED — write tests only."
    echo "File attempted: $FILE_PATH"
    exit 1
  fi
fi
```

Register in `.claude/settings.local.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hook": ".claude/hooks/tdd-guard.sh"
      }
    ]
  }
}
```

**Files to add:**
- Template hook script (shipped with docs or via `tdd-ai generate hook`)

---

### 7. Pre-commit hook: verify `.tdd-ai.json` phase is DONE

**Priority:** Medium
**Effort:** Low

```bash
#!/bin/bash
# .claude/hooks/tdd-ai-commit-check.sh

if [ -f .tdd-ai.json ]; then
  PHASE=$(jq -r '.phase' .tdd-ai.json)
  if [ "$PHASE" != "done" ]; then
    echo "ERROR: TDD cycle not complete. Current phase: $PHASE"
    echo "Complete the RED-GREEN-REFACTOR cycle before committing."
    exit 1
  fi
fi
```

---

### 8. GitHub Actions CI check for tdd-ai compliance

**Priority:** Low
**Effort:** Low

```yaml
# .github/workflows/tdd-check.yml
name: TDD Compliance
on: [push, pull_request]
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install tdd-ai
        run: go install github.com/mauricioTechDev/tdd-ai@latest
      - name: Verify TDD compliance
        run: |
          if [ -f .tdd-ai.json ]; then
            tdd-ai verify --strict
          fi
```

---

## Prompt Engineering

### 9. Anti-batching instructions for CLAUDE.md

**Priority:** High
**Effort:** None (copy-paste into project CLAUDE.md)

```markdown
## Anti-Batching Rules

- Each tool call that writes a file MUST be followed by a test run
  before writing the next file
- Before writing ANY file, run `tdd-ai guide` and follow its output
- After writing a test file:
  1. Run the tests
  2. Confirm at least one NEW test FAILS
  3. Record: `tdd-ai test --result fail`
  4. Only THEN write implementation
- NEVER write more than one source file between test runs
```

---

### 10. Negative examples in CLAUDE.md

**Priority:** Medium
**Effort:** None (copy-paste into project CLAUDE.md)

```markdown
## What NOT To Do

BAD (batching):
  Write constants.js -> Write constants.test.js -> Write grid.js -> Write grid.test.js -> npm test

BAD (impl before test):
  Write grid.js -> Write grid.test.js -> npm test

BAD (parallel modules):
  "I'll tackle Stages C, D, E, F, G, H in rapid succession"

GOOD (one spec, test first):
  Write grid.test.js -> npm test (FAIL) -> tdd-ai test --result fail ->
  tdd-ai phase next -> Write grid.js -> npm test (PASS) ->
  tdd-ai test --result pass -> tdd-ai phase next -> refactor -> tdd-ai phase next
```

---

## Implementation Sequence

Recommended order for implementing these changes:

| Order | Item | Reason |
|-------|------|--------|
| 1 | #2 Restrict `phase set` | Low effort, closes critical bypass |
| 2 | #9 Anti-batching instructions | Zero effort, immediate impact on new projects |
| 3 | #10 Negative examples | Zero effort, immediate impact |
| 4 | #1 `tdd-ai verify` | High leverage, enables CI integration |
| 5 | #6 Claude Code hook template | Strongest enforcement for AI agents |
| 6 | #5 `generate claude-md` | Reduces friction for new projects |
| 7 | #3 Agent mode | Medium effort, comprehensive solution |
| 8 | #4 Compliance score in status | Nice-to-have, builds on verify logic |
| 9 | #7 Pre-commit hook | Simple gate for commits |
| 10 | #8 CI workflow | Final layer of enforcement |
