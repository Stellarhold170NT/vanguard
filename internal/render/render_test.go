package render

import (
	"errors"
	"io"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
)

// stubRenderer is the smallest possible Renderer; it exists only to prove
// the render contract is implementable as declared. The real pretty/json/
// sarif renderers arrive with w2-05.
type stubRenderer struct{}

func (stubRenderer) Render(w io.Writer, report *engine.Report) error { return nil }

// TestRendererContractCompiles pins the interface w2-05 implements and the
// CLI (w2-06) consumes.
func TestRendererContractCompiles(t *testing.T) {
	var renderer Renderer = stubRenderer{}
	if err := renderer.Render(io.Discard, &engine.Report{}); err != nil {
		t.Fatalf("stub render: %v", err)
	}
}

// TestForFailsLoudlyUntilW205 documents that every format fails with the
// sentinel error instead of silently rendering nothing before w2-05.
func TestForFailsLoudlyUntilW205(t *testing.T) {
	for _, format := range []Format{FormatPretty, FormatJSON, FormatSARIF} {
		if _, err := For(format); !errors.Is(err, errNotImplemented) {
			t.Fatalf("For(%q) error = %v, want errNotImplemented", format, err)
		}
	}
}

// TestFormatConstantsAreDistinct guards the §6.2 flag values.
func TestFormatConstantsAreDistinct(t *testing.T) {
	formats := []Format{FormatPretty, FormatJSON, FormatSARIF}
	seen := make(map[Format]bool, len(formats))
	for _, f := range formats {
		if f == "" {
			t.Fatal("format constant must not be empty")
		}
		if seen[f] {
			t.Fatalf("duplicate format %q", f)
		}
		seen[f] = true
	}
}
