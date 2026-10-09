# R2xx-06 put-full-update

`category: methods · severity: WARN · AIP-134` — charter §3.2 (w3-04).

A PUT endpoint should receive the full replacement entity — a payload type
that differs from the declared response type looks like a partial update
(PATCH).

## Heuristic (convention — documented comparison, symmetric to R2xx-03)

Fires only when ALL of:

1. `Verb == PUT` (HTTP; rpc never matches);
2. the method declares a payload (`Payload != nil`);
3. the declared response type is non-empty AND `Payload.Name` differs from
   `Response.Type.Name` case-insensitively — `PUT` with
   `CogUpdateRequest` in and `CogDto` out reads as a partial update done
   through the wrong verb (w1-03 pain 7c, `partialUpdateYouth`).

Matching types (full replacement) or an undeclared response stay silent.

## Suggestion

`replace PUT with PATCH for this partial update`

## False-positive surface

A PUT whose request DTO deliberately differs from the response DTO while
still replacing the whole resource (e.g. a command-style `XxxRequest`
carrying every field) is indistinguishable at the signature level — the
message names both types so the reader can verify; suppress per path if
intended (§6.4.1). Transport wrappers on either side are already unwrapped
by the w3-02 adapter.
