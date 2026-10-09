# R2xx-03 patch-partial

`category: methods · severity: WARN · AIP-134` — charter §3.2 (w3-04).

A PATCH endpoint should send a partial payload — a body typed as the full
response entity means the update is a full replacement (PUT).

## Heuristic (convention — documented comparison)

Fires only when ALL of:

1. `Verb == PATCH` (HTTP; rpc never matches);
2. the method declares a payload (`Payload != nil`);
3. the declared response type is non-empty AND `Payload.Name` equals
   `Response.Type.Name` case-insensitively (the "body = entity cùng type
   response" heuristic of the brief) — the same type in and out reads as
   full-entity replacement.

When the payload type differs from the response type (a partial DTO) or the
response is undeclared, the heuristic cannot establish "full" and the rule
stays silent.

## Suggestion

`replace PATCH with PUT for this full-replacement update`

## False-positive surface

A PATCH that legitimately accepts the full representation but applies it
partially is indistinguishable from a full replacement at the signature
level — the message names the type so the reader can verify; suppress per
path if the pattern is intended (§6.4.1). Generic wrappers on the response
(`ResponseEntity<...>`, `Mono<...>`) are already unwrapped by the w3-02
adapter, so the comparison sees the body types, not the transport wrappers.
