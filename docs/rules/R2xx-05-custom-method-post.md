# R2xx-05 — custom-method-post

| | |
|---|---|
| ID | `R2xx-05` |
| Slug | `custom-method-post` |
| Category | `methods` |
| Default severity | **INFO** |
| AIP reference | AIP-136 |
| Family | R2xx — methods & verbs |

## Why

An action that does not map to a CRUD verb belongs on a custom method:
`POST /books/{id}:generate-code` (AIP-136). An action token on a non-POST
verb leaks command semantics into a noun route — the operation name leaks
into every client SDK.

## What fires

ALL of these hold on the same `ir.Method`:

1. the method's verb is **not POST** (a POST endpoint already satisfies
   the verb half of the convention; reshaping its path overlaps the R1xx
   family);
2. the **method-level path** (service base path removed by
   `methodRelPath` — one `@RequestMapping("/v1/search")` decision must
   not make every endpoint under it read as an action) carries a literal
   segment that tokenizes to an action-verb token.

**The lexicon** (shared with R2xx-02, `rules/methods.go actionVerbs`):
activate, assign, approve, calculate, cancel, cleanup, clone, compute,
convert, deactivate, disable, download, duplicate, enable, execute,
export, fetch, find, generate, get, grant, import, invoke, list, lock,
lookup, migrate, process, publish, purge, query, refresh, reject, renew,
reset, restore, retry, revoke, run, scan, search, send, submit, sync,
translate, trigger, unassign, uninstall, unlock, unpublish, update,
upload, validate, verify. Tokens are **exact** after splitting each
segment on separators and camelCase boundaries — no stemming, so plural
nouns (`books`, `runs`, `playlists`, `scans`) never match. Deliberately
ABSENT (noun/verb ambiguous in REST paths): archive, batch, bulk, compact,
copy, draft, take. Known documented miss: `/auto-complete` → `auto` +
`complete`, neither in the lexicon.

## What stays silent (FP shield)

- POST endpoints with action tokens (already the AIP-136 shape).
- Action tokens inside the service base path (stripped before judging).
- Substring matches (`preview`, `updates`) — token-boundary only.

## Spring example — compliant

```java
// The custom method is a POST — AIP-136 satisfied.
@PostMapping("/books/{bookId}:generate-code")
public CodeDto generateCode(@PathVariable Long bookId) { ... }
```

## Spring example — violation

```java
// An action path on GET — the verb and the route disagree.
@GetMapping("/members/export")
public List<MemberDto> exportMembers() { ... }
```

## Suppression

```java
// vanguard:ignore R2xx-05 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1).

## Scope notes

- The suggestion is derived from the real path — the action segment
  becomes a `:action` suffix on the preceding `{id}` segment (or the
  collection root): `GET /members/export` → `POST /members:export`.
- Because the lexicon is a heuristic, the message ends with "verify
  manually".
