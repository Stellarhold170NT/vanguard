# Changelog

All notable changes to vanguard are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-10

First tagged release of **vanguard** — a source-first API design linter for
Java/Spring Boot: it scans your source code directly (no proto/OpenAPI
needed), discovers the HTTP API surface, and flags design drift at review
time, before a spec even exists.

Scope is exactly the v0.1 charter (docs/charter.md §1–§2); charter §1.4
exclusions (autofix `--fix`, differential `--new-from-*`, full
type-resolution, LLM in the lint path, web UI) and the Go/Python adapters
(charter R3, planned wave 2) are **not** in this release.

### Added

- **Scanner engine**: `vanguard scan <path>` auto-detects language and
  framework, builds the API surface IR from source. Java Spring Boot
  adapter (MVC + WebFlux annotations + springdoc overlay) on an embedded
  tree-sitter grammar (charter R1, R3 — v0.1 language matrix).
- **Rule catalog**: 27 active rules across 6 families `R1xx`–`R6xx`
  (resource naming, methods & HTTP semantics, payload, errors & status,
  versioning, surface hygiene) with neutral `R<F>xx-NN` ids (charter R2, R8).
- **CLI UX**: pretty output (TTY color, grouped by rule/severity, fix
  hints), JSON, and SARIF 2.1.0 (schema-validated); `explain <rule-id>`,
  `check`, `init`; exit-code contract `0` clean / `1` ERROR findings /
  `2` tool error (charter R5).
- **Configuration**: `.vanguard.yaml` (enable/disable rules, include/exclude
  paths, severity overrides, framework detection hints) plus inline
  `// vanguard:ignore <rule> <reason>` suppression with 3-tier precedence
  (charter R6).
- **Packaging (R4)**: multi-platform static binaries via goreleaser —
  linux/darwin/windows × amd64/arm64 cross-compiled through a pinned
  `zig cc` toolchain (cgo tree-sitter), tag-stamped version
  (`vanguard version`), sha256 `checksums.txt`, tag-triggered release
  workflow (`.github/workflows/release.yml`).
- **Docker**: scratch image (~9 MB, one static binary, no shell), compose
  demo (`docker compose run --rm scan`); three install paths documented
  and verified in docs/install.md (GitHub Releases + checksum verify,
  Docker, `go install` pinned commit with Go ≥ 1.22 + C toolchain).

### Quality evidence (v0.1 gates)

- 171-case adversarial corpus + mutation harness: declared catch-rate
  100 % (56/56), strict 85.7 % with pre-declared expected-misses (w4-02).
- Real-fire baseline on an 182-endpoint production-shaped service:
  recall 100 % vs the swagger inventory; 244 findings triaged 129 TP /
  115 FP with a remediation backlog (w4-04, w5-04, w5-05).
- Determinism: 6/6 output pairs byte-identical; p95 39.5 s @ 10k files;
  cold start ≤ 47 ms (w4-03).
- Robustness: hostile-input suite (malformed configs, traversal, SARIF
  re-validation 34/34) (w4-05).

[0.1.0]: https://github.com/vanguard-lint/vanguard/releases/tag/v0.1.0
