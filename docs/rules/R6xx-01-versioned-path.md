# R6xx-01 — versioned-path (WARN, default-OFF)

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
      versionPattern: "/v[0-9]+"   # the default
```

## Fires when

ALL of these hold on the same `ir.Service`:

1. the service declares a base path (`BasePath != ""`) — the class-level
   `@RequestMapping` the adapter merged; a controller without one has
   nothing to check at service level (method-level versioning is a future
   extension, not the charter's contract);
2. the effective pattern does **not** match the base path.

The default pattern `/v[0-9]+` is a segment-anchored substring: `/api/v1`
matches, `/api/rev1` does not, `/api/v10` does. The `versionPattern`
option replaces it with any Go regexp (anchor it with `^/api/v\d+` for an
explicit position).

## Suggestion

`add a version segment to the base path (e.g. /api/v1/…), or set the
R6xx-01 versionPattern option to your versioning scheme`

## Scope notes (attack surface — W4, read this)

- A **malconfigured pattern** (a regexp that does not compile) is loud:
  the check aborts with a diagnostic naming the rule and the pattern; the
  rule is skipped for the run. A broken pattern must never silently scan
  with the default.
- Version drift *between* endpoints of one app (half-migrated versioning)
  is the charter's R6xx-03, not this rule.
- gRPC services surface through `GrpcServices`, not `Service.BasePath` —
  proto package versioning is out of scope here.

## Good / Bad

```java
// good — clients pin the version
@RequestMapping("/api/v1/orders")

// bad — an unversioned base path
@RequestMapping("/api/orders")
```
