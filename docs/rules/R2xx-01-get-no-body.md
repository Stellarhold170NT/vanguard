# R2xx-01 get-no-body

`category: methods · severity: ERROR · AIP-131` — charter §3.2 (w3-04).

GET endpoints must not declare a request body — many HTTP clients silently
drop GET bodies.

## Heuristic (measured, the family's one ERROR)

The rule fires when the IR shows **Verb == GET** and the method declares a
request body — either view of the w2-02 invariant (`Payload` non-nil, or an
`in=body` Param; the stub adapter may set either half alone, the spring
adapter always fills both). The claim is not a convention: the handler
itself declares input on the one verb whose bodies popular HTTP clients
drop, so the finding is a broken contract, not a style note.

gRPC operations (`Verb: rpc`) never match — HTTP verb semantics do not
apply to them.

## Suggestion

`move the payload to query parameters` — the finding anchors at the body
parameter (child node, not the whole method).

## False-positive surface

None known for v0.1: adapters derive Verb from the mapping annotation and
the body from `@RequestBody`; a false finding means an adapter bug, which
the w3-01/w3-02 adapter tests pin. If a stack genuinely requires GET bodies
(e.g. Elasticsearch-style DSL over GET), suppress per path (§6.4.1).
