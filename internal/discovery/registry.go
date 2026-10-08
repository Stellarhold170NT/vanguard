package discovery

import (
	"errors"
	"fmt"
)

// AdapterRegistry is the production Registry (satisfies the w2-01 Registry
// contract): adapters keyed by Adapter.Language(), iterated in registration
// order. The order is deliberate — it doubles as selection priority in
// SelectAdapter and keeps every iteration deterministic (charter B-6).
type AdapterRegistry struct {
	byLang map[string]Adapter
	order  []Adapter
}

// Compile-time pin: AdapterRegistry satisfies the w2-01 Registry contract.
var _ Registry = (*AdapterRegistry)(nil)

// NewRegistry returns an empty registry.
func NewRegistry() *AdapterRegistry {
	return &AdapterRegistry{byLang: make(map[string]Adapter)}
}

// Register adds an adapter. It rejects nil adapters, empty languages and
// duplicate languages: misregistration is a programming error and is
// surfaced here, at registration time — never at scan time.
func (r *AdapterRegistry) Register(a Adapter) error {
	if a == nil {
		return errors.New("discovery: cannot register a nil adapter")
	}
	lang := a.Language()
	if lang == "" {
		return errors.New("discovery: adapter language must not be empty")
	}
	if _, dup := r.byLang[lang]; dup {
		return fmt.Errorf("discovery: adapter %q is already registered", lang)
	}
	r.byLang[lang] = a
	r.order = append(r.order, a)
	return nil
}

// Adapters returns the registered adapters in registration order. The
// slice is a copy: callers cannot reorder or grow the registry through it.
func (r *AdapterRegistry) Adapters() []Adapter {
	out := make([]Adapter, len(r.order))
	copy(out, r.order)
	return out
}

// DetectRecord is one adapter's verdict during SelectAdapter — the raw
// Detect answer plus whether the scan actually picked it. Both fields exist
// because "detected but not selected" (a later adapter also said yes) and
// "not detected" are different stories, and verbose mode tells them apart.
type DetectRecord struct {
	Language string
	Detected bool // raw Detect verdict
	Selected bool // true only for the one adapter the scan picked
	Evidence Evidence
}

// SelectAdapter asks EVERY registered adapter — in registration order, so
// priority is explicit and deterministic (charter B-6) — and picks the
// first whose Detect accepts the file set. Every verdict comes back in
// records: charter §5.3 requires the evidence trail so a sparse or wrong
// selection is debuggable from verbose output alone.
//
// It accepts the w2-01 Registry interface, so any Register/Adapters
// implementation works — not only AdapterRegistry.
func SelectAdapter(reg Registry, files []string) (Adapter, []DetectRecord) {
	var records []DetectRecord
	var chosen Adapter
	for _, a := range reg.Adapters() {
		ok, ev := a.Detect(files)
		records = append(records, DetectRecord{
			Language: a.Language(),
			Detected: ok,
			Selected: ok && chosen == nil,
			Evidence: ev,
		})
		if ok && chosen == nil {
			chosen = a
		}
	}
	return chosen, records
}

// NewBuiltinRegistry registers every adapter shipped in this build, bound
// to the scan root. v0.1 ships the stub only; the java adapter joins ahead
// of it from w3-01 (order = priority; the two Detect criteria are disjoint
// — *.stub.json vs java sources — so the order is stable either way).
//
// Adapters are root-bound by design: Detect/Parse receive paths RELATIVE to
// the scan root (the charter §5.3 signature pins the methods, not the path
// convention), and an adapter that opens files needs the root to resolve
// them. See StubAdapter for the reference implementation of the convention.
func NewBuiltinRegistry(root string) *AdapterRegistry {
	r := NewRegistry()
	_ = r.Register(NewStubAdapter(root)) // cannot fail on a fresh registry
	return r
}
