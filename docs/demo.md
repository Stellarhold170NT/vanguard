# vanguard demo — from bug to clean in 60 seconds

This walkthrough is the story the README promises: take a repo with an API
design bug, scan it, understand the finding, fix one line, and re-scan clean.
Every transcript below is **real output** captured from the w6-03 verification
run (sandbox `dind-sandbox:29`, binary built from `agent-w6-03@012fb5b +
docs`, `go build -o vanguard ./cmd/vanguard`).

You can replay the whole story yourself with the two commands at the end.

## 1. The buggy repo

`testdata/stub-repo` is a small fixture service whose payload DTO carries a
snake_case JSON field. Copy it to a scratch directory so the original stays
pristine:

```console
$ cp -r testdata/stub-repo /tmp/demo-shop
$ ls /tmp/demo-shop
README.md
node_modules
scratch
src
```

## 2. Scan — one finding, clearly explained

```console
$ vanguard scan /tmp/demo-shop
 vanguard 0.1.0-dev · /tmp/demo-shop · stub · stub-http
 1 services · 27 rules (27 active) · 3 files


 [WARN]  R4xx-02 field-casing (1)
 ──────────────────────────────────────────────────────────────────
 src/orders.stub.json:1:1
   Field "total_amount" is not lowerCamelCase — the payload JSON convention is lowerCamelCase.
   suggest: totalAmount
   suppress: // vanguard:ignore R4xx-02 <reason>


 ──────────────────────────────────────────────────────────────────
 1 findings · 0 ERROR · 1 WARN · 0 INFO · 0 suppressed
 1 files with findings · 3 scanned · 3 skipped · 2 parse diagnostics
 next: vanguard explain R4xx-02
```

Note three things:

- the finding points at the **file and position** (`src/orders.stub.json:1:1`),
- `suggest:` already contains the fix,
- and the exit code is **0** — this is a WARN, and only ERROR findings fail
  (exit 1). CI keeps passing; the finding still shows up in every scan.

![the scan-fix loop: one WARN finding, the fix, a clean rescan](images/demo-scan-fix-loop.png)

## 3. Explain — the rule behind the finding

```console
$ vanguard explain R4xx-02
R4xx-02 — DTO fields must serialize as lowerCamelCase — snake_case or PascalCase field names break the payload JSON convention.


Slug:     field-casing
Category: payload
Severity: WARN
Docs:     docs/rules/R4xx-02-field-casing.md


Good:
  public record BookDto(
        Long id,
        String authorName) { ... }


Bad:
  public record BookDto(
        Long id,
        @JsonProperty("author_name") String authorName) { ... }
```

## 4. Fix — one line

The DTO maps `total_amount` as the wire name. Rename it to `totalAmount`:

```console
$ sed -i 's/"jsonName": "total_amount"/"jsonName": "totalAmount"/' /tmp/demo-shop/src/orders.stub.json
$ grep totalAmount /tmp/demo-shop/src/orders.stub.json
      {"name": "totalAmount", "jsonName": "totalAmount", "type": "int64"}]},
```

(On the real fixture this is the `@JsonProperty("total_amount")` annotation on
a DTO field; the stub fixture stores the same decision in `jsonName`.)

## 5. Re-scan — clean

```console
$ vanguard scan /tmp/demo-shop
 vanguard 0.1.0-dev · /tmp/demo-shop · stub · stub-http
 1 services · 27 rules (27 active) · 3 files


 ──────────────────────────────────────────────────────────────────
 0 findings · 0 ERROR · 0 WARN · 0 INFO · 0 suppressed
 0 files with findings · 3 scanned · 3 skipped · 2 parse diagnostics
```

## 6. Check — the CI view

`vanguard check` is the CI-facing alias of scan: same detection, one line on
stderr, exit code as the contract.

```console
$ vanguard check /tmp/demo-shop
vanguard check: clean — 0 ERROR, 0 WARN, 0 INFO, 0 suppressed (exit 0)
```

If the bug had been an **ERROR**-severity finding, this line would read
`findings — 1 ERROR, …` and the process would exit **1**, failing the job.

## What a real repository looks like

The stub demo is intentionally tiny. The W5 baseline run against a real
774-file Spring Boot backend (24 controllers auto-detected) shows what a first
`vanguard scan` on an un-reviewed API surface looks like — 244 findings across
27 rules, grouped by rule with the worst severity first:

![vanguard scan of the military-youth backend: 753 files, 244 findings](images/demo-terminal-military-youth.png)

The same run as SARIF 2.1.0, ready for GitHub Code Scanning — every alert gets
a stable fingerprint (rule, file, line, column, message) so GitHub tracks it
across pushes instead of reopening it:

![SARIF 2.1.0 output of the same scan in an editor view](images/demo-sarif-editor.png)

## Replay it

```console
$ git clone https://github.com/Stellarhold170NT/vanguard && cd vanguard
$ go build -o vanguard ./cmd/vanguard
$ cp -r testdata/stub-repo /tmp/demo-shop
$ ./vanguard scan /tmp/demo-shop          # 1 WARN — R4xx-02 field-casing
$ ./vanguard explain R4xx-02              # why, with a Good/Bad example
$ ./vanguard check /tmp/demo-shop && echo "CI is green either way"
```
