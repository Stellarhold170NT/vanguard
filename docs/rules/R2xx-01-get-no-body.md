# R2xx-01 — get-no-body

| | |
|---|---|
| ID | `R2xx-01` |
| Slug | `get-no-body` |
| Category | `methods` |
| Default severity | **ERROR** |
| AIP reference | AIP-131, AIP-133 |
| Family | R2xx — methods & verbs |

## Why

GET endpoints must not declare a request body — many HTTP clients silently
drop GET bodies, so the endpoint loses its declared input. This is the
family's one "measured" rule: the handler itself declares input on the one
verb whose bodies popular clients drop, so the finding is a broken
contract, not a style note.

## What fires

The rule fires when the IR shows **Verb == GET** and the method declares a
request body — either view of the w2-02 invariant (`Payload` non-nil, or
an `in=body` Param; the stub adapter may set either half alone, the spring
adapter always fills both).

## What stays silent (FP shield)

- gRPC operations (`Verb: rpc`) never match — HTTP verb semantics do not
  apply to them.
- GET with query parameters (`in=query`) is the compliant shape.

## Spring example — compliant

```java
// The query travels in the URL, where every client can send it.
@GetMapping("/members/auto-complete")
public MemberSuggestions autoComplete(@RequestParam String term) { ... }
```

## Spring example — violation

```java
// A declared body on GET — silently dropped by popular HTTP clients.
@GetMapping("/members/auto-complete")
public MemberSuggestions autoComplete(@RequestBody MemberQuery query) { ... }
```

## Suppression

```java
// vanguard:ignore R2xx-01 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1). A stack that genuinely requires GET bodies (e.g.
Elasticsearch-style DSL over GET) suppresses per path — the endpoint still
works, the transport is the risk.

## Scope notes

- None known for v0.1: adapters derive Verb from the mapping annotation
  and the body from `@RequestBody`; a false finding means an adapter bug,
  which the w3-01/w3-02 adapter tests pin.
- The demo rule R6xx-93 carries the same shape at INFO for the demo
  family; this rule is the production-graded one.
