# R5xx-02 — no-500-for-business (ERROR)

A business exception must never be mapped to HTTP 500 (AIP-193): 500 means
"the server is broken", while a violated business rule is a
client-comprehensible 4xx outcome. A 500 mapping actively lies to
monitoring (alerts on business flow) and to clients (retry semantics).

## Fires when

ALL of these hold on the same `ir.ErrorHandler`:

1. the mapping to 500 is **certain** — the declared status is 500, where
   the effective status is the handler's `@ResponseStatus` (or the advice
   class's), falling back to the exception class's own `@ResponseStatus`
   only when the handler declares none (that is Spring's resolution
   order);
2. the exception is **app-owned**: the adapter resolved its declaration
   inside the scanned repo (`ExceptionPackage != ""`) — the charter's
   "package signal", made structural. External fallbacks
   (`@ExceptionHandler(Exception.class)`) are the *correct* place for a
   500 and stay silent;
3. the simple name carries the `Exception` suffix (the charter's name
   signal; `DomainRuleViolation`-style names stay silent).

This is deliberately FP-first: no type-resolver, no inheritance chase
(backlog per the charter) — only the declared package of a class the scan
actually saw.

## Suggestion

`map <Exception> to a 4xx status naming the outcome (e.g. 409 Conflict or
422 Unprocessable Entity)`

## Scope notes (attack surface — W4, read this)

- An app exception mapped to 500 **implicitly** (no `@ResponseStatus`
  anywhere — Spring answers 200 for a body-returning handler, or 500 for
  the fallback path) is out of scope: the IR cannot prove the effective
  status, and the 200-with-error-body defect is a different rule.
- Inheritance is not chased: an app exception extending a technical base
  is judged by its own name + ownership, per the charter's heuristic.
- The rule fires on the handler node; an exception class carrying
  `@ResponseStatus(INTERNAL_SERVER_ERROR)` but handled by *no* advice is
  invisible here (no handler, no anchor).
- Response-type-less (`void`) handlers still count when they declare the
  status — the body is irrelevant to the status defect.

## Good / Bad

```java
// good — the business outcome names its own status
@ExceptionHandler(InsufficientBalanceException.class)
@ResponseStatus(HttpStatus.UNPROCESSABLE_ENTITY)
public ErrorResponse handle(InsufficientBalanceException ex) { ... }

// bad — the business rule reports as a server failure
@ExceptionHandler(InsufficientBalanceException.class)
@ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
public ErrorResponse handle(InsufficientBalanceException ex) { ... }
```
