package discovery

import (
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// fakeAdapter is a configurable adapter for registry/selection tests.
type fakeAdapter struct {
	lang    string
	detects bool
	evid    Evidence
}

func (f *fakeAdapter) Language() string { return f.lang }

func (f *fakeAdapter) Detect([]string) (bool, Evidence) { return f.detects, f.evid }

func (f *fakeAdapter) Parse([]string) (*ir.ApiSurface, []ir.Diagnostic) {
	return &ir.ApiSurface{}, nil
}

// TestRegisterValidation pins what Register rejects: nil adapters, empty
// languages and duplicate languages are programming errors surfaced at
// registration time, with the offending language in the message.
func TestRegisterValidation(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Error("nil adapter must be rejected")
	}
	if err := r.Register(&fakeAdapter{lang: ""}); err == nil {
		t.Error("empty language must be rejected")
	}
	if err := r.Register(&fakeAdapter{lang: "a"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.Register(&fakeAdapter{lang: "a"})
	if err == nil || !strings.Contains(err.Error(), `"a"`) {
		t.Fatalf("duplicate register err = %v, want a message naming %q", err, "a")
	}
}

// TestAdaptersOrderAndCopy: registration order is preserved (selection
// priority, B-6) and the returned slice is a copy — callers cannot reorder
// or grow the registry through it.
func TestAdaptersOrderAndCopy(t *testing.T) {
	r := NewRegistry()
	for _, lang := range []string{"a", "b", "c"} {
		if err := r.Register(&fakeAdapter{lang: lang}); err != nil {
			t.Fatal(err)
		}
	}
	first := r.Adapters()
	if len(first) != 3 || first[0].Language() != "a" || first[2].Language() != "c" {
		t.Fatalf("adapters = %v", first)
	}
	first[0] = nil
	first = append(first, &fakeAdapter{lang: "intruder"})
	if got := r.Adapters(); len(got) != 3 || got[0].Language() != "a" {
		t.Fatalf("registry mutated through returned slice: %v", got)
	}
}

// TestSelectAsksAllAndPicksFirst: SelectAdapter consults EVERY adapter (so
// verbose mode can explain each verdict) and picks the first that detects —
// registration order is the priority.
func TestSelectAsksAllAndPicksFirst(t *testing.T) {
	r := NewRegistry()
	ev := func(reason string) Evidence { return Evidence{Reason: reason} }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(r.Register(&fakeAdapter{lang: "first", detects: true, evid: ev("first detects")}))
	must(r.Register(&fakeAdapter{lang: "second", detects: true, evid: ev("second detects too")}))
	must(r.Register(&fakeAdapter{lang: "third", detects: false, evid: ev("third declines")}))

	chosen, records := SelectAdapter(r, []string{"f.txt"})
	if chosen == nil || chosen.Language() != "first" {
		t.Fatalf("chosen = %v, want first", chosen)
	}
	if len(records) != 3 {
		t.Fatalf("records = %+v, want one per adapter", records)
	}
	if !records[0].Detected || !records[0].Selected {
		t.Errorf("first record = %+v, want detected+selected", records[0])
	}
	if !records[1].Detected || records[1].Selected {
		t.Errorf("second record = %+v, want detected but not selected", records[1])
	}
	if records[2].Detected || records[2].Selected {
		t.Errorf("third record = %+v, want neither", records[2])
	}
}

// TestSelectNoWinner: nothing detects → nil adapter, one declining record
// per adapter — the input to ScanResult.NoAPI.
func TestSelectNoWinner(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeAdapter{lang: "shy", detects: false}); err != nil {
		t.Fatal(err)
	}
	chosen, records := SelectAdapter(r, nil)
	if chosen != nil {
		t.Fatalf("chosen = %v, want nil", chosen)
	}
	if len(records) != 1 || records[0].Detected {
		t.Fatalf("records = %+v, want one declining record", records)
	}
}
