# R5xx-01 — unified-error-shape (WARN)

Every `@ExceptionHandler` of the app must answer with the SAME error
envelope (AIP-193; w1-03 pain 5: the app has one `ExceptionTranslator`
contract, and a handler that returns its own `Map`/wrapper silently splits
it — clients would parse two error contracts depending on which failure
hit).

## The standard, and when it is provable

The rule never re-opens a source file. The standard comes from the IR:

- **auto (default)** — the envelope most handlers share IS the standard,
  but only when it is *provable*: at least **2** handlers return the same
  non-empty `ResponseType`, that spelling **strictly dominates** every
  other (no tie), and it **resolves to an IR Type** (the scan can see its
  fields). Without a consensus the rule stays silent — an unprovable
  standard must never produce a finding.
- **`scheme: code-message-details`** — config pins the expected shape: a
  handler whose response type resolves to an IR Type must carry `code` and
  `message` fields (serialized name, so a `@JsonProperty` rename counts).
- **`scheme: problem-json`** — the RFC-7807 reading: the resolved response
  type must carry at least **two** of `type`, `title`, `status`, `detail`,
  `instance`.

## Fires when

ALL of these hold on the same `ir.ErrorHandler`:

1. the handler declares a response body (`ResponseType != ""` — a void
   handler carries no envelope contract and is out of scope);
2. in auto mode: the scheme-wide majority exists (above) and this handler
   returns a **different** spelling — including an unresolvable one such
   as `Map<String, Object>`, which is exactly the w1-03 pain;
3. in a pinned scheme: the response type resolves to an IR Type **and**
   lacks the required fields.

## Suggestion

- auto: `answer with the shared <Envelope> envelope from every @ExceptionHandler`
- pinned: `answer with the unified error envelope (<required fields>)`

## Scope notes (attack surface — W4, read this)

- Field checking is impossible for unresolvable types (`Map`, external
  classes) — under a pinned scheme they stay silent (under-report).
- The majority inference trusts the adapters' ErrorScheme; handlers of
  *local* `@ExceptionHandler` methods inside controllers are not collected
  by the w3-02 adapter (documented limitation) and therefore out of scope.
- Two resolvable envelopes in one app (the "2 envelope contracts" case)
  fire on the minority handlers — deliberate: that IS the split contract.
- The RFC-7807 check is a signature-level reading of the DTO fields; it
  does not verify media type negotiation (`produces = "application/problem+json"`).

## Good / Bad

```java
// good — every handler answers with the one envelope
@ExceptionHandler(BookNotFoundException.class)
public ErrorResponse handleNotFound(BookNotFoundException ex) { ... }

// bad — a private envelope for one failure
@ExceptionHandler(ValidationFailed.class)
public Map<String, Object> handleValidation(ValidationFailed ex) { ... }
```
