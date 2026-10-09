package ir

import (
	"fmt"
	"strings"
)

// Validate checks the structural invariants of a fully built ApiSurface
// (charter §5.2) and returns one Diagnostic per violation. Violations are
// always adapter bugs, never user-code problems: an adapter that produced
// them would make rules read nonsense. Recommended use: adapters call it
// before returning their surface; the engine may call it defensively and
// merge the results into Report.Diagnostics. Validate never aborts, never
// mutates the surface, and returns the diagnostics in a deterministic
// traversal order (services in order, then their methods and params, then
// types and fields, then error handlers, then gRPC services) — golden
// snapshots and tests rely on that.
//
// What is checked:
//   - Location well-formedness of every node (file present; line/column
//     both zero (synthetic) or both positive — UC1).
//   - Identity and duplicates: service names, types per (package, name),
//     method names per service, parameter (name, in) pairs per method,
//     field names per type, rpc names per gRPC service, gRPC service
//     names, and error handlers per exception type. The later occurrence
//     is flagged.
//   - Enum membership: Verb, ParamIn, PaginationStyle, TypeKind.
//   - The Payload ↔ body-parameter invariant for HTTP verbs (both views of
//     the same @RequestBody parameter must agree; rpc methods carry the
//     request message in Payload and take no body parameter).
//   - HTTP status codes: 0 (inferred/undeclared) or 100..599.
//   - Path normalization: BasePath empty or leading "/", Method.Path an
//     absolute path starting with "/" for HTTP verbs (w3-02 keeps it true).
//
// Not checked on purpose: Source (scan metadata, not API content), path
// merge correctness beyond normalization (w3-02 tests own that), and
// semantic lint concerns (e.g. duplicate JSON names, missing pagination on
// collections) — those are rule jobs, not structural invariants.
func (s *ApiSurface) Validate() []Diagnostic {
	if s == nil {
		return nil
	}
	v := &validator{}
	seenService := map[string]bool{}
	for i := range s.Services {
		v.service(&s.Services[i], i, seenService)
	}
	seenType := map[string]bool{}
	for i := range s.Types {
		v.typ(&s.Types[i], i, seenType)
	}
	seenHandler := map[string]bool{}
	for i := range s.ErrorScheme.Handlers {
		v.errorHandler(&s.ErrorScheme.Handlers[i], i, seenHandler)
	}
	seenGrpc := map[string]bool{}
	for i := range s.GrpcServices {
		v.grpcService(&s.GrpcServices[i], i, seenGrpc)
	}
	return v.diagnostics
}

// validStatus reports whether an HTTP status code is representable: 0 means
// "not declared" (consumers infer from the verb), otherwise 100..599.
func validStatus(code int) bool {
	return code == 0 || (code >= 100 && code <= 599)
}

type validator struct {
	diagnostics []Diagnostic
}

func (v *validator) add(loc Location, format string, args ...any) {
	v.diagnostics = append(v.diagnostics, Diagnostic{
		Message:  fmt.Sprintf(format, args...),
		Location: loc,
	})
}

// loc checks the well-formedness of one Location. what names the owning
// node in messages, e.g. `service "S", method "get"`.
func (v *validator) loc(l Location, what string) {
	if l.File == "" {
		v.add(l, "%s: Location.File must not be empty (paths are relative to the scan root)", what)
	}
	if (l.Line < 1 || l.Column < 1) && !(l.Line == 0 && l.Column == 0) {
		v.add(l, "%s: Location.Line and Location.Column must be both zero (synthetic) or both positive (1-based), got line %d column %d", what, l.Line, l.Column)
	}
}

func (v *validator) service(s *Service, i int, seen map[string]bool) {
	what := fmt.Sprintf("service %q", s.Name)
	if s.Name == "" {
		v.add(s.Location, "service %d: Name must not be empty (REST: controller class name; gRPC: proto service name)", i)
	}
	if s.BasePath != "" && !strings.HasPrefix(s.BasePath, "/") {
		v.add(s.Location, "%s: BasePath %q must be empty or start with \"/\"", what, s.BasePath)
	}
	v.loc(s.Location, what)
	if s.Name != "" {
		if seen[s.Name] {
			v.add(s.Location, "duplicate service name %q", s.Name)
		} else {
			seen[s.Name] = true
		}
	}
	seenMethod := map[string]bool{}
	for j := range s.Methods {
		v.method(&s.Methods[j], s, seenMethod)
	}
}

