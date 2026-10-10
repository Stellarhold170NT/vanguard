---
layout: home

hero:
  name: Vanguard
  text: Source-first API design linter for Java/Spring Boot
  tagline: Scan your source directly — no OpenAPI spec, no build. Vanguard discovers the REST + gRPC API surface and flags design drift with precise file:line:col findings before a spec even exists.
  actions:
    - theme: brand
      text: Quick Start
      link: /getting-started/
    - theme: alt
      text: Rules
      link: /rules/README
    - theme: alt
      text: CLI
      link: /reference/cli

features:
  - title: Source-First Scanning
    details: tree-sitter parsing of Java/Spring Boot (MVC + WebFlux, springdoc overlay) and .proto files. Auto-detects the API surface — no spec file, no build, no file list.
  - title: 27 Rules Across 6 Families
    details: Resource naming, HTTP methods & semantics, pagination & collections, payload, errors & status, versioning & gRPC — each rule traceable to an AIP guideline.
  - title: CI-Native by Contract
    details: Pretty, JSON, and SARIF 2.1.0 output; strict 0/1/2 exit codes; byte-deterministic reports with stable partialFingerprints for GitHub alerts.
  - title: Tunable, Not Turned Off
    details: .vanguard.yaml severity overrides, path-scoped suppressions, and inline // vanguard:ignore — every suppression carries a reason, so gate fatigue stays visible.
---

## Typical Scenarios

<div class="scenario-grid">
  <a class="scenario-card" href="/ci-integration">
    <h3>Gate Pull Requests</h3>
    <p>Fail the job only on ERROR-severity findings; publish SARIF to the Security tab with stable alert identity.</p>
  </a>
  <a class="scenario-card" href="/guides/case-study">
    <h3>Audit an Existing Service</h3>
    <p>Scan a production-shaped Spring service and get a triaged, severity-ranked remediation backlog.</p>
  </a>
  <a class="scenario-card" href="/guides/tuning">
    <h3>Tune Noisy Rules</h3>
    <p>Downgrade, disable, or suppress per path — with reasons recorded in the report instead of lost in config.</p>
  </a>
  <a class="scenario-card" href="/rules/README">
    <h3>Explain Any Rule</h3>
    <p>Every rule id resolves to documentation with compliant and violating Spring examples via <code>vanguard explain</code>.</p>
  </a>
</div>

## Quick Install

::: code-group

```bash [Binary]
VER=$(curl -fsSL https://api.github.com/repos/Stellarhold170NT/vanguard/releases/latest \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
curl -fsSL -o vanguard.tar.gz \
  "https://github.com/Stellarhold170NT/vanguard/releases/download/${VER}/vanguard_${VER#v}_linux_amd64.tar.gz"
tar -xzf vanguard.tar.gz vanguard && sudo install -m 0755 vanguard /usr/local/bin/vanguard
vanguard version
```

```bash [Docker]
docker pull ghcr.io/stellarhold170nt/vanguard:latest
docker run --rm -v "$PWD:/src:ro" ghcr.io/stellarhold170nt/vanguard:latest scan /src
```

```bash [go install]
go install github.com/Stellarhold170NT/vanguard/cmd/vanguard@latest
"$(go env GOPATH)/bin/vanguard" version
```

:::

See [Installation](/install) for the full setup guide, including checksum verification and the platform matrix (linux / darwin / windows × amd64 / arm64).
