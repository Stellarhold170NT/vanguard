// Package mutation is the W4 mutation harness (w4-02; test strategy §4): it
// takes the corpus's "ok" fixtures — code the linter correctly stays silent
// on — applies a mutator that turns them into real violations, and demands
// the target rule fire. Where the adversarial corpus (w4-01) measures
// agreement on hand-written cases, mutation measures how much of that
// agreement is *real*: a rule that only fires on the exact fixture it was
// written against scores low here, and the blind-spot report says where.
//
// Contract (test-strategy §4):
//
//   - 17 MUST mutators + 8 OPTIONAL mutators, each paired with the rule(s)
//     it must trap and the pool of ok-cases it feeds on (§4.1).
//   - Mutation case = (ok-case, mutator). Valid when the mutated file still
//     parses; syntax-broken mutants are dropped from the denominator but
//     counted (broken_count, brief w4-02 protocol).
//   - catch-rate(rule) = hits / valid over the cases targeting that rule;
//     catch-rate total = hits / valid over all cases. Gate: total >= 90%,
//     every rule with >= 1 valid case > 0% (§4.2).
//   - CI budget <= 5 min: per (rule, mutator) cell at most 3 ok-cases,
//     chosen deterministically — pool sorted by file path ascending, first
//     applicable (§4.4). No randomness anywhere; two runs, same numbers.
//
// Honesty protocol (brief w4-02 "Protocol & bẫy"):
//
//   - Pre-declared expected misses: rules that do not exist in the v0.1
//     registry (R3xx-04, R4xx-04, R5xx-04, R6xx-03 — w4-01 report §7.1)
//     cannot catch anything by construction. Their cases are declared
//     expected-miss BEFORE the run (derived from the registry fact, not from
//     any observed result) and a honored declaration is reported separately,
//     exactly like the corpus-check strict/declared pair. Both numbers are
//     always printed; the gate applies to the declared number.
//   - Results are never tuned after the fact: the JSON records every case
//     verdict, including misses, verbatim (testdata/mutation-results.json).
//
// The harness execs the real vanguard binary (VANGUARD_BIN or a fresh
// `go build`, the same fixture-binary contract as internal/golden) against
// one mutated file per temp dir — the per-case isolated scan shape of
// tools/corpus-check, mirrored here because a package-main tool cannot be
// imported. Mutated fixtures live in temp dirs and are never committed;
// only this package and the results JSON are.
package mutation
