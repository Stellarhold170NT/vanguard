# Rule reference — vanguard catalog

One page per registered rule, generated-by-hand from the same metadata the
binary prints with `vanguard scan --list-rules`. The 1-1 match between the
catalog and this directory is enforced by `internal/cli/rulesdocs_test.go`
(runs in `make ci` / `make docs-check`): a rule that lands without its
page fails the build, and an orphan page fails it too.

Suppression syntax on every surface:

```java
// vanguard:ignore <RULE-ID> <reason>
```

or path-scoped in `.vanguard.yaml` (`suppressions:` with a `reason`,
docs/config.md §6.4.1).

## R1xx — resources & naming

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R1xx-01](R1xx-01-plural-collection.md) | plural-collection | WARN | `GET /book` → `GET /books` |
| [R1xx-02](R1xx-02-no-verb-path.md) | no-verb-path | ERROR | `GET /find-by-id/{id}` → `GET /books/{id}` |
| [R1xx-03](R1xx-03-resource-path-pattern.md) | resource-path-pattern | WARN | `/find-by-id/{id}` → `/<collection>/{id}` |
| [R1xx-04](R1xx-04-path-casing.md) | path-casing | WARN | `{BookId}` → `{bookId}` |
| [R1xx-05](R1xx-05-id-field-naming.md) | id-field-naming | INFO | `bookID` + `author_id` → `id` + `authorId` |

## R2xx — methods & verbs

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R2xx-01](R2xx-01-get-no-body.md) | get-no-body | ERROR | `@GetMapping` + `@RequestBody` → `@RequestParam` |
| [R2xx-02](R2xx-02-post-creates-201.md) | post-creates-201 | WARN | POST → 200 → `@ResponseStatus(CREATED)` |
| [R2xx-03](R2xx-03-patch-partial.md) | patch-partial | WARN | PATCH + full-entity body → dedicated partial DTO |
| [R2xx-04](R2xx-04-delete-no-body.md) | delete-no-body | WARN | DELETE + body → query params / batch path |
| [R2xx-05](R2xx-05-custom-method-post.md) | custom-method-post | INFO | `GET /members/export` → `POST /members:export` |
| [R2xx-06](R2xx-06-put-full-update.md) | put-full-update | WARN | PUT + request DTO ≠ response type → same type (or PATCH) |

## R3xx — pagination

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R3xx-01](R3xx-01-list-paginated.md) | list-paginated | WARN | `List<T>` no params → `Pageable` or `page`/`size` |
| [R3xx-02](R3xx-02-list-envelope.md) | list-envelope | INFO | bare `List<T>` → `Page<T>` / named envelope |
| [R3xx-03](R3xx-03-unbounded-page-size.md) | unbounded-page-size | INFO | `int size` → `@Max(100) int size` |

## R4xx — payload

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R4xx-01](R4xx-01-no-entity-in-payload.md) | no-entity-in-payload | ERROR | `@RequestBody Book` (entity) → DTO |
| [R4xx-02](R4xx-02-field-casing.md) | field-casing | WARN | `author_name` → `authorName` |
| [R4xx-03](R4xx-03-time-field-standard.md) | time-field-standard | INFO | `String createdAt` → `Instant createdAt` |

## R5xx — errors

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R5xx-01](R5xx-01-unified-error-shape.md) | unified-error-shape | WARN | handler → `Map` → shared `ErrorResponse` |
| [R5xx-02](R5xx-02-no-500-for-business.md) | no-500-for-business | ERROR | business exception → 500 → 409/422 |
| [R5xx-03](R5xx-03-status-semantics.md) | status-semantics | WARN | POST create → 200 → `@ResponseStatus(CREATED)` |

## R6xx — http-grpc & versioning

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R6xx-01](R6xx-01-versioned-path.md) | versioned-path | WARN (default-OFF) | `/api/orders` → `/api/v1/orders` |
| [R6xx-02](R6xx-02-grpc-standard-methods.md) | grpc-standard-methods | INFO | `rpc DoMagic` → `GetBook` / `ArchiveBook` |

## R6xx demo set — golden-fixture rules (w2-04/w2-07)

| ID | Rule | Sev | Quick example (violation → fix) |
|---|---|---|---|
| [R6xx-91](R6xx-91-demo-bad-method-name.md) | demo-bad-method-name | ERROR | `demoBadGetOrder` → `getOrder` |
| [R6xx-92](R6xx-92-demo-no-response-type.md) | demo-no-response-type | WARN | void endpoint → declared response type |
| [R6xx-93](R6xx-93-demo-get-with-body.md) | demo-get-with-body | INFO | GET + body → query params |
| [R6xx-94](R6xx-94-demo-legacy-path.md) | demo-legacy-path | INFO | `/legacy/reports` → `/api/v1/reports` |
| [R6xx-99](R6xx-99-demo-trailing-slash.md) | demo-trailing-slash | ERROR | trailing `/` → trimmed path |

## Where the corpus lives

- Golden showcase: `testdata/golden/java-spring/` (every rule fires, 3
  formats pinned) and `testdata/golden/java-spring-clean/` (exit 0 — the
  FP guard). Per-rule violation/clean mapping in
  [`testdata/golden/java-spring/README.md`](../../testdata/golden/java-spring/README.md).
- Per-rule adversarial cases: `testdata/adversarial/<rule-id>/` (R1xx
  seed, w4-01 grows it), `testdata/methods-rules/` (R2xx), and the
  `testdata/spring-errors/` (R5xx/R6xx) e2e fixtures.
- Coverage gate: `internal/golden/rule_coverage_test.go` (a registered
  rule without a golden violation fails CI; the clean rewrite must stay
  silent).
