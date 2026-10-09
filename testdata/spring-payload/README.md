spring-payload fixture (w3-05)

Synthetic Spring Boot repo (charter §7, Phụ lục B-4: never copy proprietary
code) — the R3xx (pagination) + R4xx (payload schema) end-to-end corpus.
Every rule of the w3-05 set has at least one positive and one negative
sample here; the scan test in the rules package (rules_test) pins the
EXACT finding set, so any heuristic drift turns red instead of shipping.

Layout (8 source files), paths under src/main/java/com/example/payload:

| Path                    | Construct under test |
|-------------------------|----------------------|
| `order/OrderController.java` | `listOrders()` bare `List` unpaginated → R3xx-01 + R3xx-02; `page(Pageable)` + `Page<>` → negative for both; `search(size)` without `@Max` → R3xx-03 (anchored on the param) |
| `order/OrderResponse.java`   | clean DTO: lowerCamelCase fields, `Instant placedAt` → negative for R4xx-02/R4xx-03 |
| `customer/CustomerController.java` | returns the `Customer` entity directly → R4xx-01 (ERROR) |
| `domain/Customer.java`       | `@Entity` POJO with a snake_case `full_name` field — must NOT trip R4xx-02 (entities are out of the DTO-casing scope) |
| `audit/AuditController.java` | `recent(limit)` with `@Max(200)` → R3xx-03 negative (capped) while the bare list still trips R3xx-01/R3xx-02; `audit()` pulls AuditDto into the type closure |
| `dto/AuditDto.java`          | `created_by` snake_case → R4xx-02; `updatedAt` as `java.util.Date` → R4xx-03; `startDate` as `LocalDate` (date-only name) and `recordedAt` as `Instant` → negatives |
| `web/YouthController.java`   | returns a DTO whose simple name matches a typical entity name (`Youth`) with no entity in scope → R4xx-01 must stay silent (name alone is not a signal) |
| `web/Youth.java`             | plain record DTO — never an entity |

Known IR gap (not fixable in w3-05, recorded in the task report): the
spring adapter keeps ONE IR Type per simple name (first declaration wins),
so an entity and a DTO that literally share a simple name cannot both be
in scope — the shadowed-name anti-FP branch of R4xx-01 is pinned at the
unit level (rules/payload_test.go) instead.
