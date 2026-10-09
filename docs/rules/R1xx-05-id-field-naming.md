# R1xx-05 — id-field-naming

| | |
|---|---|
| ID | `R1xx-05` |
| Slug | `id-field-naming` |
| Category | resources |
| Default severity | **INFO** |
| Inspired by | AIP-122, AIP-140 |
| Docs stub | w3-03 — full page ships with w3-07 |

## Why

Id fields stay consistent: `id` for the self id, `<resource>Id` for
references. v0.1 detects MIXED usage and reports it — it does not enforce
one convention (w1-03 §2.4: 182 DTOs mixing three naming schemes, `youId`
-style fields included).

## Good / Bad

```java
// Good — id for self, <resource>Id for references
record BookDto(Long id, Long authorId, Long publisherId)

// Bad — three spellings of the same convention
record StockItemDto(Long bookID, Long member_id, Long publisherId)
```

## What fires

- One type whose reference ids mix suffix spellings — `bookId` next to
  `bookID` or `member_id`; JSON keys count (`user_id` vs `orderId`).
- A type that names its own id after the resource (`orderId` inside
  `OrderDto`) while other types on the surface use `id`.

## What stays silent (FP shield)

- The charter convention itself: `id` + `<resource>Id` references.
- One consistent reference spelling, even when it is not `<resource>Id`.
- A single reference id — nothing to be inconsistent with.
- Words that merely end in "id": the suffix must be an exact spelling
  (`Id`, `ID`, `_id`), so `android`/`valid` never read as id fields.

## Suppress

```java
// vanguard:ignore R1xx-05 <reason>
```
