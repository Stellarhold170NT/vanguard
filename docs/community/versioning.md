---
title: Versioning & Releases
description: Vanguard's SemVer policy, the Keep a Changelog format, and the tag-triggered release pipeline with checksums and multi-platform binaries.
---

# Versioning & Releases

## Versioning

Vanguard follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **MAJOR** — breaking changes to the CLI surface, the `.vanguard.yaml`
  schema, the JSON/SARIF output contract, or rule id semantics.
- **MINOR** — new rules, new adapters (the Go/Python language matrix is
  wave 2), new flags; existing ids keep their meaning.
- **PATCH** — bug fixes, message wording, documentation.

The changelog follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
(`CHANGELOG.md` at the repository root) and is the source for release notes.

## Release Pipeline

Releases are tag-triggered (`v*`):

1. Tag a commit on `main` — e.g. `v0.1.0`.
2. The release workflow (`.github/workflows/release.yml`) cross-compiles
   static binaries with goreleaser through a pinned `zig cc` toolchain:
   **linux / darwin / windows × amd64 / arm64**.
3. Artifacts are published as `vanguard_<version>_<os>_<arch>.tar.gz`
   (`.zip` for windows) plus a `checksums.txt` with the sha256 of every
   archive.
4. A scratch-based Docker image (~9 MB, one static binary, no shell) is
   published to `ghcr.io/vanguard-lint/vanguard`.
5. The GitHub Release carries the changelog section for the tag.

Binaries are tag-stamped — `vanguard version` reports the version, commit,
and build date baked in at release time.

## Verifying a Release

```bash
# checksum verification (linux/amd64 shown)
curl -fsSL -o checksums.txt \
  "https://github.com/vanguard-lint/vanguard/releases/download/${VER}/checksums.txt"
grep "vanguard_${VER#v}_linux_amd64.tar.gz" checksums.txt | sha256sum -c -

# identity of a release binary
vanguard version
# → vanguard 0.1.0 (commit <release-commit-sha>, built <RFC3339-UTC-date>)
```

A locally built binary reports `0.1.0-dev (commit unknown, built unknown)` —
expected for dev builds; the full verification matrix is in
[Installation](/install).

## Current Release

- **[v0.1.0](https://github.com/vanguard-lint/vanguard/releases/tag/v0.1.0)**
  — first tagged release: scanner engine, 27 rules across R1xx–R6xx, CLI UX
  (pretty/JSON/SARIF), `.vanguard.yaml` config v1, goreleaser multi-platform
  packaging, Docker image. Scope is exactly the v0.1 charter; the Go/Python
  adapters and `--fix` are explicitly out of scope
  ([charter §1.4](/charter)).
