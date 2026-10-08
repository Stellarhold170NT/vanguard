package discovery

import (
	"fmt"
	"io"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// ScanOptions tunes one Scan run. The zero value is a complete, sane scan:
// published walk caps, the scan root's .vanguardignore honored, built-in
// adapters, no verbose output.
type ScanOptions struct {
	// Registry selects the adapter from; nil means NewBuiltinRegistry(root)
	// (the stub today, the java adapter from w3-01).
	Registry Registry

	// Walk overrides the walker bounds per WalkOptions' semantics (zero
	// fields keep the published defaults; IgnoreFileName "" disables the
	// ignore file). Nil means the Scan default: WalkOptions with
	// IgnoreFileName ".vanguardignore".
	Walk *WalkOptions

	// Verbose receives the evidence trail — skip records, language
	// candidates, every adapter verdict, the selection, parse counts and
	// diagnostics — when non-nil (charter §5.3: debug a wrong adapter pick
	// from the output alone). Nil: fully quiet.
	Verbose io.Writer
}

// ScanResult is one pipeline pass: walk → detect → select → parse. Every
// field is safe to read after Scan returns; nothing is shared state.
type ScanResult struct {
	Root       string
	Files      []string     // walked files, relative to Root, "/" separators
	Skipped    []SkipRecord // what the walker left out, and why
	Candidates []Candidate  // ranked language hypotheses (may be empty)

	Verdicts []DetectRecord // one per registered adapter, in order
	Selected string         // Adapter.Language() of the pick, "" = none

	Surface     *ir.ApiSurface  // nil when no adapter detected
	Diagnostics []ir.Diagnostic // parse diagnostics (best-effort contract)

	// NoAPI explains a clean "nothing found" outcome: no adapter detected
	// the file set, or the parsed surface holds no services/types/gRPC
	// services. It is NOT an error — charter §6.3 pins exit code 0 for this
	// ("repo không phát hiện API surface cũng là 0 — với thông báo rõ"), so
	// internal/cli (w2-06) prints this message and returns 0.
	NoAPI string
}

// Scan runs the full discovery pipeline against root.
//
// Error contract: only root-level walk failures (missing, unreadable or
// non-directory root) return an error — the caller maps those to the
// tool-error exit code (charter §6.3, code 2 territory). Everything else —
// skipped files, unparseable files, "nothing found" — is reported in the
// result and never as an error.
func Scan(root string, opts ScanOptions) (*ScanResult, error) {
	walkOpts := WalkOptions{IgnoreFileName: ".vanguardignore"}
	if opts.Walk != nil {
		walkOpts = *opts.Walk
	}
	walkRes, err := NewWalker().WalkWithStats(root, walkOpts)
	if err != nil {
		return nil, err
	}
	registry := opts.Registry
	if registry == nil {
		registry = NewBuiltinRegistry(root)
	}

	out := &ScanResult{
		Root:     root,
		Files:    walkRes.Files,
		Skipped:  walkRes.Skipped,
		Verdicts: []DetectRecord{},
	}
	v := opts.Verbose
	vprintf(v, "walk: %d file(s) under %s, %d skipped", len(out.Files), root, len(out.Skipped))
	for _, sk := range out.Skipped {
		vprintf(v, "skip %s: %s", sk.Path, sk.Reason)
	}

	out.Candidates = Detect(out.Files)
	for _, c := range out.Candidates {
		vprintf(v, "language candidate %s: %s", c.Lang, c.Evidence.Reason)
		for _, d := range c.Evidence.Details {
			vprintf(v, "  · %s", d)
		}
	}

	adapter, verdicts := SelectAdapter(registry, out.Files)
	out.Verdicts = verdicts
	for _, rec := range verdicts {
		vprintf(v, "adapter %s: detect=%v — %s", rec.Language, rec.Detected, rec.Evidence.Reason)
	}

	if adapter == nil {
		out.NoAPI = fmt.Sprintf("no adapter detected an API surface in %s (%d file(s) walked) — nothing to check; this is not an error (charter §6.3: exit 0)", root, len(out.Files))
		vprintf(v, "%s", out.NoAPI)
		return out, nil
	}
	out.Selected = adapter.Language()
	vprintf(v, "selected adapter: %s", out.Selected)

	surface, diags := adapter.Parse(out.Files)
	out.Surface, out.Diagnostics = surface, diags
	vprintf(v, "parse: %d service(s), %d type(s), %d gRPC service(s), %d diagnostic(s)",
		len(surface.Services), len(surface.Types), len(surface.GrpcServices), len(diags))
	for _, d := range diags {
		vprintf(v, "diagnostic %s: %s", d.Location.File, d.Message)
	}

	if len(surface.Services) == 0 && len(surface.Types) == 0 && len(surface.GrpcServices) == 0 {
		out.NoAPI = fmt.Sprintf("adapter %s found no services or types in %s — nothing to check; this is not an error (charter §6.3: exit 0)", adapter.Language(), root)
		vprintf(v, "%s", out.NoAPI)
	}
	return out, nil
}

// vprintf writes one verbose line. Every line carries the [vanguard] prefix
// so verbose output stays recognizable when mixed into other logs.
func vprintf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "[vanguard] "+format+"\n", args...)
}
