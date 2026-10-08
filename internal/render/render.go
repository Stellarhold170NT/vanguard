package render

import (
	"errors"
	"fmt"
	"io"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
)

// Format is one output format (charter §6.2 flags).
type Format string

const (
	FormatPretty Format = "pretty"
	FormatJSON   Format = "json"
	FormatSARIF  Format = "sarif"
)

// errNotImplemented marks formats whose renderer has not landed yet (w2-05).
var errNotImplemented = errors.New("renderer not implemented (w2-05)")

// Renderer writes one report in one format. Implementations must be
// deterministic — rendering the same report twice yields byte-identical
// output (charter §6.6, w4-03) — and write only findings + summary to w;
// tool errors go to stderr at the CLI layer (§6.3).
type Renderer interface {
	Render(w io.Writer, report *engine.Report) error
}

// For returns the renderer for format.
//
// TODO(w2-05): the pretty renderer (mock §6.6), the JSON schema-1 renderer
// (mock §6.7), and the SARIF 2.1.0 renderer (mock §6.8, schema-validated in
// CI). Until then every format fails loudly instead of silently rendering
// nothing.
func For(format Format) (Renderer, error) {
	return nil, fmt.Errorf("render: %q: %w", format, errNotImplemented)
}
