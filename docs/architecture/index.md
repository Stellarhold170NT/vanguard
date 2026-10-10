---
title: Architecture
description: Vanguard architecture overview — the scanning pipeline, the ApiSurface IR, the data-driven rules engine, and the deterministic reporting contract.
---

# Vanguard Architecture

Vanguard is a source-first API design linter. It parses source code directly
(no proto/OpenAPI needed), discovers the REST + gRPC API surface, and checks
it against a catalog of AIP-grounded design rules with precise
`file:line:col` findings.

This document describes the current architecture and the boundaries between
its four stages. The full product charter — positioning, requirements R1–R8,
the complete rule taxonomy, and the CLI UX spec — lives in the
[Charter](/charter).

## Pipeline Overview

```text
source tree
  -> Walker          include/exclude globs, built-in ignores, deterministic order
  -> Adapter layer   language/framework auto-detect, tree-sitter parsing,
                     springdoc overlay
  -> IR (ApiSurface) endpoints, methods, payload types, fields, evidence spans
  -> Rules engine    data-driven catalog (rules/data/*.yaml), severity model,
                     suppression pipeline
  -> Report          pretty (TTY) | JSON | SARIF 2.1.0 — byte-deterministic
```

Every stage feeds the next through the in-memory IR. Nothing in the pipeline
reads the network, runs a build, or shells out to a compiler.

## The Four Stages

1. **Walk & parse** — the walker visits the scan root in deterministic order,
   applies `include`/`exclude` globs and built-in ignores, and hands files to
   the adapter layer. Parsers are embedded tree-sitter grammars (cgo), so a
   malformed file produces a diagnostic — never a crash and never a partial
   abort of the scan.
2. **Discover the surface** — the adapter registry auto-detects the language
   and framework (Java Spring Boot MVC and WebFlux annotations, springdoc
   OpenAPI overlay, `.proto` files for gRPC). Discovery turns parsed
   constructs into endpoint and type records.
3. **Evaluate rules** — the rules engine evaluates the IR against the
   registered catalog. Rules are data-driven: each rule's identity, message,
   severity, default state, and examples live in `rules/data/*.yaml`, and the
   Go implementations register against that metadata.
4. **Report** — findings render as pretty (TTY, color, grouped), JSON, or
   SARIF 2.1.0. Output is byte-deterministic by contract: same tree, same
   bytes (the engine-measured duration is excluded unless `--timing` is set).

## Surfaces at a Glance

| Surface | What it owns | Documentation |
|---|---|---|
| Scanning pipeline | Walker, adapters, tree-sitter grammars, IR | [Pipeline & IR](/architecture/pipeline) |
| Rules engine | Catalog, severity model, exit codes, determinism, suppression order | [Rules Engine](/architecture/rules-engine) |
| CLI | `scan`, `check`, `explain`, `init`, flags, formats | [CLI Reference](/reference/cli) |
| Configuration | `.vanguard.yaml` schema, globs, precedence | [Configuration](/config) |
| Rule catalog | 22 production rules + 5 demo fixtures across R1xx–R6xx | [Rules](/rules/README) |

## Design Principles

### Source First

The API surface is discovered from source, not from a generated spec. Spec
linters (spectral, speccy) only see the OpenAPI document that exists after
the service builds and runs — one loop too late, and with locations that no
longer map to source. Vanguard's findings always carry
`file:line:col` evidence in the code under review.

### Neutral Rule Identity

Rule ids are `R<F>xx-NN` — family number and sequence, no prose in the id.
Messages, severities, and examples live in metadata, so ids stay stable while
wording evolves. Every rule traces to an AIP guideline
([google.aip.dev](https://google.aip.dev/)) restated in Spring terms.

### Data-Driven Catalog

A rule = one YAML metadata file + one Go implementation registered against
it. The 1-1 match between the catalog and `docs/rules/*.md` is enforced by a
test in `make ci` — a rule without documentation fails the build.

### Deterministic Output

Reports are reproducible byte-for-byte (verified 6/6 sha256-identical pairs
on a 10k-file tree, including the SARIF format). Determinism is what makes
golden-snapshot testing, stable GitHub alert identity (`partialFingerprints`),
and diff-friendly CI logs possible.

### Fail-Open Suppression

Suppression directives (inline comments and config path-globs) are
file-scoped and reason-carrying. A malformed directive suppresses nothing and
produces a diagnostic; it never aborts the scan and never changes the exit
code. Suppressed findings are counted and visible in every output format.

## Core Flows

### Scan

```text
vanguard scan <path> [--format pretty|json|sarif] [--config FILE]
  -> walk tree (globs + built-in ignores)
  -> auto-detect language/framework, parse with tree-sitter
  -> build ApiSurface IR
  -> evaluate enabled rules, apply suppressions
  -> render report; exit 0/1/2 by the contract
```

### Gate (CI)

```text
vanguard check <path>
  -> same pipeline as scan
  -> non-TTY stdout: suppress pretty output, print one summary line
  -> exit 1 only on surviving ERROR findings
```

### Explain

```text
vanguard explain <rule-id>
  -> resolve rule metadata (summary, severity, AIP reference)
  -> render why / what fires / what stays silent / compliant example
```

## References

- [Scanning Pipeline & IR](/architecture/pipeline)
- [Rules Engine & Determinism](/architecture/rules-engine)
- [Rule Catalog](/rules/README)
- [Configuration](/config)
- [CLI Reference](/reference/cli)
- [CI Integration](/ci-integration)
- [Charter v1](/charter)
