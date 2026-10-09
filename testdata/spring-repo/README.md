spring-repo fixture (w3-02)

Synthetic Spring Boot repo (charter §7, Phụ lục B-4: never copy proprietary
code) — the end-to-end IR mapping corpus. It deliberately has NO OpenAPI
spec file (acceptance: the IR stays complete from annotations alone; the
pom declares springdoc but that alone has nothing to merge — runtime-only).

Layout (16 source files):

| Path under src/main/java/com/example/library | Construct under test |
|---|---|
| `LibraryApplication.java` | `@SpringBootApplication`, no endpoints — must yield no Service and no Type |
| `controller/BookController.java` | `@RestController` + base path merge; `ResponseEntity<Page<BookDto>>` unwrap + `Pageable` pagination (pageable style); `@PathVariable`; `@RequestBody` payload; `@ResponseStatus(HttpStatus.CREATED)` → 201; `void` response |
| `controller/BookUpdateController.java` | same base path as BookController (two services may share a base); multi-line parameter list; `@RequestParam(name=…)` rename + `@Max(100)` validation → pagination params style + PageSizeCapped |
| `controller/MemberController.java` | plain list/get/register endpoints; `ResponseEntity<MemberDto>` unwrap |
| `controller/AdminController.java` | `@RequestMapping(method = RequestMethod.DELETE)` → DELETE verb; returns a `@Entity` type directly (R4xx-01 signal) |
| `controller/LoanController.java` | nested record DTO (`BookStats`); `@RequestHeader` parameter; path with two template variables `/{id}/return` |
| `controller/HealthController.java` | no class-level mapping → BasePath "" |
| `controller/LegacyViewController.java` | plain `@Controller` (no `@ResponseBody`) — still a Service |
| `controller/EdgeController.java` | unknown mapping-like annotation `@CustomGet` → unresolved, method skipped; `@RequestMapping` without `method=` → GET + unresolved note; multi-path `{"/x","/y"}` → first wins + unresolved note; local `@ExceptionHandler` inside a controller — NOT collected (only `@ControllerAdvice` classes feed ErrorScheme, v0.1 limitation) |
| `advice/GlobalExceptionHandler.java` | `@RestControllerAdvice` + 3 `@ExceptionHandler` → ErrorScheme (404 / undeclared / 500) |
| `dto/BookDto.java` | record with `@JsonProperty("book_id")` → JSONName |
| `dto/MemberDto.java` | record with `@Size(max = 100)` field constraint |
| `dto/CreateBookRequest.java` | POJO with getters/setters + `@NotBlank` |
| `dto/LoanDto.java` | record with `@JsonProperty("loan_id")` |
| `dto/ErrorResponse.java` | POJO referenced only by the advice handlers |
| `member/Member.java` | `@Entity` POJO → IsEntity in IR |
| `exception/BookNotFoundException.java` | exception class — referenced by handlers, never a Type |

Expected IR is pinned by `adapters/spring` tests: 8 services, 7 types
(BookDto, CreateBookRequest, ErrorResponse, LoanDto, Member, MemberDto,
BookStats), 3 error handlers, 3 unresolved-annotation diagnostics
(@CustomGet, verb-less @RequestMapping, multi-path @PostMapping) and zero
Validate violations.
