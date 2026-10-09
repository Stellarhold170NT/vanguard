# R4xx-01 — no-entity-in-payload (ERROR)

An API payload or response must not expose an ORM entity type (AIP-121,
AIP-203). Exposing an entity leaks the database schema and lazy relations
into the API contract and couples clients to the persistence model
(w1-03 pain 6: `GET /current` returned the `Youth` entity directly).

This is the only ERROR-severity rule of the w3-05 set: the signal is the
scan's own entity index, not a naming guess.

## The heuristic (attack surface — W4, read this)

1. The rule builds an index over `ir.ApiSurface.Types` — the types the
   ADAPTER emitted (w3-02 sets `Type.IsEntity` for `@Entity`/`@Document`
   classes). A type is an **unambiguous entity** when exactly one scanned
   type carries its simple name and that type `IsEntity`.
2. A method violates the rule when its `Payload` TypeRef or its
   `Response.Type` — at top level OR inside a generic argument
   (`Page<Youth>`, `ImportResult<Youth>`) — names an unambiguous entity.
3. **Package disambiguation (the FP guard)**: for a top-level reference
   the adapter resolved (`TypeRef.Package != ""`), the packages must
   match. An entity `com.example.domain.Youth` and a DTO
   `com.example.web.Youth` are different types — a reference the adapter
   resolved to the DTO's package NEVER fires.
4. **Shadowed names never fire**: when several scanned types share the
   simple name, the adapter's first-declaration-wins resolution makes the
   reference ambiguous; the rule stays silent instead of guessing.
5. An unresolved reference (`Package == ""`) to a lone in-scope entity
   fires — the only candidate the scan knows is that entity.
6. Types outside the scan scope are invisible: an entity from a private
   jar that the scan never saw cannot be flagged. Heuristic signal 2 of
   the charter (package-pattern config) is deliberately NOT implemented
   in v0.1 — a name/package guess would produce the FP wave charter B-2
   warns about.

## Known IR gap (reported to the orchestrator)

The w3-02 adapter keeps ONE `ir.Type` per simple name (first declaration
wins). An entity and a DTO that literally share a simple name cannot both
be in scope, so case 3's protection depends on which file sorts first.
The shadowed-name branch (case 4) is pinned at the unit level; the
end-to-end corpus uses distinct names. Fixing this needs an adapter
change (out of this task's scope by design).

## Good / Bad

```java
// good — DTO on the wire
@GetMapping("/current")
public YouthResponse current(@PathVariable Long id) { ... }

// bad — entity on the wire
@GetMapping("/current")
public Youth current(@PathVariable Long id) { ... }
```
