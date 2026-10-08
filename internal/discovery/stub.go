package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// StubLanguage is the dummy language id of the stub adapter — the registry
// key and the verbose-mode tag until the real java adapter lands (w3-01).
const StubLanguage = "stub"

// StubFramework is the default framework id of a stub parse; a stub file
// may override it per service via its "framework" field.
const StubFramework = "stub"

// stubFileExt is the only file kind the stub adapter claims and parses.
const stubFileExt = ".stub.json"

// StubMaxFileBytes bounds ONE stub file read. The walker already skips
// files above WalkOptions.MaxFileSize, but Parse is independently callable
// (unit tests, other tools), so the brief's "never load an oversized file
// into memory" guard lives at this layer too — defense in depth.
const StubMaxFileBytes = DefaultMaxFileSize

// StubAdapter is the throwaway adapter that makes the whole pipeline —
// walk → detect → select → parse → IR — runnable end-to-end before the real
// java adapter arrives (w3-01). It claims *.stub.json files: small JSON
// documents describing an API surface in the IR's own vocabulary. It is
// also the reference implementation W3 adapters copy for structure.
//
// Behavior contract (every line tested):
//   - Detect: true iff ≥1 walked file ends in .stub.json; the evidence
//     lists them.
//   - Parse: per file — read (bounded by StubMaxFileBytes), decode, convert
//     to one ir.Service; every failure (unreadable, oversized, malformed
//     JSON, missing service name, unknown verb/param-in/pagination/type
//     kind) becomes an ir.Diagnostic and the run continues — never an
//     error, never an abort (charter §5.3, the best-effort contract).
//   - A method or type that cannot be translated exactly is dropped whole
//     with a diagnostic: a half-translated method could violate the IR
//     invariants (w2-02) and make rules read nonsense.
//   - Files that are not *.stub.json are ignored silently (Detect never
//     claims them; Parse defends anyway).
//   - The finished surface is checked with ApiSurface.Validate (w2-02's
//     recommendation for adapters) and any violation is merged into the
//     returned diagnostics — a stub bug can never ship as silent IR
//     corruption.
//   - Locations are file-level (line 1, column 1): the stub describes an
//     API, it does not point into source lines — real adapters own
//     source-accurate locations (charter §5.2).
//   - Paths are relative to the scan root; the adapter resolves them
//     against the root it was constructed with (NewStubAdapter) and refuses
//     anything escaping it (".." or absolute) before opening.
type StubAdapter struct {
	root string // scan root; Detect/Parse paths are relative to it
}

// NewStubAdapter returns the stub adapter bound to a scan root.
func NewStubAdapter(root string) *StubAdapter { return &StubAdapter{root: root} }

// Language implements Adapter.
func (*StubAdapter) Language() string { return StubLanguage }

// Detect implements Adapter.
func (*StubAdapter) Detect(files []string) (bool, Evidence) {
	var hits []string
	for _, f := range files {
		if strings.HasSuffix(f, stubFileExt) {
			hits = append(hits, f)
		}
	}
	if len(hits) == 0 {
		return false, Evidence{Reason: fmt.Sprintf("no %s file in the walked set", stubFileExt)}
	}
	return true, Evidence{
		Reason:  fmt.Sprintf("%d %s file(s) present", len(hits), stubFileExt),
		Details: hits,
	}
}

