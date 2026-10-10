# Dockerfile — Vanguard static scanner image (w6-02).
#
# Multi-stage build. The final stage is `scratch`: the image contains exactly
# one file, the statically linked `/vanguard` binary. This mirrors the release
# pipeline (.goreleaser.yaml, w6-01): the same zig-cc musl toolchain, the same
# trimpath/buildvcs flags and the same ldflags identity wiring, so a locally
# built image and a released binary come from the same recipe.
#
# Why zig cc: the tree-sitter grammar is cgo (charter B-1) — CGO_ENABLED=0
# fails with "build constraints exclude all Go files" — and `zig cc -target
# x86_64-linux-musl` cross-compiles it into a fully static musl ELF. A static
# binary is what makes `scratch` viable: there is no libc, no shell and no
# package manager in the final image, so a glibc-dynamic binary would not run.
#
# Tokens: nothing secret is ever passed to this build (no COPY of .git, no
# ARG with credentials) — see .dockerignore; the layer list is audited in
# reports/w6-02.md.

# ---------- stage 1: builder ----------
FROM golang:1.26-alpine AS builder

# zig is pinned to the same version the release pipeline uses (w6-01 report
# §3.1: 0.16.0, sha256 of the official x86_64-linux tarball from
# https://ziglang.org/download/index.json — verified again per build below).
ARG ZIG_VERSION=0.16.0
ARG ZIG_SHA256=70e49664a74374b48b51e6f3fdfbf437f6395d42509050588bd49abe52ba3d00

RUN apk add --no-cache ca-certificates tar xz \
 && wget -q -O /tmp/zig.tar.xz \
      "https://ziglang.org/download/${ZIG_VERSION}/zig-x86_64-linux-${ZIG_VERSION}.tar.xz" \
 && echo "${ZIG_SHA256}  /tmp/zig.tar.xz" | sha256sum -c - \
 && tar -xJf /tmp/zig.tar.xz -C /usr/local/lib \
 && ln -s "/usr/local/lib/zig-x86_64-linux-${ZIG_VERSION}/zig" /usr/local/bin/zig \
 && rm /tmp/zig.tar.xz \
 && zig version

WORKDIR /src

# Module files first so `go mod download` is cached independently of source.
COPY go.mod go.sum ./
RUN go mod download

# Only the packages the binary imports — the .dockerignore already kept
# testdata, docs, reports and VCS metadata out of the build context.
COPY cmd ./cmd
COPY internal ./internal
COPY rules ./rules
COPY adapters ./adapters

# Build identity follows the w2-06 contract, overridable at build time:
#   docker build --build-arg VERSION=v0.1.0 --build-arg COMMIT=$(git rev-parse HEAD) .
# Defaults degrade exactly like a plain `go build` (dev/unknown, never fails).
ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG DATE=unknown

# One explicit zig target — including for the "native" linux/amd64 — so the
# build never depends on the host C compiler (same stance as .goreleaser.yaml).
# The trailing `version` run is a canary: a non-static binary cannot execute
# on this musl-only builder, so the build fails here instead of in the image.
RUN CGO_ENABLED=1 CC="zig cc -target x86_64-linux-musl" \
    go build -trimpath -buildvcs=false \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
      -o /out/vanguard ./cmd/vanguard \
 && /out/vanguard version

# ---------- stage 2: final (scratch) ----------
FROM scratch

LABEL org.opencontainers.image.title="vanguard" \
      org.opencontainers.image.description="Static CLI that scans source trees and lints REST/gRPC API design against AIP-style rules" \
      org.opencontainers.image.source="https://github.com/vanguard-lint/vanguard" \
      org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /out/vanguard /vanguard

# No shell exists in this image: run `docker run IMAGE --help`, or override
# the entrypoint with another binary only if you mount one.
ENTRYPOINT ["/vanguard"]
