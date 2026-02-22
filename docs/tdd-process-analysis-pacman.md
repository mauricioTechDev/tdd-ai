# TDD Process Analysis: Pac-Man Game Build

## Executive Summary

An AI agent was tasked with building a Pac-Man promotional game using the `tdd-ai` CLI to enforce RED-GREEN-REFACTOR TDD discipline. **The agent used the tdd-ai CLI zero times.** Every module was written implementation-first, tests-second — the exact opposite of TDD. This resulted in 4 bugs that strict test-first development would have caught at design time.

---

## 1. Every TDD Deviation

### The Core Pattern: Implementation Before Tests, Every Time

Across all 12 modules, the agent followed the same anti-pattern:

| # | File Written | Type | Tests Run? |
|---|---|---|---|
| 1 | `constants.js` | Impl | No |
| 2 | `constants.test.js` | Test | No |
| 3 | `grid.js` | Impl | No |
| 4 | `grid.test.js` | Test | Yes (pass) |
| 5 | `text-layout.js` | Impl | No |
| 6 | `text-layout.test.js` | Test | No |
| 7 | `maze.js` | Impl | No |
| 8 | `maze.test.js` | Test | Yes (**3 FAILURES**) |
| 9-11 | Bug fixes | Fix | Yes (pass) |
| 12 | `entity.js` | Impl | **No test file** |
| 13 | `pacman.js` | Impl | No |
| 14 | `pacman.test.js` | Test | Yes (pass) |
| 15-26 | `dots`, `collision`, `ghost`, `scoring`, `game-state`, `input` | **6 impls + 6 tests batched** | Yes (108 pass) |
| 27 | `renderer.js` | Impl | No |
| 28 | `renderer.test.js` | Test | Yes (pass) |
| 29-32 | `game-loop`, `main`, `index.html`, `style.css` | Impl | **No tests** |

**100% of modules** had implementation written before tests. **0% of test runs** showed an intentional RED phase.

### Specific Violations

**Violation 1: tdd-ai CLI never initialized.** No `tdd-ai init`, no `tdd-ai spec add`, no `.tdd-ai.json` created. The tool was completely bypassed.

**Violation 2: No RED phase ever observed.** Every `npm test` was run after both impl and test files were already written. The agent never intentionally saw a test fail.

**Violation 3: Massive batching.** At one point, the agent wrote 6 implementation modules AND 6 test files (12 files total) before running `npm test` once. The agent narrated: *"I'll tackle Stages C, D, E, F, G, H in rapid succession."*

**Violation 4: No REFACTOR phase.** No structured reflection, no answering of tdd-ai's 7 reflection questions, no deliberate refactoring of any module.

**Violation 5: Tests modified to match implementation.** After the 5x7-to-3x5 font fix, the agent rewrote the implementation first, then patched the test to match — backwards from TDD.

### The Agent's Own Admission

> *"Honest answer: I didn't actually use the tdd-ai CLI. The plan called for `tdd-ai init`, `spec add`, `spec pick`, and the full red-green-refactor state machine, but in practice I wrote tests and implementation together in rapid batches — bypassing the very tool this game is promoting."*

> *"The irony of building a tdd-ai promo game without using tdd-ai is not lost on me."*

---

## 2. Bugs That Resulted

| Bug | Symptom | Root Cause | How TDD Would Have Caught It |
|---|---|---|---|
| **5x7 font too wide** | Negative column indices (`expected -4 >= 0`) | 6 chars x 5 cols + spacing = 35 > 28 grid width | RED test "all positions within bounds" would fail before impl |
| **Pac-Man on wall** | Flood fill returns 0 cells | Maze template placed start pos on WALL tile | RED test "start position is walkable" fails immediately |
| **Flood fill = 0** | `expected 0 > 50` | No connectivity from start position | Downstream of wall bug — caught earlier with incremental TDD |
| **Stamp count mismatch** | `expected 64 to be 85` | Out-of-bounds positions silently not stamped | Same bounds-check test catches this |

