---
title: Scanning Pipeline & IR
description: How Vanguard walks the source tree, detects frameworks, parses with tree-sitter, and materializes the ApiSurface IR.
---

# Scanning Pipeline & IR

This page describes stages 1–2 of the pipeline: the walker, the adapter
layer, and the `ApiSurface` IR that everything downstream consumes.

## 1. Walker

The walker visits the scan root in deterministic order and applies, in
sequence:

- **Built-in ignores** — hidden files (dot-prefixed, e.g. `.vanguard.yaml`,
  `.git`) and dependency/vendor directories are skipped by discovery.
- **`include` globs** — default `["**"]`.
- **`exclude` globs** — applied after the built-in ignores; defaults add
  `**/generated/**` and `vendor/**`.

All globs are relative to the directory that contains the effective
`.vanguard.yaml` and use `/` separators on every platform. The glob dialect
(`**` spans whole segments only; malformed patterns fail the config at load
time, exit 2) is specified in [Configuration §2.2](/config).

The walker never follows symlinks out of the scan root and never mutates the
tree — scans are read-only over the target.

## 2. Adapter Layer

The adapter registry maps detected (language, framework) pairs to parsers.
Discovery is automatic; a config `frameworks:` block can pin adapters
explicitly.

### 2.1 Java — Spring Boot (v0.1 language matrix)

The Java adapter is an embedded tree-sitter grammar (cgo — a C toolchain is
required to build, not to run the release binaries). It recognizes:

- **Spring MVC annotations** — `@RestController`/`@Controller`,
  `@RequestMapping` and the composed shortcuts (`@GetMapping`,
  `@PostMapping`, `@PutMapping`, `@PatchMapping`, `@DeleteMapping`), path
  variables, request params, request bodies, response statuses.
- **WebFlux** — the same annotation surface on reactive handlers.
- **springdoc overlay** — `@Operation`, `@ApiResponse`, and schema
  annotations refine the endpoint records where present.

### 2.2 Protocol Buffers — gRPC

`.proto` files are parsed for `service`/`rpc` declarations. gRPC surface
records feed the R6xx family (for example
[R6xx-02 grpc-standard-methods](/rules/R6xx-02-grpc-standard-methods)).

### 2.3 Malformed Input

A file that fails to parse produces a diagnostic. Diagnostics never abort the
scan, never become findings, and never change the exit code — the rest of the
tree is still scanned (charter §5.3 fail-open posture).

## 3. The ApiSurface IR

The IR is the single contract between the adapter layer and the rules engine.
Its main record types:

| Record | Carries |
|---|---|
| **Endpoint** | HTTP verb, path segments, handler method name, declared status codes, request/response type references, evidence span (`file:line:col`) |
| **Type** | Payload DTOs, entities (`IsEntity` flag), envelopes; name + package identity |
| **Field** | Name, effective JSON name (`@JsonProperty`/`@SerializedName` resolved), declared Go-of-Java type (e.g. time-typed fields), entity back-reference |
| **Rpc** | Service name, rpc name, streaming mode, request/response message references |

Two properties matter for rule authors:

- **Effective JSON name** — payload rules evaluate the wire name, not the
  Java name: an explicit `@JsonProperty("created_by")` is what
  [R4xx-02](/rules/R4xx-02-field-casing) sees.
- **Surface binding** — audit rules evaluate *surface-bound* types. A DTO
  referenced by no endpoint stays silent; bind it to an endpoint to make it
  visible to the engine (this is a deliberate anti-noise choice, learned
  during the FP/FN audit — see [Test Strategy](/test-strategy)).

## 4. Determinism

The pipeline is deterministic end to end: file visit order, adapter
selection, IR serialization, and rule evaluation order are all stable. Two
scans of the same tree produce byte-identical reports in every format
(verified in the W4 determinism battery — 6/6 sha256-identical pairs on a
10k-file / 353k-LOC tree, cold start ≤ 50 ms, p95 39.5 s).

## References

- [Rules Engine & Determinism](/architecture/rules-engine)
- [Configuration — glob dialect](/config)
- [Charter §5 — architecture](/charter)
