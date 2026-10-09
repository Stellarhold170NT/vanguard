# R4xx-03 — time-field-standard (INFO)

Time-like fields should use `Instant`/`OffsetDateTime` — the types that
serialize as RFC3339 (AIP-142) — instead of `String` or legacy date
types. The recon found no direct hit (declared transparently in charter
§3.4); the rule is a guard for the 182 hand-written DTOs, which will
grow.

## Fires when

ALL of these hold on the same `ir.Field` (of a non-entity type):

1. the effective JSON name is **time-like**: the last camelCase word —
   separators and camel humps split the words — is one of `at, time,
   timestamp, date, when, until, since, deadline, expiry, expires`
   (`createdAt`, `validUntil`, `startDate`, `lastModifiedAt` qualify;
   `candidate`, `format`, `validate` do NOT — they are single words, the
   substring traps do not fire);
2. the field type base (`java.util.Date` → `Date` after the adapter's
   package resolution) is **not** one of the standard types
   `Instant`/`OffsetDateTime`/`ZonedDateTime`;
3. the type is a KNOWN legacy spelling: `String`, `CharSequence`, `Date`,
   `Calendar`, `Timestamp`, `LocalDateTime`, `LocalTime`, `long`, `Long`,
   `int`, `Integer`. Unknown custom types stay silent — the rule
   under-reports rather than guessing.

## The date-only carve-out

`LocalDate` is acceptable when the name's last word is `date`
(`startDate`, `dueDate`) — a date-only value has no zone to lose. The
same `LocalDate` on an instant-like name (`createdAt: LocalDate`) fires.

## Stays silent when

- the field belongs to an entity type (persistence territory, same scope
  line as R4xx-02);
- the type is standard (RFC3339-ready) or unknown.

## Scope notes (attack surface — W4, read this)

- Heuristic = name suffix word + type table, both closed and listed
  above. A time field named `occurrence` or `dob` is invisible in v0.1.
- Epoch-millis `long` fields named `*At` fire; renaming to
  `epochMillis`-style names (no time-like last word) silences them.
- The charter's config-driven convention override (allow-list of custom
  time types in `.vanguard.yaml`) is future work — the rule declares no
  options in v0.1.

## Good / Bad

```java
// good — RFC3339 on the wire
public record AuditDto(Instant createdAt, Instant validUntil) { ... }

// bad — unstructured strings and legacy types
public record AuditDto(String createdAt, Date validUntil) { ... }
```
