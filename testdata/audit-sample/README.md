# testdata/audit-sample — FP/FN audit sample set (w4-04, test-strategy §7)

A 51-operation API surface (41 HTTP method-level endpoints + 10 rpc) that
**reproduces the military-youth patterns w1-03 documented** (pain 1–7 +
§2.1–§2.4) on a fictional youth-union domain (`com.youthunion.audit`,
`/api/youth-union/…`). **No military-youth source is copied** (charter §12):
every file is freshly written, minimal code that exhibits the documented
*pattern* — the same policy the adversarial corpus follows, one tier closer
to the real app.

## Layout

| path | content |
|---|---|
| `.vanguard.yaml` | scan config pass — everything default (R6xx-9x demo off, R6xx-01 policy off) |
| `expectations.json` | pre-registered violation claims + declared coverage gaps (authored BEFORE the first scan — anti-gaming §9.1) |
| `src/…/entity/` | 2 JPA entities (`@Entity`) — R4xx-01's sole-entity signal |
| `src/…/dto/` | payload types: clean baseline, mixed-convention DTO, `PageResponse<T>`, import pair |
| `src/…/service/` | app-owned exceptions (`*Exception` — R5xx-02 signal) |
| `src/…/config/` | `GlobalErrorAdvice` (4 handlers, 1 deviant) + `ResponseWrapper` (2nd advice) |
| `src/…/web/rest/` | 8 controllers / **41 HTTP method-level endpoints** (incl. the `GET /reports/summary` surface-binding added by the w4-04 construct fix) |
| `proto/` | 2 services / **10 rpc** (3 non-standard, 1 commented-out corner) |

## Pattern → construct map (w1-03 traceability)

| w1-03 pattern | sample construct |
|---|---|
| pain 1 verbs in path | `POST /movement-logs/update/draft`, `/update/submit`; `GET /find-by-id/{id}`, `/find-all-id-active`; `DELETE /training-courses/delete` |
| pain 2 path casing | `GET /magazine-posts/search/all-by-Name` |
| pain 3 double delete shapes | `DELETE /{id}` vs `DELETE /training-courses/delete` |
| pain 4 pagination split | `Page<T>` (Youth, Training, UnionFee) vs `PageResponse<T>` (Report) vs bare `List<>` (archived, comments, permissions, /api, by-member) |
| pain 5 dual envelopes | `GlobalErrorAdvice` (4 handlers) + `ResponseWrapper` advice |
| pain 6 entity on the wire | `GET /youths/current` → `YouthMember`; `POST /movement-logs/import` → `ImportResult<YouthMember>` |
| pain 7 HTTP semantics | GET `/auto-complete` + `@RequestBody`; DELETE batch + `@RequestBody List<String>`; PUT `/{id}/status` partial; full-replacement PATCH |
| §2.4 DTO mix | `ReportSummaryDto` (`reportID`, `branch_id`, `created_date` String), `MovementLogDto` (`movementLogId` self-id) |
| §2.1 no version segment | every base path unversioned; bare `/api` route index |
| §2.3 gRPC naming | standard + VerbNoun rpcs vs `FetchReportData` / `RemoveById` / `DoExport` |

## Scan

```sh
./bin/vanguard scan testdata/audit-sample --format json --config testdata/audit-sample/.vanguard.yaml
```

The audit harness (`tools/audit-check`) runs exactly this scan, merges the
findings with `expectations.json`, and emits the labeling sheet — see
`docs/fpfn-protocol.md`.
