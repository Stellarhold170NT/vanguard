package ir

import (
	"encoding/json"
	"testing"
)

// stubNode is the smallest possible IR node. It exists only to prove that
// the Node interface is implementable exactly as declared; w2-02 replaces
// it with the real inventory nodes.
type stubNode struct{ loc Location }

func (n stubNode) Loc() Location { return n.loc }

// TestNodeInterfaceCompiles pins the engine input contract (charter §5.4):
// any node exposing Loc() satisfies ir.Node.
func TestNodeInterfaceCompiles(t *testing.T) {
	var node Node = stubNode{loc: Location{File: "src/X.java", Line: 1, Column: 2}}
	if node.Loc().File != "src/X.java" {
		t.Fatalf("unexpected location: %+v", node.Loc())
	}
}

// TestApiSurfaceMarshalIsStable keeps the golden-snapshot prerequisite true
// from day one (charter §5.2, w4-03 determinism): marshalling twice yields
// identical bytes, and the IR needs nothing beyond encoding/json — proof it
// stays pure Go.
func TestApiSurfaceMarshalIsStable(t *testing.T) {
	surface := ApiSurface{
		Source: Source{Lang: "java", Framework: "spring-boot", FrameworkVersion: "4.0.2"},
		Services: []Service{{
			Name:     "YouthResource",
			BasePath: "/api/military-youth",
			Methods: []Method{{
				OperationName: "autoComplete",
				Verb:          VerbGet,
				Path:          "/api/military-youth/auto-complete",
			}},
		}},
	}
	first, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal ApiSurface: %v", err)
	}
	second, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal ApiSurface again: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("ApiSurface JSON marshal is not deterministic")
	}
}

// TestVerbConstantsAreDistinct guards the §5.2 verb enumeration against
// accidental duplicates or empty values.
func TestVerbConstantsAreDistinct(t *testing.T) {
	verbs := []Verb{VerbGet, VerbPost, VerbPut, VerbPatch, VerbDelete, VerbRPC}
	seen := make(map[Verb]bool, len(verbs))
	for _, v := range verbs {
		if v == "" {
			t.Fatal("verb constant must not be empty")
		}
		if seen[v] {
			t.Fatalf("duplicate verb %q", v)
		}
		seen[v] = true
	}
}
