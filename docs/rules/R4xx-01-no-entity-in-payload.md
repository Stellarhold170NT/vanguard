# R4xx-01 — no-entity-in-payload

| | |
|---|---|
| ID | `R4xx-01` |
| Slug | `no-entity-in-payload` |
| Category | `payload` |
| Default severity | **ERROR** |
| AIP reference | AIP-121, AIP-203 |
| Family | R4xx — payload |

## Why

An API payload or response must not expose an ORM entity type (AIP-121,
AIP-203). Exposing an entity leaks the database schema and lazy relations
into the API contract and couples clients to the persistence model
(w1-03 pain 6: `GET /current` returned the `Youth` entity directly).

This is the only ERROR-severity rule of the R4xx family: the signal is the
scan's own entity index, not a naming guess.

## What fires

1. The rule builds an index over `ir.ApiSurface.Types` — the types the
   ADAPTER emitted (`@Entity`/`@Document` classes set `Type.IsEntity`).
   A type is an **unambiguous entity** when exactly one scanned type
   carries its simple name and that type `IsEntity`.
2. A method violates the rule when its `Payload` TypeRef or its
   `Response.Type` — at top level OR inside a generic argument
   (`Page<Youth>`, `ImportResult<Youth>`) — names an unambiguous entity.

## What stays silent (FP shield)

- **Package disambiguation**: for a resolved top-level reference
  (`TypeRef.Package != ""`), the packages must match — an entity
  `com.example.domain.Youth` and a DTO `com.example.web.Youth` are
  different types.
- **Shadowed names never fire**: several scanned types sharing the simple
  name make the reference ambiguous; the rule stays silent instead of
  guessing.
- Types outside the scan scope are invisible (an entity in a private jar
  the scan never saw cannot be flagged).

## Spring example — compliant

```java
// The entity exists; the wire carries the DTO.
@PostMapping("/books")
public BookDto createBook(@RequestBody BookDto dto) { ... }
```

## Spring example — violation

```java
// The database schema and lazy relations reach the wire.
@PostMapping("/books/import")
public Book importBook(@RequestBody Book entity) { ... }
```

## Suppression

```java
// vanguard:ignore R4xx-01 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1).

## Scope notes

- Known IR gap (w3-05 report): the w3-02 adapter keeps ONE `ir.Type` per
  simple name (first declaration wins), so an entity and a DTO that share
  a simple name cannot both be in scope; the shadowed-name branch is
  pinned at unit level.
- Package-pattern entity guessing (charter heuristic 2) is deliberately
  NOT implemented in v0.1 — a name guess would produce the FP wave
  charter B-2 warns about.
