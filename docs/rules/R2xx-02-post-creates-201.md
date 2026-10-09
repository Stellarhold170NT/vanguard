# R2xx-02 post-creates-201

`category: methods · severity: WARN · AIP-133` — charter §3.2 (w3-04).

A POST endpoint that creates a resource should declare 201 Created, not the
implicit 200.

## Heuristic (convention — verify manually)

The rule cannot read intent, so it approximates "creates a resource" and
**stays silent whenever the approximation cannot decide** (FP priority #1).
A POST is treated as a create only when ALL of:

1. `Response.StatusCode` is neither 201 nor 202 (0 = undeclared, and any
   other declared status);
2. the response type is a single named resource: non-void
   (`Response.Type.Name != ""`) and non-collection
   (`Response.IsCollection` / `Type.IsCollection` false) — a void POST has
   too weak a create signal;
3. the path is create-shaped: no `:action` custom-method suffix anywhere,
   no action-verb token (see the R2xx-05 lexicon) in any literal segment,
   and a literal final segment — a create posts to a collection
   (`/books`, `/books/{id}/reviews`), never to an item template
   (`/books/{id}`).

Because the detection is heuristic, the message carries
"verify that this POST creates, and answer 201".

## Suggestion

`@ResponseStatus(HttpStatus.CREATED) (or return ResponseEntity.status(HttpStatus.CREATED))`

The adapter reads the status from `@ResponseStatus` at method OR class
level (method wins; w3-02), and from the IR `Response.StatusCode` field.

## False-positive surface

* A non-create POST that returns a single DTO on a noun path (e.g. a
  command endpoint spelled without an action word) — the message says
  "verify manually"; suppress per path or downgrade via config (§6.4.1).
* A create that intentionally answers 200 with a body — a real convention
  breach per AIP-133; keep the finding.
