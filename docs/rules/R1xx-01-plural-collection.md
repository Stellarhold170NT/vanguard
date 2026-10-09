# R1xx-01 — plural-collection

| | |
|---|---|
| ID | `R1xx-01` |
| Slug | `plural-collection` |
| Category | resources |
| Default severity | **WARN** |
| Inspired by | AIP-131, AIP-122 |
| Docs stub | w3-03 — full page ships with w3-07 |

## Why

A collection path names many items: `/book` reads as one item, `/books` as
the collection. Mixed singular/plural collection roots fork the URL space
for clients (w1-03 §2.1: 23/24 controllers already follow the plural
convention — one lone singular controller creates a second URL scheme).

## Good / Bad

```java
// Good — collection noun in the plural
@GetMapping("/books/{id}")
public BookDto get(@PathVariable Long id)

// Bad — singular collection segment
@GetMapping("/book/{id}")
public BookDto get(@PathVariable Long id)
```

## What fires

- Any path segment directly followed by a path variable (`/book/{id}`).
- The final segment of a GET whose response is a collection
  (`/training-plan` returning `Page<TrainingPlanDto>`).

## What stays silent (FP shield)

- Uncountable and irregular nouns (`/equipment`, `/people`).
- s-ending singulars are flagged, but the blocklist keeps `status`,
  `address`, `process`, … singular on purpose.
- Action/aggregate sub-paths (`/books/search`, `/loans/stats`) — custom
  methods are R2xx-05's subject, not plural nouns.
- Segments containing a CRUD verb (`/find-by-id/{id}`) — R1xx-02/R1xx-03 own
  those.

## Suppress

```java
// vanguard:ignore R1xx-01 <reason>
```
