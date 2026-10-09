# java-spring-clean — the clean rewrite (w3-07)

The same library service as [`../java-spring/`](../java-spring/), written
to the AIP conventions: **exit 0, zero findings in all three formats**
while all 27 registered rules are active — `R6xx-01` is opted in via
`.vanguard.yaml` and every service base carries a version segment.

This case is the false-positive guard for the whole W3 rule set on a rich
Spring surface (controllers, DTOs, one entity, one proto, one advice
class). Per-rule clean counterparts are mapped in
[`../java-spring/README.md`](../java-spring/README.md) (section "Rule →
clean counterpart").

## Layout

| Path | Role |
|---|---|
| `book/` | paginated + capped + enveloped list, item route, 201 create, same-type PUT |
| `member/` | query-param auto-complete, Pageable list, dedicated PATCH type, body-less DELETE |
| `author/`, `loan/` | 201 creations, `Page<…>` returns, `@Max` caps, plural collections |
| `libraryinfo/` | typed response on a versioned base |
| `domain/Book.java` | an entity that exists but never reaches the wire (`R4xx-01` clean) |
| `err/` | one shared `ErrorResponse` envelope; business exception → 409 Conflict |
| `src/main/proto/library.proto` | Standard Methods + VerbNoun only |
