# Golden fixture set (w2-07)

The golden harness (`internal/golden`) runs the **real vanguard binary via
exec** — exactly how CI invokes it — against every case directory here, and
compares the normalized output of each format against the snapshot files in
the same directory. It is the tier-1 gate of the test strategy (§2.1) and
the shared harness for W3/W4: **a new case is a new directory, never a
harness edit.**

## Case layout

```
testdata/golden/<case-id>/
├── case.yaml          # manifest: target, args, exit, formats, snapshot, note
├── <fixture files>    # stub files, .vanguard.yaml, .vanguardignore, ...
├── snapshot.pretty    # normalized stdout, one file per format
├── snapshot.json
├── snapshot.sarif
└── snapshot.stderr    # only when the manifest asks for stderr
```

`case.yaml` schema (every field optional unless stated):

```yaml
target: "."            # scan target relative to the case dir; "{module}" = module root
args: []               # extra CLI args; "{case}"/"{module}" expand to module-relative paths
exit: 0                # expected exit code (charter §6.3): 0 clean, 1 ERROR findings, 2 tool error
formats: [pretty, json, sarif]   # default: all three
snapshot: stdout       # stdout (default) | stderr | both
note: >-               # what the case pins, for the reviewer
```

## Rules

1. **Snapshots are committed and reviewed.** A changed snapshot must appear
   as a diff in the PR — never regenerate to "make it pass".
2. **Regenerate intentionally** with `make golden-update` (runs
   `go test ./internal/golden/ -run TestGoldenCases -update`).
3. **Normalization covers only duration + absolute machine paths**
   (`internal/golden/normalize.go`). Any other byte difference is an
   engine/render determinism bug — fix the root cause (test strategy §5.4 #7).
4. **Exit code is part of the snapshot contract**: the manifest `exit` value
   is asserted for every format run.
5. Stub files are `.stub.json` documents in the IR's own vocabulary — see
   `internal/discovery/stub.go` for the schema. Keep them minimal; the case
   note explains what each fixture pins.

## Adding a case (W3/W4 recipe)

1. `mkdir testdata/golden/<case-id>/` — kebab-case id naming the engine
   surface it pins (e.g. `java-adapter-basic`).
2. Write `case.yaml` (target/args/exit/formats) + fixture files.
3. `make golden-update` → inspect the diff → commit fixture + snapshots
   together. The harness (`internal/golden`) must not change.

## Case index (W2 set)

| Case | Pins |
|---|---|
| `default-scan` | full default pipeline; ERROR→WARN→INFO grouping; footer counters; all 3 formats |
| `config-discovery` | `.vanguard.yaml` discovered by walking up from the scan root; severity override |
| `config-explicit` | `--config` beats discovery (decoy config in the case dir) |
| `suppression-config` | path-scoped suppression; suppressed count; exit code from surviving findings |
| `suppression-family` | family-prefix suppression (`rule: R6xx`); glob scoping |
| `severity-display-vs-exit` | `--severity` filters display only; exit code unchanged |
| `exit-codes-warn-only` | exit 0 with WARN/INFO findings |
| `exit-codes-error` | exit 1 on a single ERROR finding |
| `exit-codes-bad-config` | exit 2; stderr cites `file:line`; stdout stays empty |
| `no-api-surface` | no API surface → exit 0 + clear notice |
| `multi-format-consistency` | one scan, 3 formats; harness cross-checks json↔sarif finding sets |
| `rule-disable-config` | `rules.<id>.disabled` removes only that rule's findings |
| `sort-stability` | all findings tie on (file,line,col); pinned deterministic order |
| `detect-evidence` | `--verbose` evidence trail on stderr; `.vanguardignore` + `node_modules` skips; parse diagnostic, not abort |
| `integration-stub-repo` | production binary scans the committed `testdata/stub-repo` (CI-shaped wiring case) |
| `data-driven-proof` | adding rule R6xx-94 needed one YAML record + one table row — zero harness change (the proof case) |

The demo rules that make findings stub-reachable (R6xx-91/92/93/94, next
to the w2-04 R6xx-99) live in `rules/data/` with their checks in
`rules/demo_rules.go` — data-driven like every rule (charter §3.0).
