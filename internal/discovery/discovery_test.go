package discovery

import (
	"testing"
)

// TestAdapterAndRegistryContractsCompile pins the discovery surface that
// adapters/* (W3) code against, exercised through the production
// implementations (FSWalker, AdapterRegistry, StubAdapter).
func TestAdapterAndRegistryContractsCompile(t *testing.T) {
	var adapter Adapter = NewStubAdapter(t.TempDir())
	var registry Registry = NewRegistry()
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register stub adapter: %v", err)
	}
	if got := len(registry.Adapters()); got != 1 {
		t.Fatalf("registry holds %d adapters, want 1", got)
	}
	if adapter.Language() != StubLanguage {
		t.Fatalf("language %q, want %q", adapter.Language(), StubLanguage)
	}
	files := []string{"api/orders.stub.json"}
	ok, evidence := adapter.Detect(files)
	if !ok || evidence.Reason == "" {
		t.Fatalf("detect = %v, evidence %+v", ok, evidence)
	}
	var walker Walker = NewWalker()
	got, err := walker.Walk(t.TempDir(), WalkOptions{})
	if err != nil || len(got) != 0 {
		t.Fatalf("walk empty root = %v, %v", got, err)
	}
	surface, diags := adapter.Parse(files)
	if surface == nil {
		t.Fatal("parse returned nil surface")
	}
	_ = diags // parse reports per-file problems as diagnostics, never an error
}

// TestRegistryInterfaceAcceptsAnyImplementation keeps the Registry contract
// open exactly as w2-01 declared it: any Register/Adapters implementation —
// not only AdapterRegistry — drives SelectAdapter and Scan.
type minimalRegistry struct{ adapters []Adapter }

func (r *minimalRegistry) Register(a Adapter) error {
	r.adapters = append(r.adapters, a)
	return nil
}

func (r *minimalRegistry) Adapters() []Adapter { return r.adapters }

func TestRegistryInterfaceAcceptsAnyImplementation(t *testing.T) {
	reg := &minimalRegistry{}
	if err := reg.Register(&fakeAdapter{lang: "mini", detects: true}); err != nil {
		t.Fatal(err)
	}
	var registry Registry = reg
	chosen, records := SelectAdapter(registry, []string{"f.txt"})
	if chosen == nil || chosen.Language() != "mini" {
		t.Fatalf("chosen = %v, want mini", chosen)
	}
	if len(records) != 1 || !records[0].Selected {
		t.Fatalf("records = %+v", records)
	}
	// and Scan accepts it too, no AdapterRegistry required
	res, err := Scan(t.TempDir(), ScanOptions{Registry: registry})
	if err != nil {
		t.Fatal(err)
	}
	if res.Selected != "mini" || res.Surface == nil {
		t.Fatalf("scan result = selected %q, surface %+v", res.Selected, res.Surface)
	}
	if v := res.Surface.Validate(); len(v) != 0 {
		t.Errorf("empty surface must validate clean: %v", v)
	}
}
