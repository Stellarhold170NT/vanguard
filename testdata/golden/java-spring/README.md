# java-spring — the W3 golden showcase (w3-07)

One synthetic Spring Boot "library" service whose endpoints collectively
violate **every registered rule** (22 real W2/W3 rules + 5 demo rules),
with a **clean counterpart per rule** either in this repo or in the
companion case [`../java-spring-clean/`](../java-spring-clean/) (the clean
rewrite of the same service — exit 0, zero findings, with `R6xx-01` opted
in and silent).

Snapshots pin the full finding set (32 findings: 5 ERROR · 15 WARN ·
12 INFO) in all three formats. `.vanguard.yaml` opts into `R6xx-01`
(default-off policy rule) in both cases.

## Rule → violating fixture

| Rule | Severity | Fires on | Why |
|---|---|---|---|
| `R1xx-01` | WARN | `loan/LoanController.java:21` | singular collection segment `/api/v1/loan` |
| `R1xx-02` | ERROR | `book/BookController.java:41` | CRUD verb `find` in the path |
| `R1xx-03` | WARN | `book/BookController.java:41` | variable `{id}` under the action segment `find-by-id` |
| `R1xx-04` | WARN | `book/BookController.java:36` | path variable `{BookId}` not lowerCamel |
| `R1xx-05` | INFO ×2 | `book/BookDto.java:14` | mixed id spellings (`bookID` vs `author_id`) + self-id divergence |
| `R2xx-01` | ERROR | `member/MemberController.java:36` | `autoComplete` GET declares a body |
| `R2xx-02` | WARN | `book/BookController.java:46` | create-shaped POST without 201 |
| `R2xx-03` | WARN | `member/MemberController.java:51` | PATCH body typed as the full entity |
| `R2xx-04` | WARN | `member/MemberController.java:41` | DELETE with a body |
| `R2xx-05` | INFO ×2 | `book/BookController.java:41`, `member/MemberController.java:56` | action-shaped paths on GET |
| `R2xx-06` | WARN | `member/MemberController.java:46` | PUT payload type ≠ response type |
| `R3xx-01` | WARN | `member/MemberController.java:56` | unpaginated collection (`export`) |
| `R3xx-02` | INFO ×3 | `BookController.java:31`, `LoanController.java:21`, `MemberController.java:56` | bare `List<…>` responses |
| `R3xx-03` | INFO | `book/BookController.java:31` | `size` param without `@Max` |
| `R4xx-01` | ERROR | `book/BookController.java:51` | ORM entity `Book` on the wire |
| `R4xx-02` | WARN ×2 | `book/BookDto.java:14` | snake_case fields `author_id`, `created_at` |
| `R4xx-03` | INFO | `book/BookDto.java:14` | time-like field typed `String` |
| `R5xx-01` | WARN | `err/GlobalExceptionHandler.java:37` | `Map` handler deviating from the `ErrorResponse` envelope |
| `R5xx-02` | ERROR | `err/GlobalExceptionHandler.java:26` | app-declared `BusinessException` mapped to 500 |
| `R5xx-03` | WARN | `book/BookController.java:46` | POST create relying on the implicit 200 |
| `R6xx-01` | WARN | `legacy/LegacyReportController.java:15` | base path without a version segment |
| `R6xx-02` | INFO | `src/main/proto/library.proto:7` | rpc `DoMagic` off the standard-method pattern |
| `R6xx-91` | ERROR | `order/OrderController.java:19` | `demoBadGetOrder` marker name (demo family) |
| `R6xx-92` | WARN ×2 | `order/OrderController.java:19,23` | void endpoints (demo family) |
| `R6xx-93` | INFO | `member/MemberController.java:36` | GET with a body (demo family) |
| `R6xx-94` | INFO | `legacy/LegacyReportController.java:18` | `/legacy/` path (demo family) |
| `R6xx-99` | — | (unit-level: `rules/demo_test.go`) | the IR path contract (`adapters/spring` `mergePath`) normalizes trailing slashes away, so no adapter can produce one — the rule is pinned by unit tests on hand-built IR, not by a golden fixture |

## Rule → clean counterpart

| Rule | Clean counterpart |
|---|---|
| `R1xx-01` | every collection segment plural (`/api/v1/books`, `/api/v1/members`, …) |
| `R1xx-02` | `BookController.getBook` — `GET /{bookId}`, verb carried by HTTP |
| `R1xx-03` | `MemberController.getMember` — `/api/v1/members/{memberId}` |
| `R1xx-04` | every variable lowerCamel (`{bookId}`, `{memberId}`, `{orderId}`) |
| `R1xx-05` | `MemberDto`/`LoanDto`/`AuthorDto` — `id` + `<resource>Id` |
| `R2xx-01` | `MemberController.autoComplete` in the clean case — `@RequestParam` |
| `R2xx-02` | `AuthorController.createAuthor` — `@ResponseStatus(CREATED)` |
| `R2xx-03` | clean-case `patchMember` — dedicated `MemberPatch` type |
| `R2xx-04` | clean-case `deleteMember` — no body |
| `R2xx-05` | clean-case proto `ArchiveBook` — VerbNoun custom method |
| `R2xx-06` | clean-case `updateBook` — PUT with the same type it returns |
| `R3xx-01` | `AuthorController.listAuthors` — page/size params |
| `R3xx-02` | `Page<AuthorDto>` — Spring's paging envelope |
| `R3xx-03` | `@Max(100)` on the size param |
| `R4xx-01` | clean-case `domain/Book` — entity exists, never on the wire |
| `R4xx-02` | every clean DTO field lowerCamel |
| `R4xx-03` | `Instant createdAt/borrowedAt/generatedAt` |
| `R5xx-01` | clean-case advice — one `ErrorResponse` envelope on every handler |
| `R5xx-02` | clean-case `handleBusiness` — 409 Conflict |
| `R5xx-03` | clean-case creations answer 201 |
| `R6xx-01` | clean-case services — every base carries `/v1` (rule opted in, silent) |
| `R6xx-02` | clean-case proto — Standard Methods + VerbNoun only |
| `R6xx-91..94` | clean case — no markers, typed responses, no GET bodies, no `/legacy/` |
| `R6xx-99` | unit-level (see above) |

## Deliberate extras (pinned, correct)

- `R2xx-05` also fires on `findBookById` — the route **is** action-shaped
  (AIP-136); the correct fix is the R1xx-02 suggestion, not the custom
  method reshape.
- `R1xx-05` reports the self-id divergence (`bookID` vs the `id` used
  elsewhere) in addition to the mixed reference spellings.
