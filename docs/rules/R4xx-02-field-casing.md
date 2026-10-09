# R4xx-02 — field-casing

| | |
|---|---|
| ID | `R4xx-02` |
| Slug | `field-casing` |
| Category | `payload` |
| Default severity | **WARN** |
| AIP reference | AIP-140 |
| Family | R4xx — payload |

## Why

DTO fields must serialize as lowerCamelCase (AIP-140). Mixed casing in
the payload JSON is the natural byproduct of 182 hand-written DTOs mixing
conventions (w1-03 §2.4) — the rule catches it at the DTO level.

Charter §3.2 conflict note: DTO **field** casing belongs HERE (the R4xx
payload family); R1xx keeps path/resource naming and the id-field
convention.

## What fires

- the `ir.Field`'s **effective JSON name** is not lowerCamelCase, where
  effective = `Field.JSONName` when the adapter resolved
  `@JsonProperty`/`@SerializedName`, `Field.Name` otherwise;
- lowerCamelCase means: first character is a lower-case letter, every
  character is a letter or digit, no `_`/`-`/`.`/space anywhere. So
  `created_by`, `CreatedBy` and `URL` all fire; `createdBy`,
  `addressLine2` do not.

## What stays silent

- the field belongs to an entity type (`Type.IsEntity`): persistence
  casing is not wire territory, and entities must not be on the wire at
  all (R4xx-01 owns that);
- the name is already lowerCamelCase.

## Suggestion

The lowerCamelCase re-spelling of the flagged name — replacement text,
not advice: `created_by` → `createdBy`, `author_name` → `authorName`,
`CreatedBy` → `createdBy`.

## Spring example — compliant

```java
// lowerCamelCase on the wire
public record BookDto(Long id, String authorName) { ... }
```

## Spring example — violation

```java
// snake_case on the wire
public record BookDto(Long id, @JsonProperty("author_name") String authorName) { ... }
```

## Suppression

```java
// vanguard:ignore R4xx-02 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — the escape hatch for a global Jackson `SNAKE_CASE`
strategy the IR cannot see in v0.1.

## Scope notes (attack surface — W4, read this)

- The check reads the IR only; it cannot see Jackson strategy config
  (`SNAKE_CASE` global naming). A codebase that deliberately serializes
  snake_case via a global strategy will see WARN noise — per-path or
  family suppression is the escape hatch, and the IR cannot distinguish
  it in v0.1.
- Only the effective JSON name is judged: renaming the Java field but
  keeping `@JsonProperty("snake_name")` still fires (the wire name is
  what clients see).
