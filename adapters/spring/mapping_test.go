package spring

import (
	"reflect"
	"testing"

	"github.com/Stellarhold170NT/vanguard/adapters/java"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// ann builds one annotation from raw text plus raw arguments.
func ann(name string, args ...java.AnnotationArg) java.Annotation {
	return java.Annotation{Name: name, Raw: "@" + name, Args: args}
}

func strArg(v string) java.AnnotationArg { return java.AnnotationArg{Value: v} }

func namedArg(k, v string) java.AnnotationArg { return java.AnnotationArg{Name: k, Value: v} }

func TestMergePath(t *testing.T) {
	cases := []struct {
		base, method, want string
	}{
		{"/api/v1", "/books/{id}", "/api/v1/books/{id}"}, // the classic merge
		{"", "/books", "/books"},
		{"/api/v1", "", "/api/v1"},
		{"/api/v1/", "/books/", "/api/v1/books"},  // trailing slashes collapse
		{"/api//v1", "//books", "/api/v1/books"},  // double slashes are banned
		{"api/v1", "books/{id}", "/api/v1/books/{id}"}, // missing leading slashes normalized
		{"/a", "/**", "/a/**"},                    // wildcard survives verbatim (limitation)
	}
	for _, c := range cases {
		if got := mergePath(c.base, c.method); got != c.want {
			t.Errorf("mergePath(%q, %q) = %q, want %q", c.base, c.method, got, c.want)
		}
	}
}

func TestUnwrapResponseType(t *testing.T) {
	idx := typeIndex{"BookDto": "com.example.books"}
	cases := []struct {
		name     string
		typ      java.TypeUse
		want     ir.TypeRef
	}{
		{"responseEntity", java.TypeUse{Name: "ResponseEntity<List<BookDto>>", Base: "ResponseEntity", Args: []string{"List<BookDto>"}},
			ir.TypeRef{Name: "List<BookDto>", IsCollection: true}},
		{"responseEntitySingle", java.TypeUse{Name: "ResponseEntity<BookDto>", Base: "ResponseEntity", Args: []string{"BookDto"}},
			ir.TypeRef{Name: "BookDto", Package: "com.example.books"}},
		{"mono", java.TypeUse{Name: "Mono<BookDto>", Base: "Mono", Args: []string{"BookDto"}},
			ir.TypeRef{Name: "BookDto", Package: "com.example.books"}},
		{"flux", java.TypeUse{Name: "Flux<BookDto>", Base: "Flux", Args: []string{"BookDto"}},
			ir.TypeRef{Name: "BookDto", Package: "com.example.books", IsCollection: true}},
		{"monoOfList", java.TypeUse{Name: "Mono<List<BookDto>>", Base: "Mono", Args: []string{"List<BookDto>"}},
			ir.TypeRef{Name: "List<BookDto>", IsCollection: true}},
		{"pageStays", java.TypeUse{Name: "Page<BookDto>", Base: "Page", Args: []string{"BookDto"}},
			ir.TypeRef{Name: "Page<BookDto>", IsCollection: true}},
		{"optional", java.TypeUse{Name: "Optional<BookDto>", Base: "Optional", Args: []string{"BookDto"}},
			ir.TypeRef{Name: "BookDto", Package: "com.example.books"}},
		{"array", java.TypeUse{Name: "BookDto[]", Base: "BookDto"},
			ir.TypeRef{Name: "BookDto[]", Package: "com.example.books", IsCollection: true}},
		{"plain", java.TypeUse{Name: "BookDto", Base: "BookDto"},
			ir.TypeRef{Name: "BookDto", Package: "com.example.books"}},
		{"rawWrapperNoArgs", java.TypeUse{Name: "ResponseEntity", Base: "ResponseEntity"},
			ir.TypeRef{Name: "ResponseEntity"}},
	}
	m := &mapper{index: idx}
	for _, c := range cases {
		if got := m.typeRef(unwrapTypeUse(c.typ)); got != c.want {
			t.Errorf("%s: typeRef = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestVoidResponseTypeIsZero(t *testing.T) {
	m := &mapper{index: typeIndex{}}
	got := m.response(java.TypeUse{Name: "void", Base: "void"}, nil, 0, ir.Location{File: "a.java", Line: 1, Column: 1})
	if got.Type.Name != "" {
		t.Errorf("void response type = %q, want empty", got.Type.Name)
	}
}

func TestVerbOfMapping(t *testing.T) {
	cases := []struct {
		name     string
		a        java.Annotation
		want     ir.Verb
		wantNote string // "" = no unresolved note
	}{
		{"getMarker", ann("GetMapping"), ir.VerbGet, ""},
		{"postMarker", ann("PostMapping"), ir.VerbPost, ""},
		{"putMarker", ann("PutMapping"), ir.VerbPut, ""},
		{"patchMarker", ann("PatchMapping"), ir.VerbPatch, ""},
		{"deleteMarker", ann("DeleteMapping"), ir.VerbDelete, ""},
		{"requestMappingMethod", ann("RequestMapping", namedArg("method", "RequestMethod.DELETE")), ir.VerbDelete, ""},
		{"requestMappingArray", ann("RequestMapping", namedArg("method", "{RequestMethod.GET, RequestMethod.POST}")), ir.VerbGet, ""},
		{"requestMappingNoMethod", ann("RequestMapping", strArg(`"/x"`)), ir.VerbGet, "no method= element — verb defaulted to GET"},
		{"requestMappingGarbage", ann("RequestMapping", namedArg("method", "RequestMethod.FLY")), ir.VerbGet, "no method= element — verb defaulted to GET"},
	}
	for _, c := range cases {
		verb, note := verbOf(c.a)
		if verb != c.want || note != c.wantNote {
			t.Errorf("%s: verbOf = (%s, %q), want (%s, %q)", c.name, verb, note, c.want, c.wantNote)
		}
	}
}

func TestHTTPStatusOf(t *testing.T) {
	cases := []struct {
		name     string
		a        java.Annotation
		want     int
		wantNote string
	}{
		{"unnamed", ann("ResponseStatus", strArg("HttpStatus.NOT_FOUND")), 404, ""},
		{"valueNamed", ann("ResponseStatus", namedArg("value", "HttpStatus.CREATED")), 201, ""},
		{"codeNamed", ann("ResponseStatus", namedArg("code", "HttpStatus.GONE")), 410, ""},
		{"withReason", ann("ResponseStatus", namedArg("code", "HttpStatus.CONFLICT"), namedArg("reason", `"missing"`)), 409, ""},
		{"unknownConstant", ann("ResponseStatus", strArg("HttpStatus.FLY")), 0, `unknown HttpStatus constant "FLY" — status left undeclared`},
		{"bare", ann("ResponseStatus"), 0, "no HttpStatus element — status left undeclared"},
	}
	for _, c := range cases {
		code, note := httpStatusOf(c.a)
		if code != c.want || note != c.wantNote {
			t.Errorf("%s: httpStatusOf = (%d, %q), want (%d, %q)", c.name, code, note, c.want, c.wantNote)
		}
	}
}

func TestPathOf(t *testing.T) {
	cases := []struct {
		name string
		a    java.Annotation
		want string
	}{
		{"unnamed", ann("GetMapping", strArg(`"/books/{id}"`)), "/books/{id}"},
		{"value", ann("GetMapping", namedArg("value", `"/x"`)), "/x"},
		{"path", ann("GetMapping", namedArg("path", `"/y"`)), "/y"},
		{"arrayFirst", ann("PostMapping", strArg(`{"/x", "/y"}`)), "/x"},
		{"marker", ann("GetMapping"), ""},
	}
	for _, c := range cases {
		if got := pathOf(c.a); got != c.want {
			t.Errorf("%s: pathOf = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestBindingName(t *testing.T) {
	cases := []struct {
		name string
		a    java.Annotation
		want string
	}{
		{"unnamed", ann("PathVariable", strArg(`"bookId"`)), "bookId"},
		{"value", ann("RequestParam", namedArg("value", `"q"`)), "q"},
		{"name", ann("RequestParam", namedArg("name", `"size"`)), "size"},
		{"valueBeatsName", ann("RequestParam", namedArg("name", `"a"`), namedArg("value", `"b"`)), "b"},
		{"none", ann("PathVariable"), ""},
	}
	for _, c := range cases {
		if got := bindingName(c.a); got != c.want {
			t.Errorf("%s: bindingName = %q, want %q", c.name, got, c.want)
		}
	}
}

// param builds one synthetic adapter parameter.
func param(name, typ string, anns ...java.Annotation) java.Param {
	return java.Param{Name: name, Type: java.TypeUse{Name: typ, Base: typ}, Annotations: anns}
}

func TestBindParams(t *testing.T) {
	got := bindParams([]java.Param{
		param("id", "Long", ann("PathVariable")),
		param("q", "String", ann("RequestParam")),
		param("trace", "String", ann("RequestHeader", strArg(`"X-Trace-Id"`))),
	})
	want := []ir.Param{
		{Name: "id", In: ir.ParamInPath, Type: ir.TypeRef{Name: "Long"}},
		{Name: "q", In: ir.ParamInQuery, Type: ir.TypeRef{Name: "String"}},
		{Name: "trace", In: ir.ParamInHeader, Type: ir.TypeRef{Name: "String"}},
	}
	if !reflect.DeepEqual(got.params, want) {
		t.Errorf("params = %+v, want %+v", got.params, want)
	}
	if got.payload != nil {
		t.Errorf("payload = %+v, want nil", got.payload)
	}
	if len(got.notes) != 0 {
		t.Errorf("notes = %v, want none", got.notes)
	}
}

func TestBindParamsRequestBodySetsPayload(t *testing.T) {
	got := bindParams([]java.Param{param("book", "BookDto", ann("RequestBody"))})
	if got.payload == nil || got.payload.Name != "BookDto" {
		t.Fatalf("payload = %+v, want BookDto", got.payload)
	}
	if len(got.params) != 1 || got.params[0].In != ir.ParamInBody {
		t.Errorf("params = %+v, want one body param", got.params)
	}
}

func TestBindParamsImplicitQuery(t *testing.T) {
	got := bindParams([]java.Param{param("page", "int"), param("flag", "Boolean")})
	want := []ir.Param{
		{Name: "page", In: ir.ParamInQuery, Type: ir.TypeRef{Name: "int"}},
		{Name: "flag", In: ir.ParamInQuery, Type: ir.TypeRef{Name: "Boolean"}},
	}
	if !reflect.DeepEqual(got.params, want) {
		t.Errorf("implicit query params = %+v, want %+v", got.params, want)
	}
}

func TestBindParamsModelAttributeDropped(t *testing.T) {
	got := bindParams([]java.Param{param("filter", "BookFilter")})
	if len(got.params) != 0 {
		t.Errorf("params = %+v, want none (model-attribute binding unsupported)", got.params)
	}
	if len(got.notes) != 1 {
		t.Fatalf("notes = %v, want one", got.notes)
	}
}

func TestBindParamsValidationJoined(t *testing.T) {
	got := bindParams([]java.Param{param("size", "Integer", ann("RequestParam", namedArg("name", `"size"`)), ann("Max", strArg("100")))})
	if len(got.params) != 1 {
		t.Fatalf("params = %+v", got.params)
	}
	if got.params[0].Name != "size" {
		t.Errorf("name = %q, want size (annotation wins)", got.params[0].Name)
	}
	if got.params[0].Validation != "@Max(100)" {
		t.Errorf("validation = %q, want @Max(100)", got.params[0].Validation)
	}
}

func TestBindParamsPageablePagination(t *testing.T) {
	got := bindParams([]java.Param{param("pageable", "Pageable")})
	if got.pagination.Style != ir.PaginationPageable {
		t.Errorf("pagination = %+v, want pageable", got.pagination)
	}
	if len(got.params) != 1 || got.params[0].In != ir.ParamInQuery {
		t.Errorf("params = %+v, want Pageable bound as query", got.params)
	}
}

func TestBindParamsPageParamsCapped(t *testing.T) {
	got := bindParams([]java.Param{
		param("page", "Integer", ann("RequestParam")),
		param("size", "Integer", ann("RequestParam"), ann("Max", strArg("100"))),
	})
	if got.pagination.Style != ir.PaginationParams {
		t.Errorf("pagination style = %q, want params", got.pagination.Style)
	}
	if !got.pagination.PageSizeCapped {
		t.Errorf("PageSizeCapped = false, want true (@Max on size)")
	}
}

func TestBindParamsPageableDefault(t *testing.T) {
	got := bindParams([]java.Param{param("pageable", "Pageable", ann("PageableDefault", namedArg("size", "20")))})
	if got.pagination.Style != ir.PaginationPageable {
		t.Errorf("pagination style = %q, want pageable (@PageableDefault)", got.pagination.Style)
	}
	// @PageableDefault sets a default, not a cap.
	if got.pagination.PageSizeCapped {
		t.Errorf("PageSizeCapped = true, want false")
	}
}
