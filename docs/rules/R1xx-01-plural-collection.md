# R1xx-01 — plural-collection

| | |
|---|---|
| ID | `R1xx-01` |
| Slug | `plural-collection` |
| Category | `resources` |
| Default severity | **WARN** |
| AIP reference | AIP-131, AIP-122 |
| Family | R1xx — resources & naming |

## Why

A collection path names many items: `/book` reads as one item, `/books` as
the collection. Mixed singular/plural collection roots fork the URL space
for clients (w1-03 §2.1: 23/24 controllers already follow the plural
convention — one lone singular controller creates a second URL scheme).

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

## Spring example — compliant

```java
@RestController
@RequestMapping("/api/v1/books")
public class BookController {

    // /books names the collection; the response is a collection too.
    @GetMapping
    public Page<BookDto> listBooks(Pageable pageable) { ... }
}
```

## Spring example — violation

```java
@RestController
@RequestMapping("/api/v1/loan")     // singular collection root
public class LoanController {

    @GetMapping                     // GET /api/v1/loan — one URL scheme forked
    public List<LoanDto> listLoans() { ... }
}
```

## Suppression

```java
// vanguard:ignore R1xx-01 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) when the segment is a legacy contract you cannot rename.

## Scope notes

- The plural check is token-based (a curated irregular/uncountable
  list + s-suffix heuristics), never a dictionary lookup — extend the
  list, not the rule.
- Only the path is judged; the response type only matters for the
  final-segment GET branch.
