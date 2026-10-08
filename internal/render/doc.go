// Package render renders findings in the three output formats promised by
// the charter: pretty (human, grouped by rule and severity), JSON, and
// SARIF 2.1.0. Rendering is deterministic: the same findings must render
// byte-identically twice, and rule groups sort by severity then rule id
// (charter §6.6–6.8; implementation: w2-05).
package render
