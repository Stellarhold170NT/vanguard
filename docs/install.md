# Installing Vanguard

Vanguard ships as a single static binary. There are three supported install
paths — pick one, then verify it with `vanguard version` as shown in each
section. Platform matrix: linux / darwin / windows × amd64 / arm64.

Exit codes you will meet below: `0` = clean scan or no API surface, `1` =
findings (success for a linter — violations were found), `2` = tool/usage
error.

---

## 1. Binary from GitHub Releases (recommended)

Assets exist from the first tagged release (`v0.1.0`, produced by the tag
triggered release workflow — release SOP: `reports/w6-01.md` §5). Asset names:
`vanguard_<version>_<os>_<arch>.tar.gz` (`.zip` for windows) plus a
`checksums.txt` with sha256 of every archive.

```bash
# Resolve the latest release tag (e.g. v0.1.0):
VER=$(curl -fsSL -m 30 https://api.github.com/repos/Stellarhold170NT/vanguard/releases/latest \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
echo "${VER:?no published release yet — use Docker or go install (below)}"

# Download the archive for your platform (this example: linux, amd64 —
# substitute darwin_arm64 / windows_amd64 / … as needed):
curl -fsSL -m 120 -o vanguard.tar.gz \
  "https://github.com/Stellarhold170NT/vanguard/releases/download/${VER}/vanguard_${VER#v}_linux_amd64.tar.gz"

# Verify the checksum before running anything:
curl -fsSL -m 60 -o checksums.txt \
  "https://github.com/Stellarhold170NT/vanguard/releases/download/${VER}/checksums.txt"
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
docker pull ghcr.io/stellarhold170nt/vanguard:latest
docker run --rm ghcr.io/stellarhold170nt/vanguard version
```

If the registry image is not published yet, build it locally from a checkout
(needs Docker; the multi-stage build pins its own toolchain, so the host only
needs Docker):

```bash
git clone https://github.com/Stellarhold170NT/vanguard && cd vanguard
docker build -t vanguard:local .

docker run --rm vanguard:local version

# Scan a repo: mount it at /src read-only, report goes to stdout.
# Exit code 1 means findings were found.
docker run --rm -v "$PWD:/src:ro" vanguard:local scan /src

# Or use the bundled demo (mounts testdata/stub-repo and scans it):
docker compose -f docker-compose.yml run --rm scan
```

**Verify** — `version` prints the same one-line identity as way 1, and the
image is minimal: `docker run --rm --entrypoint /bin/sh vanguard:local` must
fail (there is no shell inside), and `docker history --no-trunc
vanguard:local` must show no layer carrying source, testdata or credentials.

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
go install github.com/Stellarhold170NT/vanguard/cmd/vanguard@latest
# before the first tagged release, install from a commit instead:
go install github.com/Stellarhold170NT/vanguard/cmd/vanguard@cb2f269d0fedc5b75dc74a1ea3f1ffa71ad9cfd8

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

A scan that prints findings and exits `1` is Vanguard working as intended.
