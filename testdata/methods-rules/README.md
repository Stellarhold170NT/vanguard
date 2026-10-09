methods-rules fixture (w3-04)

Synthetic Spring Boot repo (charter §7, Phụ lục B-4: never copy proprietary
code) — the per-rule fixture corpus of the R2xx family (methods & verbs).
Six controllers, each carrying one POSITIVE sample (the rule fires) and
negative samples (the rule stays silent) for its documented heuristic:

| Controller | Rule | Positive | Negative samples |
|---|---|---|---|
| `GetBodyController` | R2xx-01 get-no-body | `@GetMapping("/auto-complete")` + `@RequestBody` | get by id (path param), by-title (query param) |
| `CreateController` | R2xx-02 post-creates-201 | `@PostMapping("/direct")` returning `ResponseEntity<WidgetDto>` at the implicit 200 | `@ResponseStatus(CREATED)` create; `void` POST; collection response; action paths `/validate`, `/matches` |
| `PatchController` | R2xx-03 patch-partial | `fullPatch` body `GadgetDto` == response type | `patchStatus` with partial `GadgetStatusUpdate` |
| `DeleteController` | R2xx-04 delete-no-body | `deleteMany(@RequestBody List<String>)` | `deleteOne` path param only |
| `ActionController` | R2xx-05 custom-method-post | `@GetMapping("/{id}/generate-code")` (suggests `POST /api/v1/books/{id}:generate-code`) | `@PostMapping("/{id}:archive")` (already AIP-136), plain `/{id}` |
| `PutController` | R2xx-06 put-full-update | `partialPut` body `CogUpdateRequest` ≠ response `CogDto` | `fullPut` body `CogDto` == response type |

The exact finding set is pinned by `rules/methods_scan_test.go` (external
test package — internal/discovery wires the spring adapter this fixture
exercises). Line numbers in the expected set move with the controllers;
keep the positive samples one per rule and update the test together with
the fixture.
