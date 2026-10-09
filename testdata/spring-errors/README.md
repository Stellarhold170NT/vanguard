# spring-errors — w3-06 fixture repo

A minimal Spring Boot app exercising the R5xx (errors) and R6xx
(versioning / proto declaration) families end to end: advice + controllers
+ one `.proto`. Driven by `rules/errors_scan_test.go` (package `rules_test`),
which pins the exact finding set per file through the real pipeline
(walk → Spring adapter incl. the proto declaration reader → engine).

## Positive / negative map

| File | Expected findings | Why |
|---|---|---|
| `advice/GlobalExceptionHandler.java` | `R5xx-01` ×1 | `handleValidation` returns `Map<String, Object>` while the other three handlers share the `ApiError` envelope (auto: majority envelope is the standard) |
| `advice/GlobalExceptionHandler.java` | `R5xx-02` ×1 (ERROR) | `BusinessException` is declared in `com.example.shop.service` and mapped to `@ResponseStatus(INTERNAL_SERVER_ERROR)` |
| `advice/GlobalExceptionHandler.java` | (silent: `R5xx-02` ×3) | 404 outcome on `NotFoundException`; external `IllegalArgumentException` and `ValidationException` are not declared in the repo (no package signal) |
| `controller/OrderController.java` | `R5xx-03` ×2 | `create` (explicit `@ResponseStatus(OK)`) and `createV2` (implicit 200) on the create-shaped `/api/v1/orders` POST |
| `controller/OrderController.java` | (silence) | `createDraft` answers 201; `cancel` is an action path; `get` is a GET |
| `src/main/proto/library.proto` | `R6xx-02` ×1 | `DoMagic` is neither a Standard Method name nor VerbNoun; `GetBook`/`ListBooks`/`SyncData`/`GetAuditTrail` are compliant; the commented-out `GhostService` is invisible |
| `dto/*.java`, `service/BusinessException.java`, `exception/NotFoundException.java` | (silence) | envelope type, created DTO, and the two exception declarations carry no API surface |

`R6xx-01` (versioned-path) stays silent here by design: it is
default-disabled (charter §3.6) and the fixture's base paths carry `/v1`.
