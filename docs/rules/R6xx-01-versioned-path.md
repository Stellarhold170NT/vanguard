# R6xx-01 — versioned-path

| | |
|---|---|
| ID | `R6xx-01` |
| Slug | `versioned-path` |
| Category | `versioning` |
| Default severity | **WARN** (default-OFF) |
| AIP reference | AIP-215, AIP-180 |
| Family | R6xx — http-grpc & versioning |

## Why

The service base path should carry a version segment (AIP-215/180; w1-03
§2.1: the sample app has no `/v1` anywhere, and an unversioned base path
pins clients to an unversioned contract).

## Policy rule — enabled via config

This is a **policy** rule, not a defect detector (charter §3.6): whether
an API is versioned is a choice. It ships **disabled by default** and
floods nothing until a config enables it:

```yaml
rules:
  R6xx-01:
    disabled: false
    options:
      versionPattern: "/v[0-9]+"   # optional; any Go regexp
```

## What fires

ALL of these hold on the same `ir.Service`:

1. the service declares a base path (`BasePath != ""`) — the class-level
   `@RequestMapping` the adapter merged; a controller without one has
   nothing to check at service level;
2. the effective pattern does **not** match the base path.

The default pattern `/v[0-9]+` is a segment-anchored substring: `/api/v1`
matches, `/api/rev1` does not, `/api/v10` does. The `versionPattern`
option replaces it with any Go regexp (anchor it with `^/api/v\d+` for an
explicit position).

## What stays silent

- Services without a base path (method-level versioning is a future
  extension, not the charter's contract).
- gRPC services surface through `GrpcServices`, not `Service.BasePath` —
  proto package versioning is out of scope here.

## Spring example — compliant

```java
// clients pin the version
@RequestMapping("/api/v1/orders")
```

## Spring example — violation

```java
// an unversioned base path
@RequestMapping("/legacy/reports")
```

## Suppression

```java
// vanguard:ignore R6xx-01 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for a legacy module whose unversioned URL is the
published contract.

## Scope notes (attack surface — W4, read this)

- A **malconfigured pattern** (a regexp that does not compile) is loud:
  the check aborts with a diagnostic naming the rule and the pattern; the
  rule is skipped for the run. A broken pattern must never silently scan
  with the default.
- Version drift *between* endpoints of one app (half-migrated versioning)
  is the charter's R6xx-03, not this rule.