// Parse implements Adapter: never returns an error, never aborts.
func (s *StubAdapter) Parse(files []string) (*ir.ApiSurface, []ir.Diagnostic) {
	surface := &ir.ApiSurface{Source: ir.Source{Lang: StubLanguage, Framework: StubFramework}}
	var diags []ir.Diagnostic
	for _, rel := range files {
		if !strings.HasSuffix(rel, stubFileExt) {
			continue
		}
		data, err := s.read(rel)
		if err != nil {
			diags = append(diags, s.diag(rel, err.Error()))
			continue
		}
		var sf stubFile
		if err := json.Unmarshal(data, &sf); err != nil {
			diags = append(diags, s.diag(rel, "invalid JSON in stub file: "+err.Error()))
			continue
		}
		if sf.Service == "" {
			diags = append(diags, s.diag(rel, `stub file has no "service" name — file skipped`))
			continue
		}
		svc, fileDiags := s.service(&sf, rel)
		diags = append(diags, fileDiags...)
		surface.Services = append(surface.Services, svc)
		for _, st := range sf.Types {
			typ, tdiag := s.irType(st, rel)
			if tdiag != "" {
				diags = append(diags, s.diag(rel, tdiag))
				continue
			}
			surface.Types = append(surface.Types, typ)
		}
		if sf.Framework != "" && surface.Source.Framework == StubFramework {
			surface.Source.Framework = sf.Framework
		}
	}
	// w2-02's recommendation: adapters self-check before returning. A stub
	// bug surfaces as diagnostics, never as silent IR corruption.
	if v := surface.Validate(); len(v) > 0 {
		diags = append(diags, v...)
	}
	surface.Diagnostics = diags // the surface is self-contained for the engine (w2-04)
	return surface, diags
}

// read loads one stub file, bounded, with the escape-path check first.
func (s *StubAdapter) read(rel string) ([]byte, error) {
	if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return nil, fmt.Errorf("path %q escapes the scan root — refused", rel)
	}
	full := filepath.Join(s.root, filepath.FromSlash(rel))
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", rel, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, StubMaxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", rel, err)
	}
	if len(data) > StubMaxFileBytes {
		return nil, fmt.Errorf("%s exceeds the %d-byte stub read cap — file skipped", rel, StubMaxFileBytes)
	}
	return data, nil
}

// service converts one stub file's service block. Methods that cannot be
// translated exactly are dropped with a diagnostic (see type doc).
func (s *StubAdapter) service(sf *stubFile, rel string) (ir.Service, []ir.Diagnostic) {
	svc := ir.Service{Name: sf.Service, BasePath: cleanBasePath(sf.BasePath), Location: s.loc(rel)}
	var diags []ir.Diagnostic
	for _, sm := range sf.Methods {
		m, mdiag := s.method(sm, svc.BasePath, rel)
		if mdiag != "" {
			diags = append(diags, s.diag(rel, mdiag))
			continue
		}
		svc.Methods = append(svc.Methods, m)
	}
	return svc, diags
}

// method converts one stub method; a non-empty second return is the reason
// the method was dropped.
func (s *StubAdapter) method(sm stubMethod, basePath, rel string) (ir.Method, string) {
	if sm.Name == "" {
		return ir.Method{}, "a method has no name — method skipped"
	}
	verb := ir.Verb(sm.Verb)
	if !verb.Valid() {
		return ir.Method{}, fmt.Sprintf("method %s declares unknown verb %q — method skipped", sm.Name, sm.Verb)
	}
	m := ir.Method{
		OperationName: sm.Name,
		Verb:          verb,
		Path:          mergeStubPath(basePath, sm.Path),
		Response:      ir.Response{Location: s.loc(rel)},
		Pagination:    ir.Pagination{Style: ir.PaginationNone, Location: s.loc(rel)},
		Location:      s.loc(rel),
	}
	for _, sp := range sm.Params {
		in := ir.ParamIn(sp.In)
		switch in {
		case ir.ParamInPath, ir.ParamInQuery, ir.ParamInHeader, ir.ParamInBody:
		default:
			return ir.Method{}, fmt.Sprintf("method %s binds parameter %s to unknown location %q — method skipped",
				sm.Name, sp.Name, sp.In)
		}
		m.Params = append(m.Params, ir.Param{
			Name: sp.Name, In: in,
			Type: ir.TypeRef{Name: sp.Type}, Validation: sp.Validation,
			Location: s.loc(rel),
		})
	}
	if sm.Payload != "" {
		m.Payload = &ir.TypeRef{Name: sm.Payload}
	}
	if sm.Response != nil {
		m.Response.Type = ir.TypeRef{Name: sm.Response.Type}
		m.Response.StatusCode = sm.Response.StatusCode
		m.Response.IsCollection = sm.Response.IsCollection
	}
	switch p := sm.Pagination; p {
	case "":
		// default "none", already set
	case string(ir.PaginationPageable), string(ir.PaginationParams), string(ir.PaginationNone):
		m.Pagination.Style = ir.PaginationStyle(p)
	default:
		return ir.Method{}, fmt.Sprintf("method %s declares unknown pagination %q — method skipped", sm.Name, p)
	}
	return m, ""
}

