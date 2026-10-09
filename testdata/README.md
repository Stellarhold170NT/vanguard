# testdata

Corpora for the Vanguard test architecture (`docs/test-strategy.md`, w1-05).
The Go tool ignores directories named `testdata` by convention, so fixtures
here are never treated as packages.

Planned layout (populated from W2 onwards):

| Path | Purpose | Task |
|---|---|---|
| `stub-repo/` | Minimal fake project exercising discovery + language detection | w2-03 |
| `java-adapter/` | Synthetic Java files exercising the tree-sitter adapter (see its README) | w3-01 |
| `spring-repo/` | Synthetic Spring Boot service exercising the Spring mapping (see its README) | w3-02 |
| `spring-overlay/` | Same repo shape with a committed OpenAPI spec for the springdoc overlay | w3-02 |
| `spring-errors/` | Spring advice + controllers + one `.proto` exercising the R5xx/R6xx families (see its README) | w3-06 |
| `golden/` | Golden snapshot inputs/outputs for the render and IR contracts | w2-07 |
| `adversarial/<rule-id>/` | `ok-<slug>.java` / `vio-<slug>.java` cases, ≥3 vio + ≥2 ok per rule | w4-01 |
| `mutation/` | Mutator definitions for the catch-rate harness (target ≥90%) | w4-02 |

Rules for corpus authors:

- Cases are at most 40 lines and simulate patterns — **never** copy code from
  proprietary projects (charter §7, Phụ lục B-4).
- Every rule directory must contain at least one `vio-` and one `ok-` case.
- The manifest is `adversarial/manifest.yaml` (schema in test-strategy §3.6).
