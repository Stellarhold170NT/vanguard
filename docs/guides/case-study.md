---
title: Case Study — Real-Fire Baseline
description: What Vanguard's first scan of a production-shaped 182-endpoint Spring service looks like — recall, FP/FN triage, and the shape of the remediation backlog.
---

# Case Study: Real-Fire Baseline

This page summarizes a full audit Vanguard ran against a
production-shaped Spring Boot service (182 REST endpoints, gRPC services,
753 source files). Every number below comes from machine-generated artifacts
produced by the audit protocol — see [FP/FN Audit Protocol](/fpfn-protocol)
for the labeling methodology.

## Headline Numbers

| Metric | Value |
|---|---|
| Endpoints discovered | 182 (recall **100%** vs the swagger inventory) |
| Findings | 244 — 24 ERROR / 143 WARN / 77 INFO |
| After triage | 129 TP / 115 FP / 0 unresolved |
| **ERROR-tier precision** | **95.8%** (23/24 — the one FP was a test fixture) |
| FN candidates | 6 |
| Rules loaded / fired | 27 / 18 |

## How to Read the FP Number

The 47.1% FP share is real, and it is the honest first-fire number: the scan
ran with **zero tuning** — no config, no suppressions, default everything —
against a codebase Vanguard had never seen. Three things put it in context:

1. **The gate only reacts to ERROR.** The CI-blocking tier was 23 correct
   out of 24. The noise lives in WARN/INFO — advisory tiers that never fail
   a build.
2. **The labeling protocol is FP-first**: when a verdict was uncertain, the
   row was labeled FP, never TP. 115 is an upper bound by construction.
3. **Four heuristic clusters explain 88% of the FPs** (101 of 115), and each
   has a known fix — see the table below.

## The Four FP Clusters

| Cluster | FP | What happens | Fix |
|---|---|---|---|
| Create-shape heuristic (`R2xx-02` + `R5xx-03`) | 38 | 7 spans *did* return 201 (heuristic missed it) + 11 custom-action POSTs (`login`, …) judged as create-shaped | predicate fix + action-verb lexicon |
| Action-on-GET hints (`R2xx-05`, INFO) | 30 | "looks like an action — verify manually" fired on pure-read GETs (`export`, `search`, `get-term`); AIP-136 permits side-effect-free custom GETs | by design a verify-hint; severity/method-name split on the backlog |
| Action-segment naming (`R1xx-01`) | 19 | `/tree`, `/filter`, `/auto-complete`, `/all-by-condition` read as singular collection nouns | action-segment vs collection-noun split |
| Demo-fixture rules (`R6xx-92`/`93`) | 14 | golden-fixture rules fired on ordinary endpoints | disable in config — [Tuning §1](/guides/tuning) |

The engine-side fixes are tracked in the backlog; meanwhile the documented
tuning pass ([Tuning & Suppression](/guides/tuning)) removes the demo noise
immediately and accounts for the rest with reason-carrying suppressions. On
the purpose-built 41-endpoint audit sample — tuned, fixture-backed — the FP
share was **10.5%**, below the 15% quality gate.

## What the ERROR Tier Caught

The 23 true ERROR findings are the contract-breaking class
[R4xx-01 no-entity-in-payload](/rules/R4xx-01-no-entity-in-payload) leaking
ORM entities onto the wire, [R1xx-02 no-verb-path](/rules/R1xx-02-no-verb-path)
CRUD verbs in paths, [R2xx-01 get-no-body](/rules/R2xx-01-get-no-body) GET
with a request body, and [R5xx-02 no-500-for-business](/rules/R5xx-02-no-500-for-business)
business exceptions mapped to HTTP 500. Each is a real integration hazard
for clients, not a style opinion.

## The Remediation Backlog Shape

Triaged findings became a prioritized backlog, which is the practical output
of an audit:

| Priority | Items | Examples |
|---|---|---|
| P1 | 8 entity leaks + 3 missed same-shape spans, ~21 non-standard routes, 1 GET-with-body | `GET /youths/current` returns the ORM entity |
| P2 | 13 create-returning-200, 20 unpaginated lists, 4 single-span fixes | POST → 200 instead of 201 |
| P3 | 4 side-effecting GET/PUT actions, 24 bare arrays, 11 time-typed fields, 3 proto rpc names | `List<T>` with no envelope |

## Reproducibility

Everything on this page regenerates from the tree: clone the repository,
checkout the audited revision, `make bin`, and run
`vanguard scan <target> --format json`. The audit protocol — verdict
vocabulary, reason codes, FP-first tie-break, and the sheet format — is
specified in [FP/FN Audit Protocol](/fpfn-protocol), and the test strategy
that produced the sample corpus in [Test Strategy](/test-strategy).