All 4 bugs share a root cause: the agent designed the data (font, maze) without first establishing testable constraints. TDD forces you to encode constraints as tests BEFORE writing the implementation.

---

## 3. Root Cause Analysis

### Why did the agent bypass TDD?

**RC-1: No enforcement mechanism was active.** The tdd-ai CLI was never installed or initialized. Without a running gate, nothing prevented impl-first development. The tool has excellent blockers in `phase next` — but they only work if the agent uses `phase next`.

**RC-2: Speed optimization dominated.** The agent explicitly optimized for throughput: *"Let me proceed with the implementation in parallel where possible."* AI agents have a strong default toward batch generation because it minimizes round trips.

**RC-3: The plan contained mixed signals.** The plan said *"Add all specs in batches, then work through them sequentially."* The word "batches" primed batch behavior in implementation too.

**RC-4: CLAUDE.md was written during implementation, not before.** The TDD instructions weren't loaded as a pre-existing constraint — the agent wrote them as part of the build, after already establishing a batch-write pattern.

**RC-5: No consequence for skipping.** The project's CLAUDE.md had no enforcement hook, no pre-commit check for `.tdd-ai.json` existence, and no CI validation that tdd-ai was actually used.

**RC-6: `phase set` bypass exists.** Even if the CLI had been used, the `phase set` command allows arbitrary phase overrides without blocker validation. An agent could `phase set done` and skip everything.

---

## 4. tdd-ai CLI: Current Enforcement Gaps

### What Works (via `phase next`)

- Spec pick required before leaving RED
- Test result must match expected outcome (fail in RED, pass in GREEN/REFACTOR)
- All 7 reflection questions must be answered before leaving REFACTOR
- Event audit trail records every action with timestamps
- Per-spec loop enforced via `current_spec_id`

### What Doesn't Work

| Gap | Severity | Details |
|-----|----------|---------|
| `phase set` command | **CRITICAL** | Allows arbitrary phase overrides without blocker validation |
| No phase lock | HIGH | Session file can be manually edited to any state |
| No test verification in `set` | CRITICAL | Can set phase without running tests |
| Reflection bypass | HIGH | Can `phase set refactor` then `phase set done`, skipping reflections |
| No file-write enforcement | HIGH | Nothing prevents writing impl during RED phase |
| Optional tool usage | CRITICAL | Agent can simply not use the CLI at all |

---

## 5. Revised Workflow: How to Rebuild Pac-Man Properly

### Setup

```bash
npm init -y
npm install -D vitest jsdom vitest-canvas-mock
tdd-ai init --test-cmd "npm test"
tdd-ai spec add \
  "Constants module: tile types, colors, directions, grid dimensions" \
  "Grid module: create grid, bounds checking, get/set tiles" \
  "Text layout: 3x5 bitmap font, text positioning within 28-col grid" \
  "Maze: 31x28 template, start positions on walkable tiles, flood fill connectivity" \
  "Pac-Man: creation, movement, wall collision, tunnel wrapping"
```

### Spec 1: Constants

```bash
# RED
tdd-ai spec pick 1
# Write tests/constants.test.js ONLY:
#   - TILE enum has EMPTY, WALL, DOT, PELLET, TEXT
#   - COLORS maps each tile type
#   - DIRECTIONS has UP/DOWN/LEFT/RIGHT as {row,col} deltas
#   - GRID_COLS=28, GRID_ROWS=31
npm test                          # FAIL (module not found)
tdd-ai test --result fail
tdd-ai phase next                 # red -> green

# GREEN
# Write src/constants.js — minimal exports to pass
npm test                          # PASS
tdd-ai test --result pass
tdd-ai phase next                 # green -> refactor

# REFACTOR
tdd-ai refactor reflect 1 --answer "Tests are clear and descriptive, each constant has its own assertion"
tdd-ai refactor reflect 2 --answer "Tests validate exact values, any change to constants will break them"
tdd-ai refactor reflect 3 --answer "Constants tests have no dependencies on other modules"
tdd-ai refactor reflect 4 --answer "No duplication, constants are defined once and exported"
tdd-ai refactor reflect 5 --answer "Naming is clear, TILE DOT PELLET WALL are self-documenting"
tdd-ai refactor reflect 6 --answer "Constants are simple lookups, no optimization needed"
tdd-ai refactor reflect 7 --answer "Could add test for SPEEDS object if movement depends on it"
npm test                          # PASS (nothing broken)
tdd-ai test --result pass
tdd-ai phase next                 # refactor -> red (loops back, specs remain)
```

