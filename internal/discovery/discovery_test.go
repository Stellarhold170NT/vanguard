package discovery

import (
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// stubAdapter is the smallest possible Adapter; it exists only to prove the
// charter §5.3 interface is implementable as declared. The real java
// adapter arrives with w3-01/w3-02.
type stubAdapter struct{}

func (stubAdapter) Language() string { return "stub" }

func (stubAdapter) Detect(files []string) (bool, Evidence) {
	return len(files) > 0, Evidence{Reason: "stub always detects non-empty trees"}
}

func (stubAdapter) Parse(files []string) (*ir.ApiSurface, []ir.Diagnostic) {
	return &ir.ApiSurface{}, nil
}

// stubRegistry is the smallest possible Registry (same purpose as
// stubAdapter).
type stubRegistry struct{ adapters []Adapter }

func (r *stubRegistry) Register(a Adapter) error {
	r.adapters = append(r.adapters, a)
	return nil
}

func (r *stubRegistry) Adapters() []Adapter { return r.adapters }

// TestAdapterAndRegistryContractsCompile pins the discovery surface that
// w2-03 implements and adapters/* (W3) codes against.
func TestAdapterAndRegistryContractsCompile(t *testing.T) {
	var adapter Adapter = stubAdapter{}
	var registry Registry = &stubRegistry{}
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register stub adapter: %v", err)
	}
	if got := len(registry.Adapters()); got != 1 {
		t.Fatalf("registry holds %d adapters, want 1", got)
	}
	if adapter.Language() != "stub" {
		t.Fatalf("language %q, want %q", adapter.Language(), "stub")
	}
	files := []string{"pom.xml", "src/X.java"}
	ok, evidence := adapter.Detect(files)
	if !ok || evidence.Reason == "" {
		t.Fatalf("detect = %v, evidence %+v", ok, evidence)
	}
	surface, diags := adapter.Parse(files)
	if surface == nil || len(diags) != 0 {
		t.Fatalf("parse = %+v, diags %v", surface, diags)
	}
}

// TestDetectStubIsSafeToCall keeps the pipeline executable before w2-03:
// the stub returns no candidates instead of panicking.
func TestDetectStubIsSafeToCall(t *testing.T) {
	if got := Detect([]string{"pom.xml", "src/X.java"}); len(got) != 0 {
		t.Fatalf("Detect stub returned %d candidates, want 0", len(got))
	}
}
