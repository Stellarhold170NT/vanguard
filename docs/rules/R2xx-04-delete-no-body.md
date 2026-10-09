# R2xx-04 delete-no-body

`category: methods · severity: WARN · AIP-135` — charter §3.2 (w3-04).

DELETE endpoints should not declare a request body — model bulk deletes as
a batch path or query parameters.

## Heuristic (convention, transport-backed)

Fires when `Verb == DELETE` and the method declares a request body —
either view of the w2-02 invariant (`Payload` non-nil or an `in=body`
Param). Same failure mode as R2xx-01 (clients drop bodies), one severity
softer: a DELETE body is legal HTTP, the convention just routes bulk
deletes elsewhere (w1-03 pain 7b: `@DeleteMapping` + `@RequestBody
List<String> ids`).

## Suggestion

`move the body to query parameters (e.g. ?ids=1,2) or a batch path`

## False-positive surface

None known for v0.1 (same measured surface as R2xx-01). Stacks that
genuinely require DELETE bodies suppress per path (§6.4.1).
