# Installing Vanguard

Vanguard ships as a single static binary. There are three supported install
paths — pick one, then verify it with `vanguard version` as shown in each
section. Platform matrix: linux / darwin / windows × amd64 / arm64.

Exit codes you will meet below (engine policy — `internal/engine/run.go`):
`0` = clean scan, warnings-only, or no API surface; `1` = at least one
**ERROR**-severity finding; `2` = tool/usage error (bad usage or config).
WARN/INFO findings do **not** fail the process.

---

## 1. Binary from GitHub Releases (recommended)

Assets exist from the first tagged release (`v0.1.0`, produced by the tag
triggered release workflow — release SOP: `reports/w6-01.md` §5). Asset names:
`vanguard_<version>_<os>_<arch>.tar.gz` (`.zip` for windows) plus a
`checksums.txt` with sha256 of every archive.

```bash
# Resolve the latest release tag (e.g. v0.1.0):
VER=$(curl -fsSL -m 30 https://api.github.com/repos/vanguard-lint/vanguard/releases/latest \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
echo "${VER:?no published release yet — use Docker or go install (below)}"

# Download the archive for your platform (this example: linux, amd64 —
# substitute darwin_arm64 / windows_amd64 / … as needed):
curl -fsSL -m 120 -o vanguard.tar.gz \
  "https://github.com/vanguard-lint/vanguard/releases/download/${VER}/vanguard_${VER#v}_linux_amd64.tar.gz"

# Verify the checksum before running anything:
curl -fsSL -m 60 -o checksums.txt \
  "https://github.com/vanguard-lint/vanguard/releases/download/${VER}/checksums.txt"
grep "vanguard_${VER#v}_linux_amd64.tar.gz" checksums.txt | sha256sum -c -

# Unpack and install (pick a prefix on your PATH):
tar -xzf vanguard.tar.gz vanguard
sudo install -m 0755 vanguard /usr/local/bin/vanguard
# or, without root:
#   mkdir -p ~/.local/bin && install -m 0755 vanguard ~/.local/bin/vanguard
#   (keep ~/.local/bin on your PATH)

vanguard version
```

**Verify** — one line, three fields (version = release tag without the `v`):

```
vanguard 0.1.0 (commit <release-commit-sha>, built <RFC3339-UTC-date>)
```

---

## 2. Docker

The image is `scratch`-based: it contains only the statically linked
`/vanguard` binary (no shell, no package manager, no source). It is published
from the first tagged release (w6-04, registry name follows the repo owner):

```bash
docker pull ghcr.io/vanguard-lint/vanguard:latest
docker run --rm ghcr.io/vanguard-lint/vanguard version
```

If the registry image is not published yet, build it locally from a checkout
(needs Docker; the multi-stage build pins its own toolchain, so the host only
needs Docker):

```bash
git clone https://github.com/vanguard-lint/vanguard && cd vanguard
docker build -t vanguard:local .

docker run --rm vanguard:local version

# Scan a repo: mount it at /src read-only, report goes to stdout.
# Exit 0 = clean / warnings-only / no API surface; 1 = at least one
# ERROR finding; 2 = tool/usage error.
docker run --rm -v "$PWD:/src:ro" vanguard:local scan /src

# Or use the bundled demo (mounts testdata/stub-repo and scans it):
docker compose -f docker-compose.yml run --rm scan
```

**Verify** — `docker run --rm vanguard:local version` prints one line. A
registry image built at tag time carries the tag identity; an image built
locally without `--build-arg` shows the dev identity (both are correct):

```
vanguard 0.1.0 (commit <release-commit-sha>, built <RFC3339-UTC>)   # tag-built image
vanguard 0.1.0-dev (commit unknown, built unknown)                  # local build, default args
```

(To stamp a local build like CI does: `docker build --build-arg
VERSION=0.1.0 --build-arg COMMIT=$(git rev-parse HEAD) --build-arg
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) -t vanguard:local .`)

The image is minimal by construction — verify it:
`docker run --rm --entrypoint /bin/sh vanguard:local` must fail (there is
no shell inside — verified: `exec: "/bin/sh": stat /bin/sh: no such file or
directory`), and `docker history --no-trunc vanguard:local` must show one
content layer only (`COPY /out/vanguard /vanguard`) next to two 0B metadata
entries (LABEL + ENTRYPOINT) — no source, testdata or credentials layers.

---

## 3. `go install`

Requires **Go ≥ 1.22** *and* a **C toolchain** (`cc`/`gcc`/`clang`). The C
compiler is not optional: the tree-sitter grammar Vanguard parses with is cgo
(charter B-1), and without it the install fails with
`build constraints exclude all Go files`. (Building with `CGO_ENABLED=0` is
never an option for this module.) Typical installs: Debian/Ubuntu
`sudo apt-get install build-essential`, Alpine `apk add gcc musl-dev`, macOS
`xcode-select --install`.

`@latest` resolves the newest **tagged** release; before the first tag, pin a
commit (the SHA below is verified and keeps working afterwards):

```bash
go install github.com/vanguard-lint/vanguard/cmd/vanguard@latest
# before the first tagged release, install from a commit instead:
go install github.com/vanguard-lint/vanguard/cmd/vanguard@cb2f269d0fedc5b75dc74a1ea3f1ffa71ad9cfd8

"$(go env GOPATH)/bin/vanguard" version
# optional: put the install bin on PATH for this shell
export PATH="$PATH:$(go env GOPATH)/bin"
```

**Verify** — same one-line `vanguard version` output (a `go install` without
ldflags reports `0.1.0-dev (commit unknown, built unknown)` — expected).

---

## Which path should I pick?

| Path | Best for | Needs |
|---|---|---|
| Binary (Releases) | CI pipelines, servers, everyday CLI use | curl + tar; pinned checksum verify included above |
| Docker | trying Vanguard without installing; hermetic CI step | Docker only (build) / registry access (pull) |
| `go install` | Go developers tracking the repo | Go ≥ 1.22 **and** a C toolchain |

## After installing: first scan

```bash
vanguard scan /path/to/repo          # pretty report to stdout
vanguard scan /path/to/repo --format json
vanguard check /path/to/repo         # CI alias, one summary line
vanguard explain R4xx-02             # rule documentation for a finding
```

A scan exits `1` only when an **ERROR**-severity finding exists; WARN/INFO
findings (as in the example above) still exit `0`, and `2` means the tool
itself failed (bad usage or config).
