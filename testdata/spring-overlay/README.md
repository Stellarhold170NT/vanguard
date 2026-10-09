spring-overlay fixture (w3-02)

Minimal Spring Boot repo exercising the springdoc/OpenAPI overlay merge:

- `pom.xml` — spring-boot-starter-parent 3.2.5 (framework version source),
  NO springdoc dependency: the overlay must work from the spec file alone.
- `src/main/java/com/example/overlay/BookController.java` — 3 handlers:
  `list()` GET /api/books, `get()` GET /api/books/{id}, `create()`
  POST /api/books.
- `src/main/java/com/example/overlay/BookDto.java` — record DTO.
- `src/main/resources/openapi.yaml` — spec with operationIds for the two
  GET operations only; `create` has no spec entry (pins that a method the
  spec does not describe stays untouched).

Expected overlay merge (pinned byte-exact by `adapters/spring` snapshot
test against `snapshot.json`):

| Method | Path             | overlay annotations                            |
|--------|------------------|------------------------------------------------|
| list   | /api/books       | operationId=listBooks, overlaySchemas=BookDto  |
| get    | /api/books/{id}  | operationId=getBook, overlaySchemas=BookDto    |
| create | /api/books       | none (POST absent from the spec)               |

Every merged method also carries `overlay` = the spec file path
(`src/main/resources/openapi.yaml`) — the provenance marker required by
the brief ("đánh dấu nguồn (overlay)").
