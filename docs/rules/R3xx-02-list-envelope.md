# R3xx-02 — list-envelope (INFO)

A collection response should be wrapped in an envelope that carries the
items plus paging metadata (total, page) instead of a bare JSON array
(AIP-158). A bare array leaves no room for metadata and forces a breaking
change when it is needed later.

## Fires when

- the `ir.Method` response has `IsCollection` true with a non-empty type,
  AND the response spelling is a **bare container**: the base type is one
  of `List, ArrayList, LinkedList, Collection, Iterable, Stream, Set,
  HashSet, LinkedHashSet, SortedSet, TreeSet, Queue, Deque`, or the
  spelling ends with `[]` (arrays, varargs).

## Stays silent when

- the container is a recognized envelope: Spring's `Page<T>`/`Slice<T>`
  (they carry total/page metadata by definition) — the rule mirrors the
  adapter's `collectionBases` minus those envelope shapes;
- the response type is anything else (a custom `OrderPageResponse`, a
  generated wrapper). **The rule under-reports rather than guessing**: a
  custom envelope name is never required to look like one.

## Scope notes (attack surface — W4, read this)

- Heuristic by design: the IR has no envelope model (w3-02 keeps
  `Response.Envelope` empty — recognizing an envelope shape is rule
  territory). The bare-container list is closed and visible above.
- Independent of input pagination: `GET /search` with a `size` parameter
  returning `List<T>` still fires — output shape, not input paging.
- INFO severity: a convention note, not a defect. A codebase that chose
  bare arrays everywhere can disable the rule or the family in config.

## Good / Bad

```java
// good — envelope with metadata
@GetMapping
public Page<BookDto> list(Pageable pageable) { ... }

// bad — bare array
@GetMapping
public List<BookDto> list() { ... }
```
