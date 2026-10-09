# R5xx-02 — no-500-for-business

| | |
|---|---|
| ID | `R5xx-02` |
| Slug | `no-500-for-business` |
| Category | `errors` |
| Default severity | **ERROR** |
| AIP reference | AIP-193 |
| Family | R5xx — errors |

## Why

A business exception (declared in the scanned repo) must not be mapped to
HTTP 500 — 500 is reserved for unexpected server failures (AIP-193). A
business rule violation is a 4xx outcome naming the failure; routing it
through 500 also fires every naive 5xx alerting page.

## What fires

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

## What stays silent

- External exceptions (`Exception.class`, library types) — no ownership
  signal, no finding.
- App-declared classes without the `Exception` suffix
  (`ValidationFailed`) — the name signal fails, under-report by design.
- An app exception mapped to 500 **implicitly** (no `@ResponseStatus`
  anywhere) is out of scope: the IR cannot prove the effective status.

## Spring example — compliant

```java
// the same business failure names its outcome
@ExceptionHandler(BusinessException.class)
@ResponseStatus(HttpStatus.CONFLICT)
public ErrorResponse handleBusiness(BusinessException ex) { ... }
```

## Spring example — violation

```java
// an app-owned business exception routed through 500
@ExceptionHandler(BusinessException.class)
@ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
public ErrorResponse handleBusiness(BusinessException ex) { ... }
```

## Suppression

```java
// vanguard:ignore R5xx-02 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1).

## Scope notes (attack surface — W4, read this)

- Deliberately FP-first: no type-resolver, no inheritance chase (backlog
  per the charter) — only the declared package of a class the scan
  actually saw.
- Inheritance is not chased: an app exception extending a technical base
  is judged by its own name + ownership.
- The rule fires on the handler node; an exception class carrying
  `@ResponseStatus(INTERNAL_SERVER_ERROR)` but handled by *no* advice is
  invisible here (no handler, no anchor).
- Response-type-less (`void`) handlers still count when they declare the
  status — the body is irrelevant to the status defect.
