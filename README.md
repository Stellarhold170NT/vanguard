# vanguard

> Source-first API design linter — scans source code directly, auto-detects
> the REST + gRPC API surface, and checks API design against AIP-style
> (resource-oriented design) rules with precise `file:line:col` findings.

[![CI](https://github.com/Stellarhold170NT/vanguard/actions/workflows/ci.yml/badge.svg)](https://github.com/Stellarhold170NT/vanguard/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Status: v0.1 under construction.** The repository was bootstrapped at
checkpoint CH1 together with the charter; see
[docs/charter.md](docs/charter.md) for the working contract.

## Why vanguard

Spec linters (spectral, speccy) only see the OpenAPI document that is
generated after the service builds and runs — one loop too late, and with
locations that no longer map to source. Vanguard reads the source directly:

```bash
vanguard scan /path/to/project   # no build, no spec, no file list
```

Design-standard findings (naming, verb semantics, pagination, payload
schema, error model, versioning, gRPC conventions) with CI-native output:
pretty / JSON / SARIF 2.1.0, exit codes 0/1/2.

## Repository layout

| Path | Responsibility |
|---|---|
| `cmd/vanguard/` | CLI entry point |
| `internal/ir/` | `ApiSurface` IR — pure, language-agnostic (w2-02) |
| `internal/engine/` | rule engine, registry, suppression (w2-04) |
| `internal/render/` | pretty / JSON / SARIF output (w2-05) |
| `internal/discovery/` | tree walker, language detection, adapter registry (w2-03) |
| `adapters/java/` | Java/Spring adapter (w3-01, w3-02) |
| `rules/` | R1xx–R6xx rule metadata + check functions (w3-03..w3-06) |
| `testdata/` | stub-repo, golden, adversarial, mutation corpora |
| `docs/` | charter, test strategy, per-rule documentation |
| `reference/api-linter/` | local study clone of googleapis/api-linter — not committed |

## Documentation

- [docs/charter.md](docs/charter.md) — product charter: requirements R1–R8,
  rule taxonomy R1xx–R6xx, architecture, CLI spec, open decisions.
- [docs/test-strategy.md](docs/test-strategy.md) — five-tier test
  architecture and corpus plan.
- [CONTRIBUTING.md](CONTRIBUTING.md) — development setup and the process for
  adding a rule.

## License

[Apache-2.0](LICENSE) — the same license as
[googleapis/api-linter](https://github.com/googleapis/api-linter), the
spiritual ancestor of the rule model.
