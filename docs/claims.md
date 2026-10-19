# Claims: verify once, check on every run

Status: experimental. Linux only for tracing.

## Why

Agents loop. They try a tool, read the result, try again, often dozens of times
per task and with nobody reviewing each step. A review that happens once,
before the loop starts, does not survive that: a script changes, a remote
server changes, a model update changes what a skill does.

Claims split validation into two speeds:

1. **Verify (slow, once).** Run the command under a tracer and record what it
   actually did. The agent judges the claim against that evidence and against
   the files that matter (a `SKILL.md`, a script). This is where judgment and
   tokens get spent.
2. **Check (fast, always).** Distill the accepted evidence into deterministic
   checks and rerun them on every agent loop and in CI. No model involved.

The design rule is the same one the TDD flow follows: tdd-ai records what it
observed, not what an agent reports.

## Commands

| Command | What it does |
|---------|--------------|
| `tdd-ai claim add "claim" [--pin file]... -- <cmd> [args...]` | Record a claim and the command that exercises it |
| `tdd-ai claim list` | Show claims and their status |
| `tdd-ai claim verify <id>` | Run the command under `strace` and record evidence |
| `tdd-ai claim distill <id> --note "why"` | Turn accepted evidence into checks |
| `tdd-ai claim check [id...]` | Rerun the checks; exit 1 on any failure |

All of them support `--format json`.

## What verify records

- **Reads and writes inside the project**, as project-relative paths. Writes
  include creating, truncating, renaming and removing files.
- **Reads and writes outside the project**, as `~/...` or absolute paths.
  Runtime and system files (`/usr`, `/lib`, `/proc`, the loader cache, glibc's
  locale alias table, the interpreter binary itself, even when it is
  symlinked) and the temp directory are left out.
- **Network activity**: every destination of `connect`, `sendto` and `sendmsg`
  (`1.2.3.4:443`, `[::1]:9`, `unix:/run/x.sock`), listening addresses
  (`bind:0.0.0.0:8080`) and internet sockets opened (`socket:inet`). Failed
  attempts count: an attempt is what matters.
- **Programs run**, by name. The first is the command itself.
- **Exit code and output**, hashed, with the output saved for distill.
- **Pinned files**, hashed.

## What distill does

It proposes checks from the evidence: the command itself, pins, reads and
writes generalized to directory globs (`invoices/march.csv` becomes
`invoices/**`), the exit code and the expected output (saved to
`.tdd-ai/expected/<id>.out`).

It refuses, and says why, when the evidence needs a decision tdd-ai will not
make on its own:

- network activity (accept with `--allow-net`),
- anything outside the project (`--allow-external`),
- other programs (`--allow-exec`),
- a pinned file that changed after it was observed (verify again),
- `--allow-read` or `--allow-write` globs that don't cover what was observed.

`--note` is required and must say why the evidence backs the claim. It is the
record of the slow step's judgment.

## What check compares

The command, pins, exit code and output always; reads, writes, outside paths,
network and programs when traced. Editing the claim's command, a pinned file or
the expected output after distill fails the check until the claim is verified
again.

`claim check` runs the commands recorded in `.tdd-ai/claims.json`. Review that
file in pull requests the way you review test scripts: changes to its checks
change what is enforced. `--no-trace` skips the tracer for macOS and Windows and
checks pins, exit code and output only. Claims that were never distilled are
reported as pending; `--strict` fails on them.

## Limits of this first version

- Tracing uses `strace`, so `verify` needs Linux (a container or CI on macOS).
- Only successful file opens are recorded; failed opens are not.
- Directory listings are recorded but not enforced.
- Relative paths in `rename`, `unlink` and `mkdir` resolve against a tracked
  working directory; a child that starts before its parent's `clone` returns
  is assumed to share the starting directory.
- A claim is about one command. Claims about what an agent does with a skill
  (running the agent itself under observation) and pinning remote MCP tool
  descriptions are natural next steps.
