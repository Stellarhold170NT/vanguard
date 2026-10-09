# R6xx-02 — grpc-standard-methods

| | |
|---|---|
| ID | `R6xx-02` |
| Slug | `grpc-standard-methods` |
| Category | `versioning` |
| Default severity | **INFO** |
| AIP reference | AIP-131..136 |
| Family | R6xx — http-grpc & versioning |

## Why

A proto rpc should carry a **Standard Method** name — `Get`, `List`,
`Create`, `Update`, `Delete` (AIP-131..135), their batch variants, or
AIP-137's `Search` — optionally suffixed with the resource noun
(`GetBook`), or be a **VerbNoun custom method** (AIP-136: `ArchiveBook`,
`SyncData`). An off-pattern rpc name leaks into every generated client and
the `*GrpcService` impl classes built on it (w1-03 §2.3: 11 services /
88 rpcs declared in `src/main/proto`).

## What fires

An `ir.GrpcService` declares an rpc whose name:

1. does **not** start with a Standard Method verb (`Get`/`List`/`Create`/
   `Update`/`Delete`/`BatchGet`/`BatchCreate`/`BatchUpdate`/
   `BatchDelete`/`Search`, followed by end-of-name or a camel noun), and
   is **not** a VerbNoun (first camel word is a known action verb and the
   name has ≥ 2 words), and
2. is not exempted by the `allow` option.

## What stays silent

- Standard-prefixed names (`GetAllBooks` passes — the prefix branch; the
  List-vs-Get semantics drift is a documented miss, not an FP).
- VerbNoun custom methods (`ArchiveBook`, `SyncData`).
- Repos without `.proto` files (a clean skip — the w3-06 acceptance).

## The reason escape hatch

"…mà không có lý do" — the reason lives in the config, next to the name:

```yaml
rules:
  R6xx-02:
    options:
      allow: ["DoMagic"]   # keep a deliberate name explicit and greppable
```

## Example — compliant

```protobuf
// Standard Methods and a VerbNoun custom method
service LibraryService {
  rpc GetBook(GetBookRequest) returns (Book);
  rpc ArchiveBook(ArchiveBookRequest) returns (Book);
}
```

## Example — violation

```protobuf
// neither Standard nor VerbNoun
service LibraryService {
  rpc DoMagic(MagicRequest) returns (MagicResponse);
}
```

## Suppression

```java
// vanguard:ignore R6xx-02 <reason>
```

The proto surface has no Java anchor, so the working suppression is the
`allow` option above (config, greppable) or path-scoped suppression of the
`.proto` file in `.vanguard.yaml` (§6.4.1).

## Scope notes (attack surface — W4, read this)

- **Declaration level only** (the brief caps v0.1): the adapter's
  mini-reader sees `service` + `rpc` names in `.proto` files; message
  bodies, options and field names are out of scope (a real proto parser
  is backlog, charter R6xx-06).
- Weird files are skipped **silently**: unreadable files, unbalanced
  braces, `service` without a name/body contribute nothing and never
  produce a diagnostic or a tool error.
- Commented-out services are invisible (comments are stripped before the
  declaration scan — w1-03: commented code is not an API surface).
- Proto reading rides on the Spring/Java scan (one adapter is selected
  per repo); a proto-only repo without Java is out of scope in v0.1.
- Single-word rpcs (`Handle`, `Foo`) are neither Standard nor VerbNoun —
  they fire.
