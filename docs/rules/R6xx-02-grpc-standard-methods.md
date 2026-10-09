# R6xx-02 — grpc-standard-methods (INFO)

A proto rpc should carry a **Standard Method** name — `Get`, `List`,
`Create`, `Update`, `Delete` (AIP-131..135), their batch variants, or
AIP-137's `Search` — optionally suffixed with the resource noun
(`GetBook`), or be a **VerbNoun custom method** (AIP-136: `ArchiveBook`,
`SyncData`). An off-pattern rpc name leaks into every generated client and
the 10 `*GrpcService` impl classes built on it (w1-03 §2.3: 11 services /
88 rpcs declared in `src/main/proto`).

## Fires when

An `ir.GrpcService` declares an rpc whose name:

1. does **not** start with a Standard Method verb (`Get`/`List`/`Create`/
   `Update`/`Delete`/`BatchGet`/`BatchCreate`/`BatchUpdate`/
   `BatchDelete`/`Search`, followed by end-of-name or a camel noun), and
   is **not** a VerbNoun (first camel word is a known action verb and the
   name has ≥ 2 words), and
2. is not exempted by the `allow` option.

## The reason escape hatch

"…mà không có lý do" — the reason lives in the config, next to the name:

```yaml
rules:
  R6xx-02:
    options:
      allow: ["DoMagic"]   # keep a deliberate name explicit and greppable
```

## Suggestion

`rename the rpc to a Standard Method form or a <Verb><Noun> custom method
(AIP-136), or allow-list the name in the R6xx-02 config with a reason`

## Scope notes (attack surface — W4, read this)

- **Declaration level only** (the brief caps v0.1): the adapter's
  mini-reader sees `service` + `rpc` names in `.proto` files; message
  bodies, options and field names are out of scope (a real proto parser
  is backlog, charter R6xx-06).
- Weird files are skipped **silently**: unreadable files, unbalanced
  braces, `service` without a name/body contribute nothing and never
  produce a diagnostic or a tool error — the no-proto repo is a clean
  skip (the w3-06 acceptance).
- Commented-out services are invisible (comments are stripped before the
  declaration scan — w1-03: commented code is not an API surface).
- Proto reading rides on the Spring/Java scan (one adapter is selected
  per repo); a proto-only repo without Java is out of scope in v0.1.
- Standard-**prefixed** names like `GetAllBooks` pass (the prefix branch);
  their List-vs-Get semantics drift is a documented miss, not an FP.
- Single-word rpcs (`Handle`, `Foo`) are neither Standard nor VerbNoun —
  they fire.

## Good / Bad

```protobuf
// good — Standard Methods and a VerbNoun custom method
service LibraryService {
  rpc GetBook(GetBookRequest) returns (Book);
  rpc ArchiveBook(ArchiveBookRequest) returns (Book);
}

// bad — neither Standard nor VerbNoun
service LibraryService {
  rpc DoStuff(DoStuffRequest) returns (Book);
}
```
