package discovery

import (
	"errors"
	"fmt"

	"github.com/Stellarhold170NT/vanguard/adapters/spring"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
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

// springSurfaceAdapter adapts the spring adapter to the Adapter contract.
// The adapter package deliberately imports only adapters/java and
// internal/ir — never this package — so the Evidence assembly for its
// Detect verdict lives here, in the composition root.
type springSurfaceAdapter struct {
	delegate *spring.Adapter
}

func (w springSurfaceAdapter) Language() string { return w.delegate.Language() }

func (w springSurfaceAdapter) Detect(files []string) (bool, Evidence) {
	hits := spring.DetectJavaFiles(files)
	if len(hits) == 0 {
		return false, Evidence{Reason: "no .java file in the walked set"}
	}
	return true, Evidence{
		Reason:  fmt.Sprintf("%d .java file(s) present", len(hits)),
		Details: hits,
	}
}

func (w springSurfaceAdapter) Parse(files []string) (*ir.ApiSurface, []ir.Diagnostic) {
	return w.delegate.ParseSurface(files)
}

// NewBuiltinRegistry registers every adapter shipped in this build, bound
// to the scan root: the java adapter (Spring mapping, w3-02) first —
// registration order doubles as selection priority — and the stub second.
// The two Detect criteria are disjoint (.java vs *.stub.json) so the
// order is stable either way.
//
// Adapters are root-bound by design: Detect/Parse receive paths RELATIVE to
// the scan root (the charter §5.3 signature pins the methods, not the path
// convention), and an adapter that opens files needs the root to resolve
// them. See StubAdapter for the reference implementation of the convention.
func NewBuiltinRegistry(root string) *AdapterRegistry {
	r := NewRegistry()
	_ = r.Register(springSurfaceAdapter{delegate: spring.New(root)}) // cannot fail on a fresh registry
	_ = r.Register(NewStubAdapter(root))                             // cannot fail on a fresh registry
	return r
}
