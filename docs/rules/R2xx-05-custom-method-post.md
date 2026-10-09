# R2xx-05 custom-method-post

`category: methods · severity: INFO · AIP-136` — charter §3.2 (w3-04).

An action (non-CRUD) endpoint reads best as an AIP-136 custom method —
`POST <collection>/{id}:<action>` (e.g. `POST /books/{id}:archive`).

## Heuristic (convention — documented lexicon)

Fires when ALL of:

1. the method is an HTTP method whose verb is **not POST** (a POST endpoint
   already satisfies the verb half of the convention; reshaping its path
   overlaps the R1xx family, so POST stays silent);
2. the **method-level path** carries at least one literal (non-template)
   segment that tokenizes to an action-verb token. The service base path is
   removed first (`methodRelPath`): one `@RequestMapping("/v1/search")`
   decision must not make every endpoint under it read as an action — the
   action shape is the method mapping's own segments.

**The lexicon** (shared with R2xx-02, `rules/methods.go actionVerbs`):
activate, assign, approve, calculate, cancel, cleanup, clone, compute,
convert, deactivate, disable, download, duplicate, enable, execute, export,
fetch, find, generate, get, grant, import, invoke, list, lock, lookup,
migrate, process, publish, purge, query, refresh, reject, renew, reset,
restore, retry, revoke, run, scan, search, send, submit, sync, translate,
trigger, unassign, uninstall, unlock, unpublish, update, upload, validate,
verify. Tokens are **exact** after splitting each segment on separators
and camelCase boundaries (kebab `generate-code` → `generate` + `code`;
camel `generateCode` → same) — no stemming, so plural nouns (`books`,
`runs`, `playlists`, `scans`) never match.

Deliberately ABSENT (noun/verb ambiguous in REST paths): archive, batch,
bulk, compact, copy, draft, take, write-off. Extend the lexicon only with a
fixture-backed case and a matching fixture row.

Because the lexicon is a heuristic, the message ends with "verify
manually". Known documented miss: `/auto-complete` tokenizes to
`auto` + `complete`, neither in the lexicon (add the compound if a
fixture demands it).

## Suggestion (AIP pattern)

The suggestion is derived from the real path — the action segment becomes a
`:action` suffix on the preceding `{id}` segment (or the collection root):

| Path | Suggestion |
|---|---|
| `/api/v1/books/{id}/generate-code` | `POST /api/v1/books/{id}:generate-code` |
| `/books/{id}/restore` | `POST /books/{id}:restore` |
| `/v1/reports/export` | `POST /v1/reports:export` |
| `/export` (root-level, collection unknown) | `POST <collection>:export` |

## False-positive surface

Singular noun collections that collide with a lexicon verb (e.g. a
`/scan` resource read with GET) — INFO only, "verify manually" in the
message; suppress per path (§6.4.1).