// irType converts one stub type; a non-empty second return is the reason
// the type was dropped.
func (s *StubAdapter) irType(st stubType, rel string) (ir.Type, string) {
	if st.Name == "" {
		return ir.Type{}, "a type has no name — type skipped"
	}
	kind := ir.TypeKind(st.Kind)
	switch kind {
	case ir.KindRecord, ir.KindPOJO:
	case "": // documented default: record, the AIP-style data carrier
		kind = ir.KindRecord
	default:
		return ir.Type{}, fmt.Sprintf("type %s declares unknown kind %q — type skipped", st.Name, st.Kind)
	}
	t := ir.Type{Name: st.Name, Kind: kind, IsEntity: st.IsEntity, Location: s.loc(rel)}
	for _, sfld := range st.Fields {
		if sfld.Name == "" {
			return ir.Type{}, fmt.Sprintf("type %s has a field without a name — type skipped", st.Name)
		}
		t.Fields = append(t.Fields, ir.Field{
			Name: sfld.Name, JSONName: sfld.JSONName,
			Type: ir.TypeRef{Name: sfld.Type}, Location: s.loc(rel),
		})
	}
	return t, ""
}

// cleanBasePath normalizes the service base path: absolute, cleaned, no
// trailing slash; "" stays "" (the IR's "no base path" value).
func cleanBasePath(bp string) string {
	if bp == "" {
		return ""
	}
	return path.Join("/", bp)
}

// mergeStubPath joins the service base path and the method path the way the
// IR expects a full path: absolute and cleaned, no double slashes. The REAL
// merge — with its double-slash test matrix — belongs to the java adapter
// (w3-02); the stub only needs a well-formed IR path.
func mergeStubPath(basePath, methodPath string) string {
	return path.Join("/", basePath, methodPath)
}

// loc is the documented stub location: file-level, line 1, column 1.
func (s *StubAdapter) loc(rel string) ir.Location {
	return ir.Location{File: rel, Line: 1, Column: 1}
}

// diag builds one best-effort diagnostic anchored at the file.
func (s *StubAdapter) diag(rel, msg string) ir.Diagnostic {
	return ir.Diagnostic{Message: msg, Location: s.loc(rel)}
}

// stubFile mirrors the .stub.json schema (lowerCamel, like the IR wire
// format). Unknown JSON fields are ignored, so the fixture format can grow
// without breaking the adapter.
type stubFile struct {
	Service   string       `json:"service"`
	BasePath  string       `json:"basePath"`
	Framework string       `json:"framework"`
	Methods   []stubMethod `json:"methods"`
	Types     []stubType   `json:"types"`
}

type stubMethod struct {
	Name       string        `json:"name"`
	Verb       string        `json:"verb"`
	Path       string        `json:"path"`
	Params     []stubParam   `json:"params"`
	Payload    string        `json:"payload"`
	Response   *stubResponse `json:"response"`
	Pagination string        `json:"pagination"`
}

type stubParam struct {
	Name       string `json:"name"`
	In         string `json:"in"`
	Type       string `json:"type"`
	Validation string `json:"validation"`
}

type stubResponse struct {
	Type         string `json:"type"`
	StatusCode   int    `json:"statusCode"`
	IsCollection bool   `json:"isCollection"`
}

type stubType struct {
	Name     string      `json:"name"`
	Kind     string      `json:"kind"`
	IsEntity bool        `json:"isEntity"`
	Fields   []stubField `json:"fields"`
}

type stubField struct {
	Name     string `json:"name"`
	JSONName string `json:"jsonName"`
	Type     string `json:"type"`
}
