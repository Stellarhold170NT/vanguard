# R1xx-05 — id-field-naming

| | |
|---|---|
| ID | `R1xx-05` |
| Slug | `id-field-naming` |
| Category | `resources` |
| Default severity | **INFO** |
| AIP reference | AIP-122, AIP-140 |
| Family | R1xx — resources & naming |

## Why

Id fields stay consistent: `id` for the self id, `<resource>Id` for
references. v0.1 detects MIXED usage and reports it — it does not enforce
one convention (w1-03 §2.4: 182 DTOs mixing three naming schemes, `youId`
-style fields included). Mixed spellings mean every consumer maps the same
concept through three field names.

## What fires

- A scanned DTO type whose id-suffixed fields mix the spellings —
  `bookID` (Id suffix), `member_id` (_id suffix) and `publisherId`
  (camel Id) in one type read as three conventions.
- The self-id convention is checked against the same spelling set: a type
  whose references say `libraryId` while its own key says `bookID`
  diverges from the majority spelling of the surface.

## What stays silent (FP shield)

- Exact case-sensitive spellings, so words like `android` or `valid`
  never read as id fields (the suffix match is `id`, `_id`, `Id`, `ID`
  with exact casing, not a substring).
- Types with a single id field in any ONE spelling — consistency, not
  spelling, is the v0.1 contract.

## Spring example — compliant

```java
// id for the self id, <resource>Id for references — one convention.
public record BookDto(Long id, Long authorId, Long publisherId) { }
```

## Spring example — violation

```java
// Three spellings of the same convention.
public record BookDto(Long bookID, Long author_id, Long publisherId) { }
```

## Suppression

```java
// vanguard:ignore R1xx-05 <reason>
```

Type-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with
`paths` matching the DTO file, §6.4.1) when a wire type is frozen by a
published contract.

## Scope notes

- The rule reads only the `ir.Type` field list the adapters emit — no
  reflection, no source re-reading.
- INFO severity: the mixed spelling still serializes; the cost is
  consumer confusion, not breakage.
