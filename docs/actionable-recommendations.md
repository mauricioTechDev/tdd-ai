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

## Implementation Sequence

Recommended order for implementing these changes:

| Order | Item | Reason |
|-------|------|--------|
| 1 | #2 Restrict `phase set` | Low effort, closes critical bypass |
| 2 | #1 `tdd-ai verify` | High leverage, enables CI integration |
| 3 | #6 Claude Code hook template | Strongest enforcement for AI agents |
| 4 | #3 Agent mode | Medium effort, comprehensive solution |
| 5 | #4 Compliance score in status | Nice-to-have, builds on verify logic |
| 6 | #7 Pre-commit hook | Simple gate for commits |
| 7 | #8 CI workflow | Final layer of enforcement |
