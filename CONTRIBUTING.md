# Contributing to vanguard

Thanks for helping build Vanguard. This document covers the development setup
and — most importantly — the **process for adding a rule**, which is the main
way this repository grows.

## Development setup

- Go 1.22+ (`go.mod` pins the minimum).
- Build, test and vet locally — CI (`.github/workflows/ci.yml`) runs exactly
  these three:

  ```bash
  go build ./...
  go test ./...
  go vet ./...
  ```

- `reference/api-linter/` holds a local clone of
  [googleapis/api-linter](https://github.com/googleapis/api-linter) used as a
  design reference; it is git-ignored and never committed.

## Branch policy

- From W2 onwards, all work happens on branches or git worktrees (`.worktrees/`
  is git-ignored); `main` receives changes through PRs only.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
  (`feat:`, `fix:`, `docs:`, `ci:`, `chore:`, `refactor:`, `test:`).

## Process for adding a rule

Every rule goes through the same pipeline. A rule that skips a step is not
merged. The product contract is `docs/charter.md` §3 (taxonomy, id format,
severity discipline) and §3.8 (governance); the quality bar is
`docs/test-strategy.md`.

1. **Proposal** — open a GitHub issue with the `api-design` label using the
   *API design rule / finding* template. State the problem in the language of
   the source code (e.g. Spring annotation model), not in proto/OpenAPI
   terms. Reference the AIP that inspires the rule (`https://aip.dev/…`) —
   the rule is re-expressed for source, never copied 1:1.
2. **Charter fit** — the rule must map to one of the six families
   `R1xx..R6xx` and get an id of the form `R<F>xx-<NN>` (family digit +
   literal `xx` + two-digit sequence) plus a kebab-case slug, e.g.
   `R1xx-02 no-verb-path`. Rule ids and messages never contain organization
   names; organization-specific detection goes through configuration.
3. **Metadata** — add one metadata record under `rules/` (data file, embedded
   at build time): id, slug, category, default severity, AIP reference,
   summary, good/bad examples, doc path. Severity discipline: a check that
   relies on a heuristic and might be wrong defaults to `WARN` or `INFO`;
   only certain contract violations are `ERROR`. Only `ERROR` affects the
   exit code.
4. **Check** — implement the check as a pure function over the IR and
   register it in the registry. The engine assigns the rule id to findings;
   checks never set it themselves.
5. **Fixtures** — add adversarial cases under
   `testdata/adversarial/<rule-id>/`: at least 3 `vio-<slug>.java` and 2
   `ok-<slug>.java` cases, each ≤ 40 lines, simulating the pattern (never
   pasted proprietary code). See `docs/test-strategy.md` §3.
6. **Documentation** — add `docs/rules/<id>-<slug>.md` generated from the
   metadata; `vanguard explain <rule-id>` and `--list-rules` must agree with
   the docs.
7. **Quality gates** — the rule must survive the mutation harness (catch-rate
   target ≥ 90%) and the false-positive/negative audit before it leaves the
   `experimental` stability label.

Adding or removing a rule is a **minor** version bump; changing a core
interface (IR, adapter, check signature) is a **major** bump. Proposing rules
outside the charter process is not accepted — raise it in the issue tracker
instead.

## Reporting false positives

Open an issue with the `api-design` label, kind *False positive report*, and
include the finding location (`file:line:col`), the rule id, and a minimal
pattern description (not proprietary code). False positives are the top
priority of the quality process; suppressions
(`// vanguard:ignore <rule-id> <reason>`) are the immediate workaround.

## License

By contributing you agree that your contributions are licensed under the
Apache-2.0 license of this repository.
