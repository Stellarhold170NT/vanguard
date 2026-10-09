# R1xx-03 — resource-path-pattern

| | |
|---|---|
| ID | `R1xx-03` |
| Slug | `resource-path-pattern` |
| Category | resources |
| Default severity | **WARN** |
| Inspired by | AIP-127, AIP-131 |
| Docs stub | w3-03 — full page ships with w3-07 |

## Why

Resource paths follow `/collection/{id}/sub-collection/{subId}`: every path
variable sits under the resource that owns it. Flat action-shaped routes
(`find-by-id/{id}`) hide the hierarchy and cannot grow sub-resources
(w1-03 pain 1: `/find-by-id/{id}`, `/find-all-id-active`).

## Good / Bad

```java
// Good — the variable sits under the collection that owns it
@GetMapping("/authors/{authorId}/books/{bookId}")

// Bad — a flat action route instead of the hierarchy
@GetMapping("/find-by-id/{id}")
public BookDto get(@PathVariable Long id)
```

## What fires

- A path variable with no predecessor segment (`/{memberId}/loans`).
- Two consecutive variables — the sub-collection owning the second id is
  missing (`/orders/{orderId}/{itemId}`).
- A variable under an action segment — `find-by-id/{id}` is not a resource.

## What stays silent (FP shield)

- Legal shapes: `/libraries/{libraryId}`, `/books/{bookId}/reviews/{reviewId}`.
- Paths without variables (`/health`) — action verbs there belong to
  R1xx-02 (verbs) and R2xx-05 (custom methods).

## Suppress

```java
// vanguard:ignore R1xx-03 <reason>
```
