---
title: Guides
description: Operational guides for Vanguard — CI gating, tuning false positives, and a real-fire audit case study.
---

# Guides

Task-oriented guides for running Vanguard in anger. Concept-first material
lives in [Architecture](/architecture/); reference material in
[Reference](/reference/cli).

## Available Guides

- **[CI Integration](/ci-integration)** — gate pull requests with the exit-code
  contract; GitHub Actions, GitLab CI, and shell-runner recipes; SARIF upload
  with stable alert identity.
- **[Tuning & Suppression](/guides/tuning)** — keep signal high on a real
  codebase: severity overrides, path-scoped suppressions, inline ignores, and
  the demo-fixture rules you should disable outside `testdata`.
- **[Case Study: Real-Fire Baseline](/guides/case-study)** — what a first scan
  of a production-shaped 182-endpoint Spring service actually looks like:
  recall, triage, FP clusters, and the remediation backlog shape.

## Quick Pointers

| Task | Start here |
|---|---|
| Fail CI on design regressions | [CI Integration](/ci-integration) |
| A rule is too noisy for my repo | [Tuning & Suppression](/guides/tuning) |
| Generate a config to start from | `vanguard init` — [Configuration](/config) |
| Understand one finding | `vanguard explain <rule-id>` — [Rules](/rules/README) |
| Produce the SARIF artifact for GitHub | [CI Integration §2](/ci-integration) |