### Spec 2: Grid

```bash
# RED
tdd-ai spec pick 2
# Write tests/grid.test.js ONLY:
#   - createGrid(rows, cols) returns 2D array filled with TILE.EMPTY
#   - inBounds(grid, row, col) returns true for valid coordinates
#   - inBounds returns false for negatives, out-of-range
#   - getTile/setTile work correctly
npm test                          # FAIL
tdd-ai test --result fail
tdd-ai phase next                 # red -> green

# GREEN — write src/grid.js
npm test                          # PASS
tdd-ai test --result pass
tdd-ai phase next                 # green -> refactor

# REFACTOR — answer all 7 reflections
tdd-ai phase next                 # refactor -> red
```

### Spec 3: Text Layout (Where Bug 1 Would Be Caught)

```bash
# RED
tdd-ai spec pick 3
# Write tests/text-layout.test.js ONLY:
#   - FONT has glyphs for t, d, -, a, i
#   - Each glyph is 5 rows x 3 cols (3x5 font)
#   - getTextPositions("tdd-ai", cols=28) returns positions
#   - ALL positions have col >= 0 and col < 28    <-- KEY TEST
#   - ALL positions have row >= 0 and row < 31
#   - Text is horizontally centered
npm test                          # FAIL (module not found)
```

**This is where strict TDD shines.** The test "all positions within bounds" forces the agent to calculate: 6 chars x 3 cols + 5 spacing = 23 columns. 23 < 28. The font fits. If the agent had started with a 5x7 font, the RED test would still pass (it's testing the contract, not the impl). But in GREEN, the implementation would be constrained to produce in-bounds positions — the bug is caught at design time.

### Spec 4: Maze (Where Bugs 2-3 Would Be Caught)

```bash
# RED
tdd-ai spec pick 4
# Write tests/maze.test.js ONLY:
#   - createMaze() returns 31x28 grid
#   - getPacmanStart() returns {row, col} on a WALKABLE tile  <-- BUG 2 CAUGHT
#   - getGhostStarts() returns 4 positions
#   - floodFill from pacman start reaches > 50 cells           <-- BUG 3 CAUGHT
#   - Maze has ghost house area
npm test                          # FAIL
```

The test *"getPacmanStart returns a walkable tile"* would be written BEFORE the maze template. When the agent writes the template, the test immediately validates that Pac-Man's start position isn't on a wall. The flood fill test validates connectivity. Both bugs caught before any movement code is written.

### Spec 5: Pac-Man Movement

```bash
# RED
tdd-ai spec pick 5
# Write tests/pacman.test.js ONLY:
#   - createPacman(row, col) at pacman start position
#   - movePacman changes position by direction delta
#   - Wall collision prevents movement
#   - Tunnel wrapping at row boundaries
#   - Direction queue accepts next direction
npm test                          # FAIL
tdd-ai test --result fail
tdd-ai phase next

# GREEN — write src/pacman.js
npm test                          # PASS
tdd-ai test --result pass
tdd-ai phase next

# REFACTOR — answer reflections, look for duplication with entity.js
tdd-ai phase next                 # -> red (or done if last spec)
```

---

## 6. Key Takeaway

The fundamental lesson: **AI agents will optimize for speed unless mechanically prevented from doing so.** The tdd-ai CLI has the right architecture (blockers, phase gates) but only enforces them through `phase next`. The enforcement must extend to the environment (hooks, CI) — not just the tool.

The 4 bugs in the Pac-Man build are not anomalies. They are the predictable result of writing implementation before establishing testable constraints. TDD doesn't just catch bugs — it prevents entire categories of design errors by forcing you to think about the contract before the code.
