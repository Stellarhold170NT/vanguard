# Vanguard Makefile — local entry points.
#
# `make ci` runs build → test → vet → lint. The GitHub Actions pipeline
# (.github/workflows/ci.yml) keeps the exact w1-06 pipeline (build → test →
# vet); the gofmt gate here is the local superset.

GO ?= go

.PHONY: build test vet lint ci clean

## build: compile all packages
build:
	@echo "== build =="
	$(GO) build ./...

## test: run all tests
test:
	@echo "== test =="
	$(GO) test ./...

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

## ci: full local pipeline (build → test → vet → lint)
ci: build test vet lint

## clean: clear the build cache for this module
clean:
	$(GO) clean ./...
