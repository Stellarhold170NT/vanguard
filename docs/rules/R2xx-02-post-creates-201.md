# R2xx-02 — post-creates-201

| | |
|---|---|
| ID | `R2xx-02` |
| Slug | `post-creates-201` |
| Category | `methods` |
| Default severity | **WARN** |
| AIP reference | AIP-133 |
| Family | R2xx — methods & verbs |

## Why

A POST that creates a resource must answer **201 Created** (or 202
Accepted) so clients can tell creation from a plain success (AIP-133).
The implicit 200 hides the created-resource semantics and loses the
`Location` convention.

## What fires

The rule cannot read intent, so it approximates "creates a resource" and
**stays silent whenever the approximation cannot decide** (FP priority
#1). A POST is treated as a create only when ALL of:

1. `Response.StatusCode` is neither 201 nor 202 (0 = undeclared, and any
   other declared status);
2. the response type is a single named resource: non-void and
   non-collection — a void POST has too weak a create signal;
3. the method-level path is create-shaped (the service base path is
   removed first — the base is one controller-wide naming decision; a
   POST on the base itself counts as the collection root): no `:action`
   custom-method suffix anywhere, no action-verb token (the R2xx-05
   lexicon) in any literal segment, and a literal final segment.

Because the detection is heuristic, the message carries "verify that this
POST creates, and answer 201".

## What stays silent (FP shield)

- POST answering 201/202; POST returning void or a collection; POST on an
  action path (`/import`, `/books/{id}:generate-code`).
- `@ResponseStatus(HttpStatus.CREATED)` at method OR class level (method
  wins; w3-02 adapter).

## Spring example — compliant

```java
@PostMapping
@ResponseStatus(HttpStatus.CREATED)
public BookDto createBook(@RequestBody BookDto dto) { ... }
```

## Spring example — violation

```java
// create-shaped POST answering the implicit 200.
@PostMapping
public BookDto createBook(@RequestBody BookDto dto) { ... }
```

## Suppression

```java
// vanguard:ignore R2xx-02 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for a command endpoint spelled without an action word
whose 200-with-body is intended.

## Scope notes

- The companion rule R5xx-03 (errors family) carries the same defect from
  the status-semantics side; the two can fire on the same endpoint — one
  is the verb contract, the other the error-envelope contract.
- The action-word lexicon is shared with R2xx-05 (`rules/methods.go
  actionVerbs`); extending it changes both rules and needs a fixture row.
