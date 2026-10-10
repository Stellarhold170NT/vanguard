package rules

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/vanguard-lint/vanguard/internal/engine"
)

// Registry is the production engine.Registry: it joins rule metadata
// (data) to check functions and guards registration with the §5.4
// validation catalog. Registration errors fail tests, never runtime — the
// CLI binary embeds an already-validated set (w3-03).
type Registry struct {
	byID   map[string]engine.Rule
	slugs  map[string]string // slug → owning rule id
	order  []string          // insertion order, kept for stable diagnostics
	sorted []engine.Rule     // All() cache, rebuilt on insertion
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: map[string]engine.Rule{}, slugs: map[string]string{}}
}

// Register validates and adds one rule. Duplicate ids and duplicate slugs
// are registration errors (§5.4 "unique, đúng họ").
func (r *Registry) Register(rule engine.Rule) error {
	if err := ValidateRule(rule); err != nil {
		return err
	}
	if _, dup := r.byID[rule.ID]; dup {
		return fmt.Errorf("duplicate rule id %s", rule.ID)
	}
	if owner, dup := r.slugs[rule.Slug]; dup {
		return fmt.Errorf("duplicate rule slug %q (already used by %s)", rule.Slug, owner)
	}
	r.byID[rule.ID] = rule
	r.slugs[rule.Slug] = rule.ID
	r.order = append(r.order, rule.ID)
	r.sorted = nil
	return nil
}

// ByID resolves one rule by exact id.
func (r *Registry) ByID(id string) (engine.Rule, bool) {
	rule, ok := r.byID[id]
	return rule, ok
}

// All lists rules sorted by id — deterministic iteration (charter B-6).
func (r *Registry) All() []engine.Rule {
	if r.sorted == nil {
		r.sorted = make([]engine.Rule, 0, len(r.order))
		for _, id := range r.order {
			r.sorted = append(r.sorted, r.byID[id])
		}
		sort.Slice(r.sorted, func(i, j int) bool { return r.sorted[i].ID < r.sorted[j].ID })
	}
	out := make([]engine.Rule, len(r.sorted))
	copy(out, r.sorted)
	return out
}

// ruleIDPattern pins §3.0: R<F>xx-NN, family digit 1..6, literal "xx",
// two-digit sequence. "R9xx-01" is outside the taxonomy; "R1xx-1" is not
// the format.
var ruleIDPattern = regexp.MustCompile(`^R[1-6]xx-[0-9]{2}$`)

// slugPattern pins the kebab-case discipline: lower-case words joined by
// single dashes.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateRule applies the §5.4 registry validation catalog to a joined
// engine.Rule (Compile calls it; Registry.Register re-calls it so a
// hand-built rule cannot skip the gate). Field names are quoted in errors
// so the failing catalog entry is obvious in test output.
func ValidateRule(r engine.Rule) error {
	if !ruleIDPattern.MatchString(r.ID) {
		return fmt.Errorf("invalid rule id %q: want R<F>xx-NN with family 1..6 (e.g. R1xx-02)", r.ID)
	}
	if !slugPattern.MatchString(r.Slug) {
		return fmt.Errorf("invalid rule slug %q: want lower-case kebab-case", r.Slug)
	}
	if r.Category == "" {
		return fmt.Errorf("rule %s: category is required", r.ID)
	}
	if _, ok := engine.ParseSeverity(string(r.Severity)); !ok {
		return fmt.Errorf("rule %s: invalid severity %q (want ERROR, WARN or INFO)", r.ID, string(r.Severity))
	}
	if r.Summary == "" {
		return fmt.Errorf("rule %s: summary is required", r.ID)
	}
	if r.DocPath == "" {
		return fmt.Errorf("rule %s: docPath is required", r.ID)
	}
	if r.ExampleGood == "" || r.ExampleBad == "" {
		return fmt.Errorf("rule %s: both exampleGood and exampleBad are required", r.ID)
	}
	seen := make(map[string]bool, len(r.Options))
	for _, o := range r.Options {
		if o == "" {
			return fmt.Errorf("rule %s: option names must be non-empty", r.ID)
		}
		if seen[o] {
			return fmt.Errorf("rule %s: duplicate option %q", r.ID, o)
		}
		seen[o] = true
	}
	if r.Selector == nil {
		return fmt.Errorf("rule %s: selector is required", r.ID)
	}
	if r.Check == nil {
		return fmt.Errorf("rule %s: check is required", r.ID)
	}
	return nil
}

// Compile joins a metadata record with its selector and check into an
// engine.Rule, converting the severity spelling to the canonical constant
// and running the full §5.4 validation catalog before returning. A rule
// that fails any check never reaches a registry.
func Compile(m Metadata, selector engine.Selector, check engine.CheckFunc) (engine.Rule, error) {
	sev, ok := engine.ParseSeverity(m.Severity)
	if !ok {
		return engine.Rule{}, fmt.Errorf("rule %s: invalid severity %q (want ERROR, WARN or INFO)", m.ID, m.Severity)
	}
	rule := engine.Rule{
		ID:              m.ID,
		Slug:            m.Slug,
		Category:        m.Category,
		Severity:        sev,
		DefaultDisabled: m.DefaultDisabled,
		Summary:         m.Summary,
		DocPath:         m.DocPath,
		ExampleGood:     m.ExampleGood,
		ExampleBad:      m.ExampleBad,
		Options:         append([]string(nil), m.Options...),
		Selector:        selector,
		Check:           check,
	}
	if err := ValidateRule(rule); err != nil {
		return engine.Rule{}, err
	}
	return rule, nil
}
