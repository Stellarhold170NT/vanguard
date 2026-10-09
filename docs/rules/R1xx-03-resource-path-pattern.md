# R1xx-03 — resource-path-pattern

| | |
|---|---|
| ID | `R1xx-03` |
| Slug | `resource-path-pattern` |
| Category | `resources` |
| Default severity | **WARN** |
| AIP reference | AIP-127, AIP-131 |
| Family | R1xx — resources & naming |

## Why

Resource paths follow `/collection/{id}/sub-collection/{subId}`: every path
variable sits under the resource that owns it. Flat action-shaped routes
(`find-by-id/{id}`) hide the hierarchy and cannot grow sub-resources
(w1-03 pain 1: `/find-by-id/{id}`, `/find-all-id-active`).

## What fires

One finding per misplaced variable, on the same `ir.Method`:

1. the path **starts** with a variable — `/{id}/profile` has no owner;
2. two variables are **consecutive** — `/{bookId}/{reviewId}` skips the
   sub-collection that owns `{reviewId}`;
3. a variable sits **under an action segment** — `/find-by-id/{id}` hides
   the collection; serve the item as `/<collection>/{id}`.

## What stays silent (FP shield)

- Proper nesting: `/authors/{authorId}/books/{bookId}` (each variable
  follows its owning noun).
- Wildcard segments (`*`, `**`) — no owner to name.

## Spring example — compliant

```java
// The variable sits under the collection that owns it.
@GetMapping("/authors/{authorId}/books/{bookId}")
public BookDto get(@PathVariable Long authorId, @PathVariable Long bookId) { ... }
```

## Spring example — violation

```java
// A flat action route instead of the hierarchy.
@GetMapping("/find-by-id/{id}")
public BookDto get(@PathVariable Long id) { ... }
```

## Suppression

```java
// vanguard:ignore R1xx-03 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1).

## Scope notes

- The rule judges only where variables sit; what the segment says (verb
  vs resource) is R1xx-02's subject — the two fire together on
  `/find-by-id/{id}` on purpose (two aspects, one defect).
- The suggestion is a shape template (`/<collection>/{id}`); the concrete
  collection name comes from the code, so the suggestion stays a pattern.
