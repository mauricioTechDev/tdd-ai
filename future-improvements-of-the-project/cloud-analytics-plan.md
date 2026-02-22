# tdd-ai — Product Strategy & Future Direction

## Original Plan Summary

Add an optional cloud layer that persists session history and surfaces analytics, with a free open-source Community Edition and a paid Managed Cloud tier (~$8-12/month).

---

## Initial Review — What Worked

- **Open core model is the right call.** Keeping the CLI fully open source and charging for convenience (not features) builds trust and adoption. Self-hosters become evangelists.
- **Minimal data capture.** The session data being collected is lightweight and non-sensitive. Opt-in sync with `tdd-ai login` is the right UX.
- **The problem is real.** Session history and analytics are a natural extension. The tool already has structured data (phases, specs, iterations, reflections) — surfacing it over time is genuinely useful.
- **Lean tech stack.** Supabase free tier + Cloudflare Workers keeps costs near zero until there are paying users.

## Initial Review — Things to Refine

1. **Start even smaller on the dashboard.** Before building a full Next.js app, consider shipping a `tdd-ai history` CLI command that reads local session history from disk. This validates that people actually want to look back at past sessions before building a web UI.
2. **$8-12/month might be high for individual devs.** Consider $5/month or a generous free tier (e.g., 30 days of history free, pay for unlimited).
3. **Team/org features should come before they're needed for revenue.** Teams are where managed hosting becomes genuinely painful to DIY — that's the real wedge for paid conversion.
4. **"Language and framework detected" feels like scope creep.** Cut it from v1.
5. **Define what "session" means precisely.** Does a "session" end when the phase reaches DONE? When the user runs `tdd-ai init` again?
6. **Docker Compose self-host is table stakes but also a support burden.** Document it well and set expectations.

---

## Revised Strategy — Entire.io Changes the Picture

### What Entire.io Already Does

[Entire.io](https://entire.io) ([docs](https://docs.entire.io/core-concepts)) is an open-source (MIT) tool that already covers the session tracking and analytics layer:

- **Session capture** — automatically records AI coding sessions with full transcripts and code changes
- **Checkpoints** — snapshots tied to commits, stored as JSON metadata on a special `entire/checkpoints/v1` branch
- **Attribution** — tracks AI vs human code contributions
- **Token usage** — cost/usage metrics per session
- **Local-first architecture** — data lives in your repo's `.entire/` directory, synced via Git. GitHub is the data store, not Entire's servers
- **Web dashboard** — reads from the checkpoints branch to visualize activity

### What This Means for tdd-ai

The cloud analytics plan (session history, dashboards, managed hosting) overlaps heavily with what Entire already provides for free. Building that infrastructure would be duplicating solved work.

**Decision: Drop the cloud analytics plan.** Don't build session tracking, dashboards, or a managed hosting layer. Entire handles that. Focus tdd-ai on what Entire can't do.

### Monetization Timing

Don't worry about monetization yet. tdd-ai has:
- Zero infrastructure costs (stateless binary, no APIs, no hosting)
- No burn rate forcing revenue
- Not enough users yet to make a paid tier meaningful

**Monetization becomes relevant when:**
1. Meaningful usage numbers (hundreds of active developers)
2. Teams start asking for coordination features (shared TDD standards, org-wide compliance, team dashboards)
3. Features are added that have real marginal cost per user

**Until then, focus on adoption.** Growth is the only thing that matters right now.

---

## Core Thesis — Verification Infrastructure for AI

### The Asymmetry of Verification

Jason Wei's [Asymmetry of Verification and Verifier's Law](https://www.jasonwei.net/blog/asymmetry-of-verification-and-verifiers-law) argues that AI will solve any task that is easy to verify, provided five conditions are met:

1. **Objective truth** — consensus on what a good solution looks like
2. **Fast verification** — seconds per solution check
3. **Scalable verification** — simultaneous checking of many solutions
4. **Low noise** — tight correlation between verification signal and quality
5. **Continuous reward** — ability to rank solution goodness

### TDD Satisfies All Five Conditions

Tests are the perfect verification mechanism for AI-generated code:

| Condition | How TDD Satisfies It |
|---|---|
| Objective truth | Pass or fail, no ambiguity |
| Fast verification | Test suites run in seconds |
| Scalable verification | Run hundreds of tests in parallel |
| Low noise | A failing test points directly to what's wrong |
| Continuous reward | Measurable progress (3/10 passing → 7/10 → 10/10) |

TDD gives AI agents the tightest possible verification loop — write code, run tests, get instant unambiguous feedback, iterate. The tighter that loop, the faster AI converges on correct solutions.

### Where tdd-ai Fits vs Entire

| | tdd-ai | Entire.io |
|---|---|---|
| **Role** | Verification loop | Audit trail |
| **What it does** | Controls *how* the work happens | Tracks *what* happened |
| **Value** | Makes AI more effective right now | Records history for review |
| **Mechanism** | Enforces red/green/refactor discipline | Captures sessions and diffs |

These are **complementary, not competing**:
- **tdd-ai** = the verification infrastructure (makes each iteration more effective)
- **Entire** = the observability layer (records what happened)

### The Real Pitch

As AI gets smarter, the bottleneck shifts from "can AI write code" to "can we verify AI's code fast enough." tdd-ai isn't just a TDD tool — it's **verification infrastructure for AI agents**. The asymmetry of verification is *why* TDD-driven AI agents will outperform agents that just generate code and hope it works.

The value isn't in looking back at history — it's in making each iteration more effective right now.

---

## Revised Next Steps

1. **Focus on the core TDD loop.** Make tdd-ai the best possible verification state machine for AI agents
2. **Grow adoption.** Content about TDD with AI agents, integrations with more AI coding tools
3. **Let Entire handle persistence.** Session tracking and analytics are covered
4. **Revisit monetization when teams come knocking.** Enterprise TDD compliance, shared standards, and governance will be the revenue driver — but only after there's a user base
5. **Lean into the verification thesis.** Position tdd-ai as verification infrastructure, not just a TDD helper
