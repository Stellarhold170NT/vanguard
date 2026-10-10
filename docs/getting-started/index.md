---
title: Quick Start
description: Get Vanguard running locally in minutes and produce your first API design report.
---

# Quick Start

Vanguard is a source-first API design linter for Java/Spring Boot: it scans
your source code directly, discovers the HTTP API surface, and flags design
drift at review time — before a spec even exists.

## Prerequisites

- **Nothing** for the Docker path (the image is a ~9 MB scratch image with a
  single static binary).
- **Go ≥ 1.22 and a C toolchain** only if you build from source (the
  tree-sitter grammar is cgo).

## 1. Install

::: code-group

```bash [Binary]
# Download, verify, and install the latest release binary (linux/amd64 shown)
VER=$(curl -fsSL https://api.github.com/repos/vanguard-lint/vanguard/releases/latest \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
curl -fsSL -o vanguard.tar.gz \
  "https://github.com/vanguard-lint/vanguard/releases/download/${VER}/vanguard_${VER#v}_linux_amd64.tar.gz"
tar -xzf vanguard.tar.gz vanguard && sudo install -m 0755 vanguard /usr/local/bin/vanguard
```

```bash [Docker]
docker pull ghcr.io/vanguard-lint/vanguard:latest
```

```bash [Source]
git clone https://github.com/vanguard-lint/vanguard && cd vanguard
go build -o vanguard ./cmd/vanguard
```

:::

Full instructions — including `checksums.txt` sha256 verification and
`go install` — live in [Installation](/install). Verify with:

```bash
vanguard version
# → vanguard 0.1.0 (commit <sha>, built <date>)
```

## 2. Run Your First Scan

```bash
vanguard scan /path/to/project
```

No build, no spec file, no file list — point it at a source tree and it
auto-detects the language, framework, and API surface. The pretty report
groups findings by rule and severity and carries a fix hint per finding.

Produce a machine-readable report instead:

```bash
vanguard scan /path/to/project --format json
vanguard scan /path/to/project --format sarif --output vanguard.sarif
```

## 3. Understand a Finding

Every finding cites its rule id (`R<F>xx-NN`, neutral by design). Resolve it
to full documentation — why it exists, what fires, what stays silent, and a
compliant Spring example:

```bash
vanguard explain R4xx-02
```

The same documentation is browsable in the [rule catalog](/rules/README).

## 4. Use It as a CI Gate

```bash
vanguard check /path/to/project
# → vanguard check: findings — 1 ERROR, 1 WARN, 0 INFO, 0 suppressed (exit 1)
```

The exit-code contract is the integration surface:

| Code | Meaning |
|---|---|
| `0` | Clean — no surviving ERROR finding (WARN/INFO may exist) |
| `1` | At least one **ERROR**-severity finding — fail the job |
| `2` | Tool or config error (bad `.vanguard.yaml`, bad usage) |

WARN and INFO findings never change the exit code. See
[CI Integration](/ci-integration) for ready-to-paste GitHub Actions and
GitLab CI jobs.

## 5. Configure (Optional)

Drop a `.vanguard.yaml` at the project root to tune behavior:

```yaml
version: 1
rules:
  R6xx-92: { disabled: true }    # demo-fixture rules are for testdata
  R1xx-02: { severity: WARN }    # downgrade repo-wide
suppressions:
  - rule: R3xx-02
    paths: ["**/generated/**"]
    reason: "generated DTOs — refactor planned"
```

The full schema — discovery, glob dialect, precedence, inline suppression —
is specified in [Configuration](/config). Generate a starter file with
`vanguard init`.

## Next Steps

- [Installation](/install) — all three install paths with checksum verification
- [Rule catalog](/rules/README) — all 27 rules with examples
- [Configuration](/config) — the complete `.vanguard.yaml` schema
- [Tuning & Suppression](/guides/tuning) — keep signal high without losing findings
- [CI Integration](/ci-integration) — gate workflows and SARIF upload
