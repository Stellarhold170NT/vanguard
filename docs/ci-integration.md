# vanguard in CI — gate integration guide

vanguard ships as a CI gate: the process exits with a code that fails the
job when the scanned tree carries at least one **ERROR**-severity API-design
finding. This document is the integration contract for target repositories
(GitHub Actions first, then internal runners) and was written by the GATE
CH3 task (w4-05) as part of the quality report `reports/w4-05.md`.

Every claim below reflects the implemented behaviour (charter §6.3, verified
end to end in w2-06); where a number is quoted it cites its W4 artifact.

---

## 1. Exit-code contract — the thing a gate keys on

| Exit | Meaning | CI semantics |
|---|---|---|
| **0** | Scan completed, **no ERROR finding**. WARN/INFO findings may be present; a repo with no API surface at all is also 0 (with a `nothing to check` notice). | **Pass.** |
| **1** | At least one finding with severity **ERROR** survived config and suppressions. | **Fail the job** — this is the gate. |
| **2** | Tool or config error: unreadable/invalid `.vanguard.yaml` (message carries `file:line`), unknown flag/rule, unreadable scan root. Findings are never printed on this path. | **Fail the job** (tool error — do not confuse with "your code violated a rule"). |

- `vanguard check <path>` is the CI alias of `scan`: same pipeline, same exit
  codes; on a non-TTY stdout it suppresses the pretty report and prints one
  summary line to stderr
  (`vanguard check: FAILED — 2 ERROR, 3 WARN, 0 INFO, 1 suppressed (exit 1)`).
- `scan --format json|sarif` keeps stdout purely machine-readable; with
  `--output <file>` the report goes to the file and stdout stays empty.
- WARN and INFO findings **never** change the exit code (charter §6.3 — the
  spectral behaviour). v0.1 fails fixed at ERROR; a `--fail-severity` knob is
  backlog.
- The recommended split: run **`check`** as the blocking step, and a second
  **`scan --format sarif`** invocation purely to produce the artifact. They
  share the same engine, so their verdicts cannot diverge.

## 2. GitHub Actions (target repository)

```yaml
name: api-design-gate
on:
  pull_request:
  push:
    branches: [main]

jobs:
  vanguard:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # v0.1: build from a PINNED commit (no release yet — w6-01 ships
      # goreleaser artifacts; switch to that download when it exists).
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go build -o /usr/local/bin/vanguard github.com/Stellarhold170NT/vanguard/cmd/vanguard@<PINNED-COMMIT>

      # The gate. exit 0 passes; exit 1 = ERROR findings (job fails);
      # exit 2 = tool/config error (job fails with a distinct cause).
      - name: vanguard check (gate)
        id: gate
        run: vanguard check . 

      # SARIF for the Security tab. The scan completed whenever a SARIF file
      # exists (exit 1 included), so upload even on gate failure; with exit 2
      # there is no report file and upload is skipped by the hash guard.
      - name: vanguard scan (SARIF)
        if: always()
        run: vanguard scan . --format sarif --output vanguard.sarif
        continue-on-error: true

      - uses: github/codeql-action/upload-sarif@v3
        if: always() && hashFiles('vanguard.sarif') != ''
        with:
          sarif_file: vanguard.sarif
```

Notes on the workflow:

- **`check` gates, `scan` reports.** One `check` invocation decides the job;
  the separate `scan --format sarif` produces the upload artifact. Running
  them as two steps keeps the exit-code semantics readable in the Actions UI.
- **`continue-on-error` on the SARIF step** is deliberate: the SARIF run must
  not mask the gate verdict — its only job is to emit the report file.
- **Findings identity is stable.** vanguard's SARIF carries
  `partialFingerprints` derived from (rule, file, line, column, message) and
  is byte-deterministic across runs (w4-03: 6/6 sha256-identical pairs,
  including the SARIF format on a 10k-file tree) — GitHub therefore tracks
  the same alert across pushes instead of reopening them.
- **Repo size budget.** p95 on a 10k-file / 353k-LOC Spring tree is 39.5 s on
  2 vCPU (w4-03 `testdata/perf-results.json`); cold start is ≤ 50 ms. Set the
  job timeout comfortably above the measured p95 for your tree size.

## 3. Internal CI (GitLab CI / shell runners)

Same contract, one job:

```yaml
vanguard-gate:
  stage: verify
  script:
    - vanguard check .                  # gate: exit 1 fails the job
  after_script:
    - vanguard scan . --format sarif --output vanguard.sarif || true
  artifacts:
    when: always
    paths: [vanguard.sarif]
    reports:
      sast: vanguard.sarif              # GitLab SAST report ingestion
```

Or the minimal shell-gate on any runner (all three codes handled):

```sh
vanguard check . --format json --output vanguard.json
code=$?
case $code in
  0) echo "gate: clean (WARN/INFO may exist — see vanguard.json)";;
  1) echo "gate: FAILED — ERROR findings (see vanguard.json)"; exit 1;;
  2) echo "gate: TOOL ERROR — config/invocation problem, not a violation"; exit 2;;
esac
```

- Treat exit **2** as an operations alert, not a code-quality signal: a red
  job because someone shipped a broken `.vanguard.yaml` is a config fix, not
  a design regression.
- Publish the JSON/SARIF artifacts regardless of the verdict; the §6.7 JSON
  summary (`summary.findings.{error,warning,info}` + `suppressed`) is what
  trend dashboards should ingest.

## 4. Suppression discipline (config v1)

Gate fatigue kills linters. vanguard's suppression model is file-scoped and
reason-carrying (charter §6.4.1, `docs/config.md`):

```yaml
version: 1
rules:
  R1xx-02: { severity: WARN }        # downgrade a noisy rule repo-wide
  R6xx-01: { disabled: true }        # policy rule, opt-in by design
suppressions:
  - rule: R3xx-02
    paths: ["**/generated/**"]
    reason: "generated DTOs — refactor planned for W5"
```

Every suppression carries a `reason`; the JSON/SARIF summary counts
suppressed findings, so a silent suppression wave is visible in the gate
output (`N suppressed` in the `check` one-liner). Prefer a recorded
suppression over disabling a rule family.

## 5. vanguard's own repository

The tool gates itself with the same pipeline: `make ci` runs build → tests →
golden snapshots → `go vet` → `gofmt` (see `Makefile`). The W4 quality
evidence that led to the GATE CH3 verdict — mutation catch-rate, perf and
determinism numbers, FP/FN audit, and the robustness suite
(`internal/robustness`) — lives in `reports/w4-05.md` and its cited
artifacts.
