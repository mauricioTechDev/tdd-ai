# tdd-ai Cloud Analytics — Product Plan Review

## Original Plan Summary

Add an optional cloud layer that persists session history and surfaces analytics, with a free open-source Community Edition and a paid Managed Cloud tier (~$8-12/month).

---

## What Works Well

- **Open core model is the right call.** Keeping the CLI fully open source and charging for convenience (not features) builds trust and adoption. Self-hosters become evangelists.
- **Minimal data capture.** The session data being collected is lightweight and non-sensitive. Opt-in sync with `tdd-ai login` is the right UX.
- **The problem is real.** Session history and analytics are a natural extension. The tool already has structured data (phases, specs, iterations, reflections) — surfacing it over time is genuinely useful.
- **Lean tech stack.** Supabase free tier + Cloudflare Workers keeps costs near zero until there are paying users.

---

## Things to Refine

1. **Start even smaller on the dashboard.** Before building a full Next.js app, consider shipping a `tdd-ai history` CLI command that reads local session history from disk. This validates that people actually want to look back at past sessions before building a web UI. Ship the analytics loop inside the tool people already use.

2. **$8-12/month might be high for individual devs.** This is competing with "I could just not have analytics." Consider $5/month or a generous free tier (e.g., 30 days of history free, pay for unlimited). The goal is conversion volume, not per-seat revenue early on.

3. **Team/org features should come before they're needed for revenue.** Individual developers can self-host easily. Teams are where managed hosting becomes genuinely painful to DIY — that's the real wedge for paid conversion. Move team dashboards higher in priority.

4. **"Language and framework detected" feels like scope creep.** The CLI doesn't do this today, and it's not trivial to get right. Cut it from v1 — the valuable data (iterations, pass/fail, duration, files touched) exists without it.

5. **Define what "session" means precisely.** Right now `.tdd-ai.json` is a single mutable file. Decide: does a "session" end when the phase reaches DONE? When the user runs `tdd-ai init` again? When they explicitly close it? This matters for the data model and should be settled before building the schema.

6. **Docker Compose self-host is table stakes but also a support burden.** Document it well and set expectations — "community supported, not enterprise supported." Otherwise time gets spent debugging people's Docker setups instead of building product.

---

## Recommended Order of Next Steps

1. Define what a "session" boundary is (start/end events)
2. Add local session history — save completed sessions to `~/.tdd-ai/history/`
3. Ship `tdd-ai history` and `tdd-ai stats` CLI commands
4. Validate that people use them
5. *Then* build the backend API and dashboard
6. `tdd-ai login` and sync come after the data model is proven right

---

## Core Takeaway

The core insight — that the structured TDD data already being generated has value over time — is correct. The risk is building the cloud layer before validating that users actually want to review past sessions. Ship the local version first, learn from usage, then add sync.
