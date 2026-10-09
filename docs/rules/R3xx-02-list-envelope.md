# R3xx-02 — list-envelope

| | |
|---|---|
| ID | `R3xx-02` |
| Slug | `list-envelope` |
| Category | `pagination` |
| Default severity | **INFO** |
| AIP reference | AIP-158 |
| Family | R3xx — pagination |

## Why

A collection response should be wrapped in an envelope that carries the
items plus paging metadata (total, page) instead of a bare JSON array
(AIP-158). A bare array leaves no room for metadata and forces a breaking
change when it is needed later.

## What fires

- The `ir.Method` response has `IsCollection` true with a non-empty type,
  AND the response spelling is a **bare container**: the base type is one
  of `List, ArrayList, LinkedList, Collection, Iterable, Stream, Set,
  HashSet, LinkedHashSet, SortedSet, TreeSet, Queue, Deque`, or the
  spelling ends with `[]` (arrays, varargs).

## What stays silent (FP shield)

- Recognized envelopes: Spring's `Page<T>` / `Slice<T>` carry total/page
  metadata by definition.
- Any other response spelling (a custom `OrderPageResponse`, a generated
  wrapper) — **the rule under-reports rather than guessing**: a custom
  envelope name is never required to look like one.

## Spring example — compliant

```java
// Spring's paging envelope carries the items AND the metadata.
@GetMapping("/authors")
public Page<AuthorDto> listAuthors(@RequestParam int page, @RequestParam @Max(100) int size) { ... }
```

## Spring example — violation

```java
// A bare JSON array — no room for paging metadata.
@GetMapping("/books")
public List<BookDto> listAllBooks(@RequestParam int page, @RequestParam int size) { ... }
```

## Suppression

```java
// vanguard:ignore R3xx-02 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for endpoints whose bare array is a frozen contract.

## Scope notes

- The envelope set mirrors the adapter's `collectionBases` minus the
  Spring paging shapes; extending it is a data change, not a rule change.
- A custom envelope that merely CONTAINS a list (e.g.
  `MemberSuggestions(List<String>)`) is a non-collection response at the
  IR level and never fires.