func (v *validator) method(m *Method, svc *Service, seenMethod map[string]bool) {
	what := fmt.Sprintf("service %q, method %q", svc.Name, m.OperationName)
	if m.OperationName == "" {
		v.add(m.Location, "%s: OperationName must not be empty (the Java method name or rpc name)", what)
	}
	if !m.Verb.Valid() {
		v.add(m.Location, "%s: unknown verb %q (want GET, POST, PUT, PATCH, DELETE or rpc)", what, string(m.Verb))
	}
	if m.Verb.IsHTTP() && (m.Path == "" || !strings.HasPrefix(m.Path, "/")) {
		v.add(m.Location, "%s: Path %q must be a normalized absolute path starting with \"/\"", what, m.Path)
	}
	v.loc(m.Location, what)
	if m.OperationName != "" {
		if seenMethod[m.OperationName] {
			v.add(m.Location, "service %q: duplicate method %q", svc.Name, m.OperationName)
		} else {
			seenMethod[m.OperationName] = true
		}
	}

	bodyCount := 0
	var body *Param
	seenParam := map[string]bool{}
	for i := range m.Params {
		p := &m.Params[i]
		v.param(p, what)
		if p.Name != "" {
			key := string(p.In) + "\x00" + p.Name
			if seenParam[key] {
				v.add(p.Location, "%s: duplicate parameter %q", what, p.Name)
			} else {
				seenParam[key] = true
			}
		}
		if p.In == ParamInBody {
			bodyCount++
			body = p
		}
	}
	if bodyCount > 1 {
		v.add(m.Location, "%s: %d in=body parameters declared (at most one is meaningful)", what, bodyCount)
	}
	switch {
	case m.Verb.IsHTTP() && m.Payload != nil && bodyCount == 0:
		v.add(m.Location, "%s: Payload %q is set but no in=body parameter declares it (Payload and the body Param are two views of the same @RequestBody parameter)", what, m.Payload.Name)
	case m.Verb.IsHTTP() && m.Payload == nil && bodyCount == 1:
		v.add(m.Location, "%s: body parameter %q is declared but Payload is nil (they are two views of the same @RequestBody parameter)", what, body.Name)
	case m.Verb.IsHTTP() && m.Payload != nil && bodyCount == 1 && body.Type != *m.Payload:
		v.add(m.Location, "%s: Payload %q does not match the body parameter type %q", what, m.Payload.Name, body.Type.Name)
	case m.Verb == VerbRPC && body != nil:
		v.add(m.Location, "%s: in=body parameter %q on an rpc method (body binding is an HTTP-only concept; carry the request message in Payload)", what, body.Name)
	}

	if !validStatus(m.Response.StatusCode) {
		v.add(m.Response.Location, "%s: response status code %d out of range (want 0 (inferred) or 100..599)", what, m.Response.StatusCode)
	}
	v.loc(m.Response.Location, what+" response")
	if !m.Pagination.Style.Valid() {
		v.add(m.Pagination.Location, "%s: unknown pagination style %q (want pageable, params or none)", what, string(m.Pagination.Style))
	}
	v.loc(m.Pagination.Location, what+" pagination")
}

func (v *validator) param(p *Param, what string) {
	if p.Name == "" {
		v.add(p.Location, "%s: parameter %q: Name must not be empty", what, p.Name)
	}
	switch p.In {
	case ParamInPath, ParamInQuery, ParamInHeader, ParamInBody:
	default:
		v.add(p.Location, "%s: parameter %q: unknown param location %q (want path, query, header or body)", what, p.Name, string(p.In))
	}
	if p.Type.Name == "" {
		v.add(p.Location, "%s: parameter %q: Type.Name must not be empty", what, p.Name)
	}
	v.loc(p.Location, fmt.Sprintf("%s parameter %q", what, p.Name))
}

func (v *validator) typ(t *Type, i int, seen map[string]bool) {
	what := fmt.Sprintf("type %q", t.Name)
	if t.Name == "" {
		v.add(t.Location, "type %d: Name must not be empty (the DTO/record class name)", i)
	}
	switch t.Kind {
	case KindRecord, KindPOJO:
	default:
		v.add(t.Location, "%s: unknown kind %q (want record or pojo)", what, string(t.Kind))
	}
	v.loc(t.Location, what)
	if t.Name != "" {
		key := t.Package + "\x00" + t.Name
		if seen[key] {
			v.add(t.Location, "duplicate type %q (package %q)", t.Name, t.Package)
		} else {
			seen[key] = true
		}
	}
	seenField := map[string]bool{}
	for j := range t.Fields {
		v.field(&t.Fields[j], t, seenField)
	}
}

func (v *validator) field(f *Field, t *Type, seen map[string]bool) {
	what := fmt.Sprintf("type %q, field %q", t.Name, f.Name)
	if f.Name == "" {
		v.add(f.Location, "%s: Name must not be empty", what)
	}
	if f.Type.Name == "" {
		v.add(f.Location, "%s: Type.Name must not be empty", what)
	}
	v.loc(f.Location, what)
	if f.Name != "" {
		if seen[f.Name] {
			v.add(f.Location, "type %q: duplicate field %q", t.Name, f.Name)
		} else {
			seen[f.Name] = true
		}
	}
}

func (v *validator) errorHandler(h *ErrorHandler, i int, seen map[string]bool) {
	what := fmt.Sprintf("error handler for exception %q", h.ExceptionType)
	if h.ExceptionType == "" {
		v.add(h.Location, "error handler %d: ExceptionType must not be empty (the @ExceptionHandler exception class)", i)
	}
	if !validStatus(h.StatusCode) {
		v.add(h.Location, "%s: status code %d out of range (want 0 (undeclared) or 100..599)", what, h.StatusCode)
	}
	if !validStatus(h.ExceptionStatus) {
		v.add(h.Location, "%s: exception-declared status %d out of range (want 0 (none) or 100..599)", what, h.ExceptionStatus)
	}
	v.loc(h.Location, what)
	if h.ExceptionType != "" {
		if seen[h.ExceptionType] {
			v.add(h.Location, "duplicate error handler for exception %q", h.ExceptionType)
		} else {
			seen[h.ExceptionType] = true
		}
	}
}

func (v *validator) grpcService(g *GrpcService, i int, seen map[string]bool) {
	if g.Name == "" {
		v.add(g.Location, "gRPC service %d: Name must not be empty", i)
	}
	v.loc(g.Location, fmt.Sprintf("gRPC service %q", g.Name))
	if g.Name != "" {
		if seen[g.Name] {
			v.add(g.Location, "duplicate gRPC service name %q", g.Name)
		} else {
			seen[g.Name] = true
		}
	}
	seenRPC := map[string]bool{}
	for j, rpc := range g.Rpcs {
		if rpc == "" {
			v.add(g.Location, "gRPC service %q: rpc %d: name must not be empty", g.Name, j)
		} else if seenRPC[rpc] {
			v.add(g.Location, "gRPC service %q: duplicate rpc %q", g.Name, rpc)
		} else {
			seenRPC[rpc] = true
		}
	}
}
