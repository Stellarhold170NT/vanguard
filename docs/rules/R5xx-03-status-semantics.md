# R5xx-03 — status-semantics

| | |
|---|---|
| ID | `R5xx-03` |
| Slug | `status-semantics` |
| Category | `errors` |
| Default severity | **WARN** |
| AIP reference | AIP-133, AIP-193 |
| Family | R5xx — errors |

## Why

A POST that creates a resource must answer 201 (Created) or 202 (Accepted)
— never 200 (AIP-133/193; w1-03 pain 3: the app's convention is a 201 for
create, and a new endpoint drifting back to 200 is a silent regression).
This rule reads the **status side** of the contract; the verb/path side is
the charter's R2xx-02 (the two are a designed pair and may overlap on the
same endpoint).

## What fires

ALL of these hold on the same `ir.Method`:

1. `Verb == POST`;
2. the effective status is 200: explicitly declared (`@ResponseStatus`)
   **or** undeclared — the IR documents `0` as "not declared, consumers
   infer it from the verb", and the POST convention is 200;
3. the response declares a single body (`ResponseType != ""` and not a
   collection) — a creation returns the created resource, while action
   POSTs answer collections or nothing;
4. the path is **create-shaped**: the final path segment is a literal with
   no template variable (`/orders` yes, `/orders/{id}` no) and no AIP-136
   custom-method colon, and the segment is not an action word
   (`/orders/{id}/cancel` no).

## What stays silent

- 201/202 answers; non-POST verbs; void or collection responses; action
  paths; any other declared status (204, 3xx, 4xx, 5xx) on a create-shaped
  POST is ambiguous and stays silent in v0.1.

## Spring example — compliant

```java
// creation is visible in the status
@PostMapping
@ResponseStatus(HttpStatus.CREATED)
public OrderDto create(@RequestBody CreateOrderRequest req) { ... }
```

## Spring example — violation

```java
// 200 (explicit or the implicit convention) hides the creation
@PostMapping
@ResponseStatus(HttpStatus.OK)
public OrderDto create(@RequestBody CreateOrderRequest req) { ... }
```

## Suppression

```java
// vanguard:ignore R5xx-03 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1).

## Scope notes (attack surface — W4, read this)

- The action-word guard is an exact-token lexicon (`r5xxActionWords`, no
  stemming — the w3-04 discipline). Nouns-that-are-also-verbs (`draft`,
  `batch`) are deliberately NOT in it; a miss under-reports.
- Statuses set through `ResponseEntity.status(...)` builder chains are not
  extracted by the w3-02 adapter (only `@ResponseStatus`), so a
  builder-driven 201 reads as undeclared — it fires. The adapter
  limitation is the known cost of the implicit-200 reading; W4 evidence
  will decide whether the adapter learns the builder.
- Deliberate overlap with R2xx-02 (charter: "cặp với R2xx-02"): both may
  flag the same drifted endpoint from their two sides.
