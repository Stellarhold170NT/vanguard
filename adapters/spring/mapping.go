package spring

import (
	"fmt"
	"path"
	"strings"

	"github.com/Stellarhold170NT/vanguard/adapters/java"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// Dedicated Spring mapping annotations whose verb is fixed by their name.
var mappingVerbs = map[string]ir.Verb{
	"GetMapping":    ir.VerbGet,
	"PostMapping":   ir.VerbPost,
	"PutMapping":    ir.VerbPut,
	"PatchMapping":  ir.VerbPatch,
	"DeleteMapping": ir.VerbDelete,
}

// RequestMethod spellings the IR can represent. HEAD is served as GET
// (same handler semantics, the IR has no HEAD verb); OPTIONS and TRACE
// have no representation and skip the method.
var requestMethodVerbs = map[string]ir.Verb{
	"GET":    ir.VerbGet,
	"HEAD":   ir.VerbGet,
	"POST":   ir.VerbPost,
	"PUT":    ir.VerbPut,
	"PATCH":  ir.VerbPatch,
	"DELETE": ir.VerbDelete,
}

// httpStatusCodes maps the org.springframework.http.HttpStatus constants
// the IR can express. The table is the standard HTTP status set; a name
// outside it is an unresolved annotation, never a failure.
var httpStatusCodes = map[string]int{
	"CONTINUE":                        100,
	"SWITCHING_PROTOCOLS":             101,
	"PROCESSING":                      102,
	"CHECKPOINT":                      103,
	"OK":                              200,
	"CREATED":                         201,
	"ACCEPTED":                        202,
	"NON_AUTHORITATIVE_INFORMATION":   203,
	"NO_CONTENT":                      204,
	"RESET_CONTENT":                   205,
	"PARTIAL_CONTENT":                 206,
	"MULTI_STATUS":                    207,
	"MULTIPLE_CHOICES":                300,
	"MOVED_PERMANENTLY":               301,
	"FOUND":                           302,
	"MOVED_TEMPORARILY":               302,
	"SEE_OTHER":                       303,
	"NOT_MODIFIED":                    304,
	"USE_PROXY":                       305,
	"TEMPORARY_REDIRECT":              307,
	"PERMANENT_REDIRECT":              308,
	"BAD_REQUEST":                     400,
	"UNAUTHORIZED":                    401,
	"PAYMENT_REQUIRED":                402,
	"FORBIDDEN":                       403,
	"NOT_FOUND":                       404,
	"METHOD_NOT_ALLOWED":              405,
	"NOT_ACCEPTABLE":                  406,
	"PROXY_AUTHENTICATION_REQUIRED":   407,
	"REQUEST_TIMEOUT":                 408,
	"CONFLICT":                        409,
	"GONE":                            410,
	"LENGTH_REQUIRED":                 411,
	"PRECONDITION_FAILED":             412,
	"PAYLOAD_TOO_LARGE":               413,
	"REQUEST_ENTITY_TOO_LARGE":        413,
	"URI_TOO_LONG":                    414,
	"REQUEST_URI_TOO_LONG":            414,
	"UNSUPPORTED_MEDIA_TYPE":          415,
	"REQUESTED_RANGE_NOT_SATISFIABLE": 416,
	"RANGE_NOT_SATISFIABLE":           416,
	"EXPECTATION_FAILED":              417,
	"I_AM_A_TEAPOT":                   418,
	"UNPROCESSABLE_ENTITY":            422,
	"LOCKED":                          423,
	"FAILED_DEPENDENCY":               424,
	"TOO_EARLY":                       425,
	"UPGRADE_REQUIRED":                426,
	"PRECONDITION_REQUIRED":           428,
	"TOO_MANY_REQUESTS":               429,
	"REQUEST_HEADER_FIELDS_TOO_LARGE": 431,
	"UNAVAILABLE_FOR_LEGAL_REASONS":   451,
	"INTERNAL_SERVER_ERROR":           500,
	"NOT_IMPLEMENTED":                 501,
	"BAD_GATEWAY":                     502,
	"SERVICE_UNAVAILABLE":             503,
	"GATEWAY_TIMEOUT":                 504,
	"HTTP_VERSION_NOT_SUPPORTED":      505,
	"VARIANT_ALSO_NEGOTIATES":         506,
	"INSUFFICIENT_STORAGE":            507,
	"LOOP_DETECTED":                   508,
	"BANDWIDTH_LIMIT_EXCEEDED":        509,
	"NOT_EXTENDED":                    510,
	"NETWORK_AUTHENTICATION_REQUIRED": 511,
}

// transportWrappers unwrap to their type argument; Flux additionally marks
// the result a collection (a stream of elements, the WebFlux reading).
var transportWrappers = map[string]bool{
	"ResponseEntity":    true,
	"HttpEntity":        true,
	"RequestEntity":     true,
	"Mono":              true,
	"Flux":              true,
	"Optional":          true,
	"CompletableFuture": true,
	"CompletionStage":   true,
	"Callable":          true,
	"DeferredResult":    true,
}

// collectionBases mark a type use as a collection (wrapper erasure the
// R3xx family reads). Arrays and varargs count separately.
var collectionBases = map[string]bool{
	"List":          true,
	"ArrayList":     true,
	"LinkedList":    true,
	"Collection":    true,
	"Set":           true,
	"HashSet":       true,
	"LinkedHashSet": true,
	"SortedSet":     true,
	"TreeSet":       true,
	"Queue":         true,
	"Deque":         true,
	"Iterable":      true,
	"Page":          true,
	"Slice":         true,
	"PageImpl":      true,
	"Stream":        true,
	"Flux":          true,
}

// scalarParamTypes are the types Spring binds from the query string
// without any annotation (implicit @RequestParam binding).
var scalarParamTypes = map[string]bool{
	"int": true, "long": true, "short": true, "byte": true, "char": true,
	"boolean": true, "float": true, "double": true,
	"Integer": true, "Long": true, "Short": true, "Byte": true,
	"Character": true, "Boolean": true, "Float": true, "Double": true,
	"String": true, "CharSequence": true, "UUID": true,
	"BigDecimal": true, "BigInteger": true,
	"LocalDate": true, "LocalDateTime": true, "LocalTime": true,
	"OffsetDateTime": true, "ZonedDateTime": true, "Instant": true,
	"Duration": true,
}

// bindingAnnotations consume a parameter's binding slot; every other
// parameter annotation is a constraint and lands in Param.Validation.
var bindingAnnotations = map[string]bool{
	"PathVariable":    true,
	"RequestParam":    true,
	"RequestHeader":   true,
	"RequestBody":     true,
	"RequestPart":     true,
	"PageableDefault": true,
}

// pageParamNames are the query parameter names the R3xx family reads as
// pagination input.
var pageParamNames = map[string]bool{
	"page": true, "size": true, "limit": true, "offset": true,
}

// cappedPageParamNames are the size-carrying names whose @Max-style
// constraint marks the page size capped (R3xx-02).
var cappedPageParamNames = map[string]bool{
	"size": true, "limit": true,
}

// simpleName returns the last dot-segment of a (possibly qualified) name
// — the local mirror of the tree-sitter pass's unexported helper (the
// java package keeps it private, so the mapping layer needs its own).
func simpleName(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}

// mergePath joins the service base path and the method path the way the
// IR expects a full request path: absolute, cleaned, no double slashes,
// no trailing slash. The empty base falls through cleanly; wildcard
// segments survive verbatim (documented limitation).
func mergePath(base, methodPath string) string {
	return path.Join("/", base, methodPath)
}

// verbOf decides the HTTP verb of one mapping annotation: the dedicated
// @…Mapping spellings fix it by name; @RequestMapping reads its method=
// element (first value wins when an array). The second return is the
// unresolved-annotation reason, "" when the decision was lossless.
func verbOf(a java.Annotation) (ir.Verb, string) {
	if v, ok := mappingVerbs[a.Name]; ok {
		return v, ""
	}
	var raw string
	for _, arg := range a.Args {
		if arg.Name == "method" {
			raw = arg.Value
			break
		}
	}
	if raw == "" {
		return ir.VerbGet, "no method= element — verb defaulted to GET"
	}
	for _, part := range splitTopLevel(strings.TrimSuffix(strings.TrimPrefix(raw, "{"), "}")) {
		name := simpleName(strings.TrimSpace(part))
		if v, ok := requestMethodVerbs[name]; ok {
			if name == "HEAD" {
				return v, "RequestMethod.HEAD mapped to GET"
			}
			return v, ""
		}
	}
	return ir.VerbGet, "unsupported method= value — verb defaulted to GET"
}

// httpStatusOf reads the declared status out of @ResponseStatus. The
// second return is the unresolved-annotation reason, "" when declared.
func httpStatusOf(a java.Annotation) (int, string) {
	var first string
	for _, arg := range a.Args {
		if arg.Name != "" && arg.Name != "value" && arg.Name != "code" {
			continue
		}
		name := simpleName(strings.TrimSpace(arg.Value))
		if code, ok := httpStatusCodes[name]; ok {
			return code, ""
		}
		if first == "" {
			first = name
		}
	}
	if first == "" {
		return 0, "no HttpStatus element — status left undeclared"
	}
	return 0, fmt.Sprintf("unknown HttpStatus constant %q — status left undeclared", first)
}

// pathOf reads the mapping path: the unnamed string argument, then value=,
// then path=; an array contributes its first element (a note for the rest
// comes from pathCountOf).
func pathOf(a java.Annotation) string {
	var unnamed, value, named string
	for _, arg := range a.Args {
		switch arg.Name {
		case "":
			if unnamed == "" {
				unnamed = firstPathElement(arg.Value)
			}
		case "value":
			if value == "" {
				value = firstPathElement(arg.Value)
			}
		case "path":
			if named == "" {
				named = firstPathElement(arg.Value)
			}
		}
	}
	if unnamed != "" {
		return unnamed
	}
	if value != "" {
		return value
	}
	return named
}

// pathCountOf counts how many path values a mapping declares, so the
// caller can note "first kept" when more than one exists.
func pathCountOf(a java.Annotation) int {
	n := 0
	for _, arg := range a.Args {
		if arg.Name != "" && arg.Name != "value" && arg.Name != "path" {
			continue
		}
		n += len(splitTopLevel(strings.TrimSuffix(strings.TrimPrefix(arg.Value, "{"), "}")))
	}
	return n
}

// bindingName reads the bind name out of a binding annotation: the unnamed
// string argument, then value=, then name= (Spring's precedence for
// @RequestParam aliases). Quotes are stripped; "" means "the parameter's
// own name".
func bindingName(a java.Annotation) string {
	for _, want := range []string{"", "value", "name"} {
		for _, arg := range a.Args {
			if arg.Name == want && unquote(arg.Value) != "" {
				return unquote(arg.Value)
			}
		}
	}
	return ""
}

// splitTopLevel splits "a, b" or "{a, b}" on commas that sit outside any
// nested braces or generics, keeping annotation/array spellings intact.
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '{', '(', '[', '<':
			depth++
		case '}', ')', ']', '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// firstPathElement extracts the first path string from a raw annotation
// argument value: `"/x"`, `{"/x", "/y"}`, plain text.
func firstPathElement(raw string) string {
	parts := splitTopLevel(strings.TrimSuffix(strings.TrimPrefix(raw, "{"), "}"))
	if len(parts) == 0 {
		return ""
	}
	return unquote(parts[0])
}

// unquote strips one layer of surrounding quotes from a raw string value.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseTypeUse re-parses a raw type spelling ("List<BookDto>") into the
// adapter's TypeUse shape: full Name, erasure Base, top-level Args.
func parseTypeUse(raw string) java.TypeUse {
	raw = strings.TrimSpace(raw)
	depth := 0
	open := -1
	for i, r := range raw {
		switch r {
		case '<':
			if depth == 0 && open < 0 {
				open = i
			}
			depth++
		case '>':
			depth--
		}
	}
	if open < 0 {
		return java.TypeUse{Name: raw, Base: raw}
	}
	tu := java.TypeUse{Name: raw, Base: raw[:open]}
	inner := raw[open+1 : len(raw)-strings.Count(raw[open:], ">")] // up to the matching final '>'
	for _, a := range splitTopLevel(inner) {
		tu.Args = append(tu.Args, a)
	}
	return tu
}

// isCollectionUse reports whether a type use is a collection shape:
// array spelling, a known collection base, or varargs.
func isCollectionUse(t java.TypeUse) bool {
	if strings.HasSuffix(t.Name, "[]") {
		return true
	}
	return collectionBases[simpleName(t.Base)]
}

// typeIndex maps a type simple name to the package of the scanned file
// that declared it (first declaration wins; documented ambiguity policy).
type typeIndex map[string]string

// buildTypeIndex indexes every extracted declaration, nested included,
// by its file's package.
func buildTypeIndex(files []*java.File) typeIndex {
	idx := typeIndex{}
	var walk func(cls *java.Class, pkg string)
	walk = func(cls *java.Class, pkg string) {
		if _, seen := idx[cls.Name]; !seen {
			idx[cls.Name] = pkg
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
	return idx
}

// boundParams is the outcome of binding one method's parameters.
type boundParams struct {
	params     []ir.Param
	payload    *ir.TypeRef
	pagination ir.Pagination
	notes      []string // unresolved-annotation reasons, in source order
}

// bindParams maps the raw parameters of one handler into IR params: an
// explicit binding annotation decides the location; a scalar type binds
// implicitly as a query parameter (Spring's rule); anything else is a
// model-attribute binding v0.1 cannot represent and is skipped with a
// note. @RequestBody fills both the body param and the Payload view. The
// pagination style follows the Pageable parameter or page/size parameters.
func bindParams(params []java.Param) boundParams {
	out := boundParams{pagination: ir.Pagination{Style: ir.PaginationNone}}
	var pageableSeen, paramsSeen bool
	var capped bool
	var firstPageParam ir.Location
	for _, p := range params {
		base := simpleName(p.Type.Base)
		var in ir.ParamIn
		var binding *java.Annotation
		keep := true
		switch {
		case findAnnotation(p.Annotations, "RequestBody") != nil:
			binding = findAnnotation(p.Annotations, "RequestBody")
			in = ir.ParamInBody
		case findAnnotation(p.Annotations, "RequestPart") != nil:
			binding = findAnnotation(p.Annotations, "RequestPart")
			in = ir.ParamInBody // multipart part: the closest IR concept to a body
		case findAnnotation(p.Annotations, "PathVariable") != nil:
			binding = findAnnotation(p.Annotations, "PathVariable")
			in = ir.ParamInPath
		case findAnnotation(p.Annotations, "RequestParam") != nil:
			binding = findAnnotation(p.Annotations, "RequestParam")
			in = ir.ParamInQuery
		case findAnnotation(p.Annotations, "RequestHeader") != nil:
			binding = findAnnotation(p.Annotations, "RequestHeader")
			in = ir.ParamInHeader
		case base == "Pageable" || findAnnotation(p.Annotations, "PageableDefault") != nil:
			in = ir.ParamInQuery // a Pageable arrives as query parameters
			pageableSeen = true
		case scalarParamTypes[base]:
			in = ir.ParamInQuery
		default:
			keep = false
			out.notes = append(out.notes, fmt.Sprintf(
				"parameter %s carries no binding annotation — model-attribute binding is unsupported (v0.1), parameter skipped", p.Name))
		}
		if !keep {
			continue
		}
		name := p.Name
		if binding != nil {
			if n := bindingName(*binding); n != "" {
				name = n
			}
		}
		var validation []string
		for _, a := range p.Annotations {
			if !bindingAnnotations[a.Name] {
				validation = append(validation, a.Raw)
			}
		}
		collection := isCollectionUse(p.Type) || p.IsVarargs
		loc := p.Location
		out.params = append(out.params, ir.Param{
			Name:       name,
			In:         in,
			Type:       ir.TypeRef{Name: p.Type.Name, IsCollection: collection},
			Validation: strings.Join(validation, " "),
			Location:   loc,
		})
		if in == ir.ParamInBody && out.payload == nil {
			out.payload = &ir.TypeRef{Name: p.Type.Name, IsCollection: collection}
		}
		if !pageableSeen && in == ir.ParamInQuery && pageParamNames[strings.ToLower(name)] {
			paramsSeen = true
			if firstPageParam == (ir.Location{}) {
				firstPageParam = loc
			}
			if cappedPageParamNames[strings.ToLower(name)] && strings.Contains(strings.Join(validation, " "), "@Max") {
				capped = true
			}
		}
	}
	switch {
	case pageableSeen:
		out.pagination = ir.Pagination{Style: ir.PaginationPageable, Location: firstSignal(params)}
	case paramsSeen:
		out.pagination = ir.Pagination{Style: ir.PaginationParams, Location: firstPageParam, PageSizeCapped: capped}
	}
	return out
}

// firstSignal returns the location of the first Pageable/PageableDefault
// parameter (the signal the R3xx family anchors on).
func firstSignal(params []java.Param) ir.Location {
	for _, p := range params {
		if simpleName(p.Type.Base) == "Pageable" || findAnnotation(p.Annotations, "PageableDefault") != nil {
			return p.Location
		}
	}
	return ir.Location{}
}

// mapper carries the shared context of one mapping pass over a Result.
type mapper struct {
	index typeIndex
}

// typeRef converts a raw type use into the IR's TypeRef: transport
// wrappers (ResponseEntity/Mono/Flux/…) unwrap to their argument, Flux
// marks the result a collection, the package resolves through the scan
// scope index, and container shapes (List<…>, Page<…>, arrays) keep their
// spelling with IsCollection set.
func (m *mapper) typeRef(t java.TypeUse) ir.TypeRef {
	flux := false
	for {
		base := simpleName(t.Base)
		if len(t.Args) == 0 || (!transportWrappers[base]) {
			break
		}
		if base == "Flux" {
			flux = true
		}
		t = parseTypeUse(t.Args[0])
	}
	return ir.TypeRef{
		Name:         t.Name,
		Package:      m.index[simpleName(t.Base)],
		IsCollection: isCollectionUse(t) || flux,
	}
}

// response builds the Response view of one handler: void yields the zero
// TypeRef; everything else unwraps. Location points at the method (the
// declared status comes from annotations on it).
func (m *mapper) response(rt java.TypeUse, status int, loc ir.Location) ir.Response {
	if rt.Name == "void" {
		return ir.Response{StatusCode: status, Location: loc}
	}
	tr := m.typeRef(rt)
	return ir.Response{Type: tr, StatusCode: status, IsCollection: tr.IsCollection, Location: loc}
}

// addAnnotation stores one raw annotation under its simple name, joining
// duplicates with "\n" (the IR's shared convention).
func addAnnotation(anns map[string]string, a java.Annotation) {
	anns[a.Name] = joinUnresolved(anns[a.Name], a.Raw)
}

// joinUnresolved appends one more reason/raw text to a "\n"-joined value.
func joinUnresolved(existing, add string) string {
	if existing == "" {
		return add
	}
	return existing + "\n" + add
}

// resolve maps a raw type spelling to the declaring package recorded in
// the scan scope index ("" when the type is not declared in the scan).
func (m *mapper) resolve(raw string) string {
	return m.index[simpleName(parseTypeUse(raw).Base)]
}

// noteAt records one unresolved element: the reason joins the owning
// node's Annotations["unresolved"] entry (machine-readable metadata per
// the brief) and a diagnostic carries the full sentence for verbose mode.
func noteAt(anns map[string]string, sink *[]ir.Diagnostic, loc ir.Location, what, reason string) {
	anns["unresolved"] = joinUnresolved(anns["unresolved"], reason)
	*sink = append(*sink, ir.Diagnostic{Message: what + ": " + reason, Location: loc})
}

// findAnnotation returns the named annotation, nil when absent.
func findAnnotation(anns []java.Annotation, name string) *java.Annotation {
	for i := range anns {
		if anns[i].Name == name {
			return &anns[i]
		}
	}
	return nil
}

// classHasAny reports whether the class carries one of the named
// annotations (simple names, direct declarations only — meta-annotations
// are a documented v0.1 limitation).
func classHasAny(c *java.Class, names ...string) bool {
	for _, a := range c.Annotations {
		for _, n := range names {
			if a.Name == n {
				return true
			}
		}
	}
	return false
}

// mappingLike reports whether an unknown annotation name smells like a
// request mapping (the meta-annotation trap: such an annotation is
// reported unresolved instead of silently ignored).
func mappingLike(name string) bool {
	if strings.HasSuffix(name, "Mapping") {
		return true
	}
	lower := strings.ToLower(name)
	for _, verb := range []string{"get", "post", "put", "patch", "delete"} {
		if strings.HasSuffix(lower, verb) {
			return true
		}
	}
	return false
}

// mapService builds the IR service for one controller class. The sink
// collects diagnostics for unresolved annotations.
func (m *mapper) mapService(c *java.Class, sink *[]ir.Diagnostic) (ir.Service, bool) {
	svc := ir.Service{
		Name:        c.Name,
		Annotations: map[string]string{},
		Location:    c.Location,
	}
	classStatus := 0
	for _, a := range c.Annotations {
		addAnnotation(svc.Annotations, a)
		switch a.Name {
		case "RequestMapping":
			svc.BasePath = mergePath(pathOf(a), "")
			if pathCountOf(a) > 1 {
				noteAt(svc.Annotations, sink, c.Location,
					fmt.Sprintf("unresolved @RequestMapping on %s", c.Name),
					"multiple paths declared — first kept")
			}
		case "ResponseStatus":
			code, note := httpStatusOf(a)
			classStatus = code
			if note != "" {
				noteAt(svc.Annotations, sink, c.Location,
					fmt.Sprintf("unresolved @ResponseStatus on %s", c.Name), note)
			}
		}
	}
	for _, cm := range c.Methods {
		if mm, ok := m.mapMethod(c, cm, classStatus, svc.BasePath, sink); ok {
			svc.Methods = append(svc.Methods, mm)
		}
	}
	return svc, true
}

// mapMethod builds the IR method for one handler; ok is false for methods
// that carry no mapping annotation or an unsupported one.
func (m *mapper) mapMethod(cls *java.Class, cm java.Method, classStatus int, basePath string, sink *[]ir.Diagnostic) (ir.Method, bool) {
	var mapping *java.Annotation
	var unknown *java.Annotation
	for i := range cm.Annotations {
		a := &cm.Annotations[i]
		if mappingVerbs[a.Name] != "" || a.Name == "RequestMapping" {
			mapping = a
		} else if mappingLike(a.Name) {
			unknown = a
		}
	}
	if mapping == nil {
		if unknown != nil {
			*sink = append(*sink, ir.Diagnostic{
				Message: fmt.Sprintf("unresolved @%s on %s.%s: mapping-like annotation is not a known Spring mapping — method skipped",
					unknown.Name, cls.Name, cm.Name),
				Location: cm.Location,
			})
		}
		return ir.Method{}, false
	}
	what := fmt.Sprintf("unresolved @%s on %s.%s", mapping.Name, cls.Name, cm.Name)
	verb, vnote := verbOf(*mapping)
	if verb == "" {
		*sink = append(*sink, ir.Diagnostic{
			Message:  what + ": unsupported RequestMethod — method skipped",
			Location: cm.Location,
		})
		return ir.Method{}, false
	}
	anns := map[string]string{}
	for _, a := range cm.Annotations {
		addAnnotation(anns, a)
	}
	if vnote != "" {
		noteAt(anns, sink, cm.Location, what, vnote)
	}
	if pathCountOf(*mapping) > 1 {
		noteAt(anns, sink, cm.Location, what, "multiple paths declared — first kept")
	}
	status := classStatus
	for _, a := range cm.Annotations {
		if a.Name == "ResponseStatus" {
			code, note := httpStatusOf(a)
			if code != 0 {
				status = code
			} else if note != "" {
				noteAt(anns, sink, cm.Location, what, note)
			}
		}
	}
	bp := bindParams(cm.Params)
	// Resolve packages through the scan scope index: params and payload are
	// API surface references, so a type declared in the scan gets its
	// package (same resolution rule as response types).
	for i := range bp.params {
		bp.params[i].Type.Package = m.resolve(bp.params[i].Type.Name)
	}
	if bp.payload != nil {
		bp.payload.Package = m.resolve(bp.payload.Name)
	}
	for _, n := range bp.notes {
		noteAt(anns, sink, cm.Location, fmt.Sprintf("unresolved parameter on %s.%s", cls.Name, cm.Name), n)
	}
	pagination := bp.pagination
	if pagination.Location == (ir.Location{}) {
		pagination.Location = cm.Location
	}
	return ir.Method{
		OperationName: cm.Name,
		Verb:          verb,
		Path:          mergePath(basePath, pathOf(*mapping)),
		Params:        bp.params,
		Payload:       bp.payload,
		Response:      m.response(cm.ReturnType, status, cm.Location),
		Pagination:    pagination,
		Annotations:   anns,
		Location:      cm.Location,
	}, true
}

// mapAdvice collects the @ExceptionHandler methods of one advice class
// into global error handlers (the R5xx input).
func (m *mapper) mapAdvice(c *java.Class, sink *[]ir.Diagnostic) []ir.ErrorHandler {
	var out []ir.ErrorHandler
	classStatus := 0
	for _, a := range c.Annotations {
		if a.Name == "ResponseStatus" {
			classStatus, _ = httpStatusOf(a)
		}
	}
	for _, cm := range c.Methods {
		ha := findAnnotation(cm.Annotations, "ExceptionHandler")
		if ha == nil {
			continue
		}
		excs := exceptionNames(ha, cm)
		if len(excs) == 0 {
			*sink = append(*sink, ir.Diagnostic{
				Message:  fmt.Sprintf("unresolved @ExceptionHandler on %s.%s: no exception type declared — handler skipped", c.Name, cm.Name),
				Location: cm.Location,
			})
			continue
		}
		status := classStatus
		for _, a := range cm.Annotations {
			if a.Name == "ResponseStatus" {
				code, note := httpStatusOf(a)
				if code != 0 {
					status = code
				} else if note != "" {
					*sink = append(*sink, ir.Diagnostic{
						Message:  fmt.Sprintf("unresolved @ResponseStatus on %s.%s: %s", c.Name, cm.Name, note),
						Location: cm.Location,
					})
				}
			}
		}
		resp := m.response(cm.ReturnType, 0, cm.Location)
		for _, ex := range excs {
			out = append(out, ir.ErrorHandler{
				ExceptionType: ex,
				ResponseType:  resp.Type.Name,
				StatusCode:    status,
				Location:      cm.Location,
			})
		}
	}
	return out
}

// exceptionNames reads the exception classes out of @ExceptionHandler:
// the annotation's class arguments first (".class" stripped, arrays
// split), falling back to the method's first parameter type (Spring's
// zero-argument form).
func exceptionNames(ha *java.Annotation, cm java.Method) []string {
	var names []string
	for _, arg := range ha.Args {
		if arg.Name != "" && arg.Name != "value" {
			continue
		}
		for _, part := range splitTopLevel(strings.TrimSuffix(strings.TrimPrefix(arg.Value, "{"), "}")) {
			n := simpleName(strings.TrimSuffix(strings.TrimSpace(part), ".class"))
			if n != "" && n != "()" {
				names = append(names, n)
			}
		}
	}
	if len(names) == 0 && len(cm.Params) > 0 {
		names = append(names, simpleName(cm.Params[0].Type.Base))
	}
	return names
}

// isGetter reports whether a method reads like an accessor (getX/isY and
// not void) — the brief's "POJO with getters" qualification.
func isGetter(m java.Method) bool {
	if m.ReturnType.Name == "void" {
		return false
	}
	n := m.Name
	return (strings.HasPrefix(n, "get") && len(n) > 3) ||
		(strings.HasPrefix(n, "is") && len(n) > 2)
}

// isDTOKind reports whether a scanned class is extractable as an IR Type:
// records always (components are the accessors), classic classes when
// they carry accessors (explicit or Lombok-generated).
func isDTOKind(c *java.Class) bool {
	switch c.Kind {
	case java.KindRecord:
		return true
	case java.KindClass:
		for _, a := range c.Annotations {
			if a.Name == "Data" || a.Name == "Getter" || a.Name == "Value" {
				return true
			}
		}
		for _, m := range c.Methods {
			if isGetter(m) {
				return true
			}
		}
	}
	return false
}

// mapType converts one qualifying class into an IR Type with its fields
// in declaration order (static constants excluded for classic classes).
func (m *mapper) mapType(c *java.Class, pkg string) ir.Type {
	t := ir.Type{Name: c.Name, Package: pkg, Location: c.Location}
	if c.Kind == java.KindRecord {
		t.Kind = ir.KindRecord
	} else {
		t.Kind = ir.KindPOJO
	}
	for _, a := range c.Annotations {
		if a.Name == "Entity" || a.Name == "Document" {
			t.IsEntity = true
		}
	}
	for _, f := range c.Fields {
		if c.Kind == java.KindClass && hasModifier(f.Modifiers, "static") {
			continue
		}
		fld := ir.Field{
			Name:     f.Name,
			Type:     m.typeRef(f.Type),
			Location: f.Location,
		}
		for _, a := range f.Annotations {
			if fld.Annotations == nil {
				fld.Annotations = map[string]string{}
			}
			addAnnotation(fld.Annotations, a)
			if (a.Name == "JsonProperty" || a.Name == "SerializedName") && fld.JSONName == "" {
				fld.JSONName = bindingName(a)
			}
		}
		t.Fields = append(t.Fields, fld)
	}
	return t
}

// hasModifier reports whether the modifier keyword list contains m.
func hasModifier(mods []string, m string) bool {
	for _, x := range mods {
		if x == m {
			return true
		}
	}
	return false
}
