# Vanguard Makefile — local entry points.
#
# `make ci` runs build → test → vet → lint. The GitHub Actions pipeline
# (.github/workflows/ci.yml) keeps the exact w1-06 pipeline (build → test →
# vet); the gofmt gate here is the local superset.

GO ?= go

# Build identity wired into the binary via -ldflags (w2-06): `make bin`
# produces ./bin/vanguard whose `version` reports these values; plain
# `go build` keeps the dev/unknown fallback (never fails the build).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
LDFLAGS ?= -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY:build test vet lint ci bin golden golden-update mutation clean build test vet lint ci bin golden golden-update docs-check clean
## build: compile all packages
build:
	@echo "== build =="
	$(GO) build ./...

## bin: build ./bin/vanguard with the version metadata wired
bin:
	@echo "== bin (version=$(VERSION), commit=$(COMMIT)) =="
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/vanguard ./cmd/vanguard

## test: run all tests
test:
	@echo "== test =="
	$(GO) test ./...

## golden: run the golden fixture harness (w2-07) — the real binary via
## exec against testdata/golden/<case>/ with snapshot comparison
golden:
	@echo "== golden (fixtures + integration) =="
	$(GO) test ./internal/golden/...

## golden-update: intentionally regenerate the golden snapshots. The diff
## this produces must appear in the reviewing PR — that is the
## intentional-change contract of the test strategy (tier 2.1).
golden-update:
	@echo "== golden-update (intentional snapshot rewrite) =="
	$(GO) test ./internal/golden/ -run TestGoldenCases -update

## mutation: run the mutation harness (w4-02) — the real binary against
## mutated corpus ok-cases, gate on declared catch-rate >= 90%, and write
## testdata/mutation-results.json. Budget: <= 5 min (test-strategy §4.4);
## the harness itself enforces a 4-minute deadline.
mutation:
	@echo "== mutation (w4-02 catch-rate harness) =="
	$(GO) test ./internal/mutation/ -run TestMutationHarness -count=1 -v -timeout 9m \
		-args -write-results -out testdata/mutation-results.json
## docs-check: the w3-07 docs cross-check (brief deliverable 4) — the
## --list-rules catalog must match docs/rules/ 1-1. Adding a rule without
## its reference page fails the build; the same check also runs inside
## `make test` (internal/cli TestRuleDocsMatchCatalog).
docs-check:
	@echo "== docs-check (list-rules vs docs/rules 1-1) =="
	$(GO) test ./internal/cli/ -run 'TestRuleDocs' -count=1
## vet: go vet over all packages
vet:
	@echo "== vet =="
	$(GO) vet ./...

## lint: gofmt gate over the tracked Go files (fallback: find)
lint:
	@echo "== lint (gofmt) =="
	@files="$$(git ls-files '*.go' 2>/dev/null)"; \
	if [ -z "$$files" ]; then \
		files="$$(find . -type f -name '*.go' -not -path './.git/*' -not -path './reference/*')"; \
	fi; \
	if [ -z "$$files" ]; then \
		echo "lint: no Go files found"; \
		exit 0; \
	fi; \
	unformatted="$$(gofmt -l $$files)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi; \
	echo "gofmt: clean"

## ci: full local pipeline (build → test → vet → lint → docs-check). `test`
## includes the golden fixture harness and its integration case
## (internal/golden); the explicit `golden` step makes that gate visible in
## the log, `docs-check` makes the w3-07 list-rules↔docs 1-1 gate visible.
ci: build test golden vet lint docs-check

## clean: clear the build cache for this module
clean:
	$(GO) clean ./...
