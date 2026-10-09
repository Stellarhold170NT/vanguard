// Package spring turns the raw Java candidates extracted by adapters/java
// (w3-01) into a meaningful Spring Boot ir.ApiSurface (w3-02): controllers
// become Services, mapping annotations become Methods with merged paths,
// binding annotations become Params/Payload, ResponseEntity/Mono/Flux
// returns unwrap into Response, Pageable/page-size parameters become
// Pagination, @ControllerAdvice handlers become the ErrorScheme, referenced
// records/POJOs become Types, and build files pin Source.Framework.
//
// The optional springdoc/OpenAPI overlay merges operationId and schema
// names from a committed openapi/swagger yaml or json file into matched
// methods (Annotations keys "operationId"/"overlaySchemas", provenance
// marker "overlay"); it never invents or replaces source-derived data.
//
// The package imports only adapters/java and internal/ir — never
// internal/discovery, whose registry (the composition root) wires this
// adapter in. The Discovery.Adapter wrapper lives in
// internal/discovery/registry.go and assembles Evidence on our behalf.
//
// Known limitations of v0.1 (each pinned or documented in tests):
//
//   - Meta-annotations (@CustomGet = @GetMapping): the tree-sitter pass
//     does not capture @interface declarations, so meta-annotation
//     definitions are invisible. A mapping-like unknown annotation
//     (name ends in "Mapping" or a verb word) produces an unresolved
//     diagnostic and the method is skipped — never a failure.
//   - Wildcard paths ("/**") and matrix variables survive verbatim in the
//     merged path without semantic interpretation.
//   - @RequestMapping without method= defaults to GET plus an unresolved
//     marker; RequestMethod.HEAD maps to GET; OPTIONS/TRACE skip the
//     method (the IR has no verb for them).
//   - Multiple paths on one mapping ({"/x", "/y"}): the first is kept,
//     with an unresolved note. Likewise multiple RequestMethods: the
//     first wins.
//   - Parameters without a binding annotation bind as query parameters
//     only when their type is scalar; a bare DTO parameter (model
//     attribute binding) is skipped with an unresolved note.
//   - @PageableDefault sets a default, not a cap: PageSizeCapped only
//     follows a @Max-style constraint on a size/limit parameter.
//   - Local @ExceptionHandler methods inside a controller are not
//     collected; ErrorScheme reads @ControllerAdvice classes only.
//   - Response.Envelope stays empty: recognizing an envelope shape is
//     rule territory, not discovery.
//   - Lombok: @Data/@Getter/@Value on a class count as getters for DTO
//     qualification, but field-level generated behavior is not modeled.
//   - springdoc declared in the build file alone has nothing to merge
//     (the spec exists at runtime only); only committed spec files merge.
package spring
