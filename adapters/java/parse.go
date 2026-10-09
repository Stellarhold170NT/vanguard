package java

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// maxReadBytes bounds ONE source file read — the walker's published cap
// (internal/discovery's DefaultMaxFileSize, 1 MiB) mirrored locally: the
// builtin registry in internal/discovery wires the spring surface adapter
// that consumes this package (w3-02), so importing discovery here would
// cycle. The walker enforces the same cap independently; this is defense
// in depth.
const maxReadBytes = 1 << 20 // 1 MiB

// Language is the adapter's id — the registry key discovery uses and the
// verbose-mode tag once the adapter is wired into the pipeline (w3-02).
const Language = "java"

// javaFileExt is the only file kind the adapter parses; everything else in
// the walked set is ignored silently, mirroring the stub adapter.
const javaFileExt = ".java"

// Adapter extracts raw API candidates from Java sources with the
// tree-sitter-java grammar. It is bound to a scan root (New) and parses
// paths relative to that root, the same contract discovery hands every
// adapter (charter §5.3).
//
// The parser is reused across files and across Parse calls: the grammar is
// loaded once per adapter, and Parse is best-effort and abort-free — every
// per-file failure becomes an ir.Diagnostic and the remaining files keep
// scanning (charter §5.3). A single syntax error never costs the scan its
// other files.
type Adapter struct {
	root   string
	parser *sitter.Parser
}

// New returns a Java adapter bound to a scan root.
func New(root string) *Adapter {
	p := sitter.NewParser()
	p.SetLanguage(java.GetLanguage())
	return &Adapter{root: root, parser: p}
}

// Language implements the naming half of the discovery.Adapter contract
// (Detect/Parse wiring is w3-02, which owns the translation into the final
// IR).
func (*Adapter) Language() string { return Language }

// Parse extracts candidates from the given files, paths relative to the
// scan root, in input order. Non-.java paths are ignored silently; every
// failure — escaping path, unreadable, oversized, syntax error — becomes
// one diagnostic naming the file and never aborts the pass (the stub
// adapter's best-effort contract, kept for w3-01). A file with any syntax
// error is excluded whole: a half-parsed class could ship wrong candidates
// downstream, and tree-sitter's recovery positions are not stable enough
// to promise partial extraction in v0.1.
func (a *Adapter) Parse(files []string) (*Result, []ir.Diagnostic) {
	res := &Result{}
	var diags []ir.Diagnostic
	for _, rel := range files {
		if !strings.HasSuffix(rel, javaFileExt) {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
			diags = append(diags, a.diag(rel, fmt.Sprintf("path %q escapes the scan root — refused", rel)))
			continue
		}
		src, err := a.read(rel)
		if err != nil {
			diags = append(diags, a.diag(rel, err.Error()))
			continue
		}
		tree := a.parser.Parse(nil, src)
		root := tree.RootNode()
		if root.HasError() {
			diags = append(diags, a.syntaxDiag(rel, root, newSrcLines(src), src))
			tree.Close()
			continue
		}
		e := &extractor{src: newSrcLines(src), rel: rel}
		f := e.file(root)
		f.Path = rel
		res.Files = append(res.Files, f)
		tree.Close()
	}
	return res, diags
}

// read loads one source file, bounded by the read cap — the walker
// already refuses oversized files (w2-03), but Parse is independently
// callable, so the guard lives here too (defense in depth, same as the
// stub adapter).
func (a *Adapter) read(rel string) ([]byte, error) {
	full := filepath.Join(a.root, filepath.FromSlash(rel))
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rel, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxReadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rel, err)
	}
	if len(data) > maxReadBytes {
		return nil, fmt.Errorf("%s exceeds the %d-byte read cap — file skipped", rel, maxReadBytes)
	}
	return data, nil
}

// diag builds a file-level diagnostic (line 1, column 1) — the stub's
// convention for failures that have no meaningful source position.
func (a *Adapter) diag(rel, msg string) ir.Diagnostic {
	return ir.Diagnostic{Message: msg, Location: ir.Location{File: rel, Line: 1, Column: 1}}
}

// syntaxDiag reports why a file could not be extracted: the first
// tree-sitter error or missing node, named by file, line and UTF-16 column
// with a short verbatim snippet of the offending text.
func (a *Adapter) syntaxDiag(rel string, root *sitter.Node, lines *srcLines, src []byte) ir.Diagnostic {
	n := firstErrorNode(root)
	if n == nil {
		return a.diag(rel, "cannot fully parse file: syntax error (no error position reported) — file excluded from extraction")
	}
	loc := lines.loc(n)
	loc.File = rel
	what := "unexpected token"
	if n.IsMissing() {
		what = fmt.Sprintf("missing %q", n.Type())
	} else {
		what = "unexpected token " + quoteSnippet(n.Content(src))
	}
	return ir.Diagnostic{
		Message: fmt.Sprintf("cannot fully parse %s: syntax error at line %d, column %d (%s) — file excluded from extraction",
			rel, loc.Line, loc.Column, what),
		Location: loc,
	}
}

// quoteSnippet flattens a source snippet into one short quoted string.
func quoteSnippet(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) > 48 {
		text = string(runes[:48]) + "…"
	}
	return `"` + text + `"`
}

// firstErrorNode returns the first ERROR or MISSING node in source order,
// nil when the tree has none.
func firstErrorNode(n *sitter.Node) *sitter.Node {
	if n == nil {
		return nil
	}
	if n.IsError() || n.IsMissing() {
		return n
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		if hit := firstErrorNode(n.Child(i)); hit != nil {
			return hit
		}
	}
	return nil
}
