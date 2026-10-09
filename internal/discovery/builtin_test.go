package discovery

import (
	"testing"

	"github.com/Stellarhold170NT/vanguard/adapters/spring"
)

// Compile-time pin: the builtin wrapper satisfies the Adapter contract.
var _ Adapter = springSurfaceAdapter{}

func TestBuiltinRegistryOrder(t *testing.T) {
	r := NewBuiltinRegistry(".")
	adapters := r.Adapters()
	if len(adapters) != 2 {
		t.Fatalf("builtin adapters = %d, want 2", len(adapters))
	}
	if adapters[0].Language() != "java" || adapters[1].Language() != "stub" {
		t.Errorf("builtin order = [%s, %s], want [java, stub] (java first = priority)",
			adapters[0].Language(), adapters[1].Language())
	}
}

func TestBuiltinSelectsJava(t *testing.T) {
	adapter, records := SelectAdapter(NewBuiltinRegistry("."), []string{
		"pom.xml", "src/main/java/A.java",
	})
	if adapter == nil || adapter.Language() != "java" {
		t.Fatalf("selected = %v, want java", adapter)
	}
	if len(records) != 2 || !records[0].Selected || records[1].Selected {
		t.Errorf("records = %+v", records)
	}
}

func TestBuiltinSelectsStub(t *testing.T) {
	adapter, _ := SelectAdapter(NewBuiltinRegistry("."), []string{
		"api.stub.json", "README.md",
	})
	if adapter == nil || adapter.Language() != "stub" {
		t.Fatalf("selected = %v, want stub (no .java files)", adapter)
	}
}

// TestBuiltinJavaWinsOverStub pins the w3-02 priority: with both Detect
// criteria satisfied the java adapter (registered first) is selected.
func TestBuiltinJavaWinsOverStub(t *testing.T) {
	adapter, records := SelectAdapter(NewBuiltinRegistry("."), []string{
		"pom.xml", "src/A.java", "api/orders.stub.json",
	})
	if adapter == nil || adapter.Language() != "java" {
		t.Fatalf("selected = %v, want java (registered first)", adapter)
	}
	if len(records) != 2 || !records[0].Selected || records[1].Selected {
		t.Errorf("records = %+v, want java selected, stub not", records)
	}
}

// TestBuiltinSpringDetectEvidence checks the delegate's verdict flows into
// the Evidence trail verbose mode prints.
func TestBuiltinSpringDetectEvidence(t *testing.T) {
	ok, ev := springSurfaceAdapter{delegate: spring.New(".")}.Detect([]string{"a.java", "b.java"})
	if !ok || ev.Reason == "" || len(ev.Details) != 2 {
		t.Errorf("Detect = (%v, %+v)", ok, ev)
	}
	ok, _ = springSurfaceAdapter{delegate: spring.New(".")}.Detect([]string{"README.md"})
	if ok {
		t.Errorf("Detect(README.md) = true, want false")
	}
}
