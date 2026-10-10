---
title: Contributing
description: How to build, test, and contribute to Vanguard — repo layout, the make ci pipeline, and how to add a rule to the catalog.
---

# Contributing

Vanguard is Apache-2.0 licensed and developed test-first: the quality gates
that produced v0.1 (adversarial corpus, mutation harness, FP/FN audit,
robustness suite) run in `make ci` and are part of everyday contribution.

## Prerequisites

- **Go ≥ 1.22**
- **A C toolchain** (`cc`/`gcc`/`clang`) — the tree-sitter grammar is cgo;
  `CGO_ENABLED=0` builds are not supported.
- `make` (or run the targets individually)

## Build and Test

```bash
make bin          # build ./bin/vanguard
make test         # go test ./...
make ci           # build → tests → golden snapshots → go vet → gofmt check
```

Golden snapshots pin the exact report bytes per fixture; if a change is
meant to alter output, regenerate and commit the snapshots in the same PR —
the review sees the diff of user-visible behavior.

## Repository Layout

| Path | What lives there |
|---|---|
| `cmd/vanguard/` | CLI entrypoint |
| `internal/cli/` | command wiring, output formats, `rulesdocs_test.go` |
| `internal/engine/` | scan pipeline, config, suppression, exit codes |
| `rules/data/` | one YAML metadata file per rule |
| `docs/rules/` | one documentation page per rule (1-1 with the catalog, enforced) |
| `testdata/` | adversarial corpus, audit sample, perf + robustness fixtures |
| `docs/` | this documentation site (VitePress) |

## Adding a Rule

1. **Charter check** — the rule needs a home in the
   [taxonomy](/charter) (`R<F>xx-NN`, one family per concern) and an AIP
   reference. New ids are allocated in the charter's backlog section first.
2. **Metadata** — add `rules/data/<ID>.yaml`: `id`, `slug`, `category`,
   `severity`, `summary`, `docPath`, `exampleGood`, `exampleBad`.
3. **Implementation** — register the Go implementation against the metadata;
   findings carry `file:line:col` evidence and a deterministic suggestion.
4. **Documentation** — add `docs/rules/<ID>-<slug>.md` following the
   existing page shape (Why / What fires / What stays silent / Suggestion /
   Spring examples). `rulesdocs_test.go` fails the build if catalog and pages
   diverge in either direction.
5. **Fixtures** — add an adversarial case under `testdata/adversarial/<ID>/`
   and a golden snapshot; both must pass in `make ci`.

## Pull Requests

- One rule per PR keeps review and golden diffs readable.
- Regenerate goldens deliberately and say so in the description.
- Every claim in a docs change should trace to code or an artifact — the
  docs site is part of the repo, not a separate project
  (see `docs/README.md`).
- CI must be green: tests, goldens, `go vet`, `gofmt`.

## Versioning and Releases

See [Versioning & Releases](/community/versioning) — SemVer, Keep a
Changelog, and the tag-triggered release pipeline.
