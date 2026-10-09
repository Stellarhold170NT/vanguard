package spring

import (
	"strings"

	"github.com/Stellarhold170NT/vanguard/adapters/java"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// Language is the adapter's id — the registry key discovery uses. The
// tree-sitter pass and the surface mapping share it.
const Language = java.Language

// Adapter maps Spring Boot source into the final IR. It is bound to a scan
// root and parses paths relative to that root (charter §5.3). Detect/Parse
// wiring into the discovery registry lives in internal/discovery (the
// composition root — this package must not import it), where a thin
// wrapper assembles the Evidence this package's verdict produces.
type Adapter struct {
	root string
}

// New returns a Spring adapter bound to a scan root.
func New(root string) *Adapter { return &Adapter{root: root} }

// Language implements the naming half of the discovery.Adapter contract.
func (*Adapter) Language() string { return Language }

// DetectJavaFiles lists the walked files the Spring mapping consumes —
// the adapter claims the same .java set the tree-sitter pass parses.
func DetectJavaFiles(files []string) []string {
	var hits []string
	for _, f := range files {
		if strings.HasSuffix(f, ".java") {
			hits = append(hits, f)
		}
	}
	return hits
}

// ParseSurface builds the API surface best-effort (charter §5.3, the stub
// adapter's abort-free contract): parse diagnostics flow through, a file
// that cannot be read costs one diagnostic, and the finished surface is
// validated with ApiSurface.Validate so an adapter bug can never ship as
// silent IR corruption. Never returns an error, never aborts.
func (a *Adapter) ParseSurface(files []string) (*ir.ApiSurface, []ir.Diagnostic) {
	res, diags := java.New(a.root).Parse(files)
	m := &mapper{index: buildTypeIndex(res.Files), classes: buildClassIndex(res.Files)}

	surface := &ir.ApiSurface{Source: ir.Source{Lang: Language}}
	info := detectFramework(a.root, files, res)
	surface.Source.Framework = info.framework
	surface.Source.FrameworkVersion = info.version

	for _, f := range res.Files {
		var walk func(cls *java.Class)
		walk = func(cls *java.Class) {
			if classHasAny(cls, "RestController", "Controller") {
				if svc, ok := m.mapService(cls, &diags); ok {
					surface.Services = append(surface.Services, svc)
				}
			}
			if classHasAny(cls, "ControllerAdvice", "RestControllerAdvice") {
				surface.ErrorScheme.Handlers = append(surface.ErrorScheme.Handlers, m.mapAdvice(cls, &diags)...)
			}
			for _, n := range cls.Nested {
				walk(n)
			}
		}
		for _, c := range f.Types {
			walk(c)
		}
	}

	surface.GrpcServices = grpcServicesFrom(a.root, files)
	m.emitTypes(surface, res.Files)
	a.applyOverlay(surface, files, &diags)

	diags = append(diags, surface.Validate()...)
	surface.Diagnostics = diags
	return surface, diags
}

// buildClassIndex indexes every extracted declaration, nested included, by
// simple name — the exception-class lookup the R5xx-02 app-ownership
// signal reads (first declaration wins, mirroring buildTypeIndex).
func buildClassIndex(files []*java.File) map[string]*java.Class {
	idx := map[string]*java.Class{}
	var walk func(cls *java.Class)
	walk = func(cls *java.Class) {
		if _, seen := idx[cls.Name]; !seen {
			idx[cls.Name] = cls
		}
		for _, n := range cls.Nested {
			walk(n)
		}
	}
	for _, f := range files {
		for _, c := range f.Types {
			walk(c)
		}
	}
	return idx
}

// emitTypes extracts the DTO layer: every record/POJO-with-getters the API
// surface references (directly or through another included type) becomes
// an IR Type, in file × declaration order. The reference closure runs to a
// fixpoint so inclusion never depends on file order.
func (m *mapper) emitTypes(s *ir.ApiSurface, files []*java.File) {
	ref := map[string]bool{}
	var chase func(raw string)
	chase = func(raw string) {
		t := parseTypeUse(raw)
		name := simpleName(t.Base)
		if name == "" || name == "void" {
			return
		}
		ref[name] = true
		for _, a := range t.Args {
			chase(a)
		}
	}
	for si := range s.Services {
		for _, meth := range s.Services[si].Methods {
			for _, p := range meth.Params {
				chase(p.Type.Name)
			}
			if meth.Payload != nil {
				chase(meth.Payload.Name)
			}
			if meth.Response.Type.Name != "" {
				chase(meth.Response.Type.Name)
			}
		}
	}
	for _, h := range s.ErrorScheme.Handlers {
		if h.ResponseType != "" {
			chase(h.ResponseType)
		}
	}

	type candidate struct {
		cls *java.Class
		pkg string
	}
	var order []candidate
	byName := map[string]candidate{}
	var walk func(cls *java.Class, pkg string)
	walk = func(cls *java.Class, pkg string) {
		if isDTOKind(cls) {
			if _, seen := byName[cls.Name]; !seen {
				byName[cls.Name] = candidate{cls: cls, pkg: pkg}
				order = append(order, candidate{cls: cls, pkg: pkg})
			}
		}
		for _, n := range cls.Nested {
			walk(n, pkg)
		}
	}
	for _, f := range files {
		for _, c := range f.Types {
			walk(c, f.Package)
		}
	}

	emitted := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, cand := range order {
			name := cand.cls.Name
			if ref[name] && !emitted[name] {
				emitted[name] = true
				s.Types = append(s.Types, m.mapType(cand.cls, cand.pkg))
				for _, f := range cand.cls.Fields {
					chase(f.Type.Name)
				}
				changed = true
			}
		}
	}
}
