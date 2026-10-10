package spring_test

// The Scan-based end-to-end tests live in the EXTERNAL test package: the
// in-package variant would cycle (internal/discovery wires this adapter,
// so its test binaries import each other). Everything asserted here uses
// the exported adapter surface.

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/vanguard-lint/vanguard/adapters/spring"
	"github.com/vanguard-lint/vanguard/internal/discovery"
	"github.com/vanguard-lint/vanguard/internal/ir"
)

const (
	repoDir    = "../../testdata/spring-repo"
	overlayDir = "../../testdata/spring-overlay"
)

// scan runs the full discovery pipeline (walk → detect → select → parse)
// over a fixture repo and fails the test on any error-level surprise.
func scan(t *testing.T, root string) *discovery.ScanResult {
	t.Helper()
	res, err := discovery.Scan(root, discovery.ScanOptions{})
	if err != nil {
		t.Fatalf("Scan(%s): %v", root, err)
	}
	return res
}

func findService(t *testing.T, s *ir.ApiSurface, name string) *ir.Service {
	t.Helper()
	for i := range s.Services {
		if s.Services[i].Name == name {
			return &s.Services[i]
		}
	}
	t.Fatalf("no service %q in %+v", name, s.Services)
	return nil
}

func findMethod(t *testing.T, svc *ir.Service, name string) *ir.Method {
	t.Helper()
	for i := range svc.Methods {
		if svc.Methods[i].OperationName == name {
			return &svc.Methods[i]
		}
	}
	t.Fatalf("no method %q on service %q", name, svc.Name)
	return nil
}

func findType(t *testing.T, s *ir.ApiSurface, name string) *ir.Type {
	t.Helper()
	for i := range s.Types {
		if s.Types[i].Name == name {
			return &s.Types[i]
		}
	}
	t.Fatalf("no type %q in %+v", name, s.Types)
	return nil
}

func marshalSurface(t *testing.T, s *ir.ApiSurface) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		t.Fatalf("marshal surface: %v", err)
	}
	return buf.Bytes()
}

func TestDetectJavaFiles(t *testing.T) {
	hits := spring.DetectJavaFiles([]string{"pom.xml", "src/A.java", "README.md", "src/main/java/B.java"})
	if !reflect.DeepEqual(hits, []string{"src/A.java", "src/main/java/B.java"}) {
		t.Errorf("DetectJavaFiles = %v", hits)
	}
	if len(spring.DetectJavaFiles([]string{"pom.xml"})) != 0 {
		t.Errorf("spring.DetectJavaFiles(pom only) = non-empty")
	}
}

func TestLanguage(t *testing.T) {
	if spring.New(".").Language() != "java" {
		t.Errorf("Language() = %q, want java", spring.New(".").Language())
	}
}

func TestScanSpringRepo(t *testing.T) {
	res := scan(t, repoDir)

	if res.Selected != "java" {
		t.Fatalf("selected adapter = %q, want java", res.Selected)
	}
	s := res.Surface
	if s.Source.Lang != "java" || s.Source.Framework != "spring-boot" || s.Source.FrameworkVersion != "3.2.5" {
		t.Errorf("source = %+v", s.Source)
	}
	if len(s.Services) != 8 {
		t.Errorf("services = %d, want 8", len(s.Services))
	}
	if len(s.Types) != 7 {
		t.Errorf("types = %d (%v), want 7", len(s.Types), typeNames(s.Types))
	}
	if len(s.ErrorScheme.Handlers) != 3 {
		t.Errorf("error handlers = %d, want 3", len(s.ErrorScheme.Handlers))
	}
	if v := s.Validate(); len(v) != 0 {
		t.Errorf("Validate() = %v, want clean", v)
	}
	if len(s.Diagnostics) != 3 {
		t.Errorf("diagnostics = %d (%v), want 3 unresolved-annotation notes", len(s.Diagnostics), diagMessages(s))
	}
}

func typeNames(types []ir.Type) []string {
	out := make([]string, 0, len(types))
	for _, ty := range types {
		out = append(out, ty.Name)
	}
	return out
}

func diagMessages(s *ir.ApiSurface) []string {
	out := make([]string, 0, len(s.Diagnostics))
	for _, d := range s.Diagnostics {
		out = append(out, d.Message)
	}
	return out
}

func TestSpringRepoPathMerge(t *testing.T) {
	s := scan(t, repoDir).Surface
	cases := []struct {
		svc, method, verb, path string
	}{
		{"BookController", "list", "GET", "/api/v1/books"},
		{"BookController", "get", "GET", "/api/v1/books/{id}"},
		{"BookController", "create", "POST", "/api/v1/books"},
		{"BookController", "delete", "DELETE", "/api/v1/books/{id}"},
		{"BookUpdateController", "update", "PUT", "/api/v1/books/{id}"},
		{"BookUpdateController", "search", "GET", "/api/v1/books/search"},
		{"MemberController", "list", "GET", "/api/v1/members"},
		{"AdminController", "purge", "DELETE", "/api/admin"},
		{"LoanController", "checkout", "POST", "/api/v1/loans/checkout"},
		{"LoanController", "giveBack", "PUT", "/api/v1/loans/{id}/return"},
		{"HealthController", "status", "GET", "/health"},
		{"LegacyViewController", "home", "GET", "/legacy"},
		{"LegacyViewController", "book", "GET", "/legacy/books/{id}"},
		{"EdgeController", "anyVerb", "GET", "/anything"},
		{"EdgeController", "multiPath", "POST", "/x"},
	}
	for _, c := range cases {
		m := findMethod(t, findService(t, s, c.svc), c.method)
		if m.Verb != ir.Verb(c.verb) || m.Path != c.path {
			t.Errorf("%s.%s = (%s %s), want (%s %s)", c.svc, c.method, m.Verb, m.Path, c.verb, c.path)
		}
	}
}

func TestSpringRepoParamsAndPayload(t *testing.T) {
	s := scan(t, repoDir).Surface

	// create(@RequestBody CreateBookRequest request): body param + payload agree.
	create := findMethod(t, findService(t, s, "BookController"), "create")
	if create.Payload == nil || create.Payload.Name != "CreateBookRequest" {
		t.Errorf("create payload = %+v, want CreateBookRequest", create.Payload)
	}
	if len(create.Params) != 1 || create.Params[0].In != ir.ParamInBody {
		t.Errorf("create params = %+v, want one body param", create.Params)
	}

	// checkout(@PathVariable Long memberId, @PathVariable Long bookId, @RequestHeader("X-Request-Id") String requestId)
	checkout := findMethod(t, findService(t, s, "LoanController"), "checkout")
	wantIn := []ir.ParamIn{ir.ParamInPath, ir.ParamInPath, ir.ParamInHeader}
	wantName := []string{"memberId", "bookId", "X-Request-Id"}
	for i, p := range checkout.Params {
		if p.In != wantIn[i] || p.Name != wantName[i] {
			t.Errorf("checkout param %d = (%s, %s), want (%s, %s)", i, p.Name, p.In, wantName[i], wantIn[i])
		}
	}

	// search: renamed size param + @Max validation.
	search := findMethod(t, findService(t, s, "BookUpdateController"), "search")
	if len(search.Params) != 2 {
		t.Fatalf("search params = %+v, want 2", search.Params)
	}
	if search.Params[0].Name != "q" || search.Params[0].In != ir.ParamInQuery {
		t.Errorf("search param 0 = %+v", search.Params[0])
	}
	if search.Params[1].Name != "size" || search.Params[1].Validation != "@Max(100)" {
		t.Errorf("search param 1 = %+v, want name=size validation=@Max(100)", search.Params[1])
	}
}

func TestSpringRepoResponses(t *testing.T) {
	s := scan(t, repoDir).Surface

	// list() ResponseEntity<Page<BookDto>> → Page<BookDto>, collection.
	list := findMethod(t, findService(t, s, "BookController"), "list")
	if list.Response.Type.Name != "Page<BookDto>" || !list.Response.IsCollection {
		t.Errorf("list response = %+v, want Page<BookDto> collection", list.Response)
	}
	// create() @ResponseStatus(HttpStatus.CREATED) → 201.
	create := findMethod(t, findService(t, s, "BookController"), "create")
	if create.Response.StatusCode != 201 {
		t.Errorf("create status = %d, want 201", create.Response.StatusCode)
	}
	// delete() void → zero response type.
	del := findMethod(t, findService(t, s, "BookController"), "delete")
	if del.Response.Type.Name != "" {
		t.Errorf("delete response type = %q, want empty", del.Response.Type.Name)
	}
	// search() List<BookDto> → collection, no wrapper.
	search := findMethod(t, findService(t, s, "BookUpdateController"), "search")
	if search.Response.Type.Name != "List<BookDto>" || !search.Response.IsCollection {
		t.Errorf("search response = %+v, want List<BookDto> collection", search.Response)
	}
	// members() List<Member> → collection; only transport wrappers unwrap,
	// the List spelling stays, and the element drags Member into Types.
	members := findMethod(t, findService(t, s, "AdminController"), "members")
	if members.Response.Type.Name != "List<Member>" || !members.Response.Type.IsCollection {
		t.Errorf("members response = %+v, want List<Member> collection", members.Response.Type)
	}
	memberType := findType(t, s, "Member")
	if memberType.Package != "com.example.library.member" {
		t.Errorf("Member package = %q", memberType.Package)
	}
}

func TestSpringRepoPagination(t *testing.T) {
	s := scan(t, repoDir).Surface

	list := findMethod(t, findService(t, s, "BookController"), "list")
	if list.Pagination.Style != ir.PaginationPageable {
		t.Errorf("list pagination = %+v, want pageable", list.Pagination)
	}
	search := findMethod(t, findService(t, s, "BookUpdateController"), "search")
	if search.Pagination.Style != ir.PaginationParams || !search.Pagination.PageSizeCapped {
		t.Errorf("search pagination = %+v, want params + capped", search.Pagination)
	}
	get := findMethod(t, findService(t, s, "BookController"), "get")
	if get.Pagination.Style != ir.PaginationNone {
		t.Errorf("get pagination = %+v, want none", get.Pagination)
	}
}

func TestSpringRepoUnresolvedAnnotations(t *testing.T) {
	s := scan(t, repoDir).Surface
	edge := findService(t, s, "EdgeController")

	// @CustomGet is mapping-like but unknown → method skipped, diagnostic recorded.
	if m := findMethodMaybe(edge, "weird"); m != nil {
		t.Errorf("weird should not be mapped, got %+v", m)
	}
	if !hasDiag(s, "@CustomGet") {
		t.Errorf("no diagnostic mentioning @CustomGet in %v", diagMessages(s))
	}
	// @RequestMapping without method= → GET + unresolved marker kept on the method.
	anyVerb := findMethod(t, edge, "anyVerb")
	if anyVerb.Annotations["unresolved"] == "" {
		t.Errorf("anyVerb annotations = %+v, want unresolved marker", anyVerb.Annotations)
	}
	// Multi-path @PostMapping → first path kept + unresolved marker.
	multiPath := findMethod(t, edge, "multiPath")
	if multiPath.Path != "/x" || multiPath.Annotations["unresolved"] == "" {
		t.Errorf("multiPath = (%s, %+v), want /x + unresolved marker", multiPath.Path, multiPath.Annotations)
	}
	// Local @ExceptionHandler inside a controller is NOT collected (v0.1
	// limitation): only the advice's 3 handlers exist.
	if n := len(s.ErrorScheme.Handlers); n != 3 {
		t.Errorf("error handlers = %d, want 3 (local handler excluded)", n)
	}
}

func findMethodMaybe(svc *ir.Service, name string) *ir.Method {
	for i := range svc.Methods {
		if svc.Methods[i].OperationName == name {
			return &svc.Methods[i]
		}
	}
	return nil
}

func hasDiag(s *ir.ApiSurface, substr string) bool {
	for _, d := range s.Diagnostics {
		if bytes.Contains([]byte(d.Message), []byte(substr)) {
			return true
		}
	}
	return false
}

func TestSpringRepoErrorScheme(t *testing.T) {
	s := scan(t, repoDir).Surface
	h := s.ErrorScheme.Handlers
	if len(h) != 3 {
		t.Fatalf("handlers = %d, want 3", len(h))
	}
	want := []ir.ErrorHandler{
		{ExceptionType: "BookNotFoundException", ResponseType: "ErrorResponse", StatusCode: 404},
		{ExceptionType: "IllegalArgumentException", ResponseType: "ErrorResponse"},
		{ExceptionType: "Exception", ResponseType: "ErrorResponse", StatusCode: 500},
	}
	for i, w := range want {
		if h[i].ExceptionType != w.ExceptionType || h[i].ResponseType != w.ResponseType || h[i].StatusCode != w.StatusCode {
			t.Errorf("handler %d = %+v, want %+v", i, h[i], w)
		}
	}
}

func TestSpringRepoTypes(t *testing.T) {
	s := scan(t, repoDir).Surface

	book := findType(t, s, "BookDto")
	if book.Kind != ir.KindRecord || book.Package != "com.example.library.dto" {
		t.Errorf("BookDto = %+v", book)
	}
	if len(book.Fields) != 3 || book.Fields[0].Name != "id" || book.Fields[0].JSONName != "book_id" {
		t.Errorf("BookDto fields = %+v", book.Fields)
	}
	req := findType(t, s, "CreateBookRequest")
	if req.Kind != ir.KindPOJO {
		t.Errorf("CreateBookRequest kind = %q, want pojo", req.Kind)
	}
	if len(req.Fields) != 2 || req.Fields[0].Annotations["NotBlank"] != "@NotBlank" {
		t.Errorf("CreateBookRequest fields = %+v", req.Fields)
	}
	member := findType(t, s, "Member")
	if !member.IsEntity {
		t.Errorf("Member.IsEntity = false, want true (@Entity)")
	}
	stats := findType(t, s, "BookStats")
	if stats.Package != "com.example.library.controller" || len(stats.Fields) != 2 {
		t.Errorf("BookStats = %+v", stats)
	}
	loan := findType(t, s, "LoanDto")
	if loan.Fields[0].JSONName != "loan_id" {
		t.Errorf("LoanDto field 0 = %+v", loan.Fields[0])
	}
	// The @SpringBootApplication class and the exception class are not types.
	for _, absent := range []string{"LibraryApplication", "BookNotFoundException"} {
		for _, ty := range s.Types {
			if ty.Name == absent {
				t.Errorf("type %q must not be in the IR", absent)
			}
		}
	}
}

// TestSpringRepoNoOverlay pins the R1 acceptance: a repo without any
// swagger/openapi file still yields a complete IR — no overlay annotations
// anywhere (springdoc in the pom alone merges nothing).
func TestSpringRepoNoOverlay(t *testing.T) {
	s := scan(t, repoDir).Surface
	for _, svc := range s.Services {
		if _, ok := svc.Annotations["overlay"]; ok {
			t.Errorf("service %s carries an overlay marker without a spec file", svc.Name)
		}
		for _, m := range svc.Methods {
			for _, key := range []string{"operationId", "overlay", "overlaySchemas"} {
				if _, ok := m.Annotations[key]; ok {
					t.Errorf("%s.%s carries %q without a spec file", svc.Name, m.OperationName, key)
				}
			}
		}
	}
}

func TestOverlaySnapshot(t *testing.T) {
	s := scan(t, overlayDir).Surface

	list := findMethod(t, findService(t, s, "BookController"), "list")
	if list.Annotations["operationId"] != "listBooks" || list.Annotations["overlay"] != "src/main/resources/openapi.yaml" {
		t.Errorf("list annotations = %+v", list.Annotations)
	}
	create := findMethod(t, findService(t, s, "BookController"), "create")
	if _, ok := create.Annotations["operationId"]; ok {
		t.Errorf("create must not be merged (POST absent from the spec): %+v", create.Annotations)
	}

	got := marshalSurface(t, s)
	want, err := os.ReadFile(overlayDir + "/snapshot.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("IR snapshot mismatch:\n--- want (golden) ---\n%s\n--- got (scan) ---\n%s", want, got)
	}
}

func TestScanIsDeterministic(t *testing.T) {
	a := marshalSurface(t, scan(t, repoDir).Surface)
	b := marshalSurface(t, scan(t, repoDir).Surface)
	if !bytes.Equal(a, b) {
		t.Errorf("two scans of the same repo differ (charter B-6)")
	}
}

// TestParseSurfaceDirect covers the adapter's own entry point without the
// discovery pipeline (Parse is independently callable, like the stub's).
func TestParseSurfaceDirect(t *testing.T) {
	a := spring.New(repoDir)
	s, diags := a.ParseSurface([]string{
		"src/main/java/com/example/library/controller/HealthController.java",
	})
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %v, want none", diags)
	}
	if len(s.Services) != 1 || s.Services[0].Name != "HealthController" {
		t.Fatalf("services = %+v", s.Services)
	}
	// Framework from the pom next to the parsed source (the file list names
	// no pom — detection reads the walked set, direct calls get annotations).
	if s.Source.Framework != "spring-boot" {
		t.Errorf("framework = %q, want spring-boot (annotation signal)", s.Source.Framework)
	}
	if s.Source.FrameworkVersion != "" {
		t.Errorf("version = %q, want empty (no build file in the file list)", s.Source.FrameworkVersion)
	}
}

// TestParseSurfaceRefusesEscape keeps the stub's path-escape guard visible
// at the surface level too.
func TestParseSurfaceRefusesEscape(t *testing.T) {
	a := spring.New(repoDir)
	s, diags := a.ParseSurface([]string{"../escape.java"})
	if s == nil || len(s.Services) != 0 {
		t.Errorf("surface = %+v, want empty", s)
	}
	if len(diags) != 1 || !bytes.Contains([]byte(diags[0].Message), []byte("escapes")) {
		t.Errorf("diagnostics = %+v, want one escape refusal", diags)
	}
}

// TestUnrecognizedJavaFileIgnored: non-.java paths never parse.
func TestUnrecognizedJavaFileIgnored(t *testing.T) {
	a := spring.New(repoDir)
	s, diags := a.ParseSurface([]string{"pom.xml"})
	if len(s.Services) != 0 || len(diags) != 0 {
		t.Errorf("non-java parse = %+v / %v, want silence", s.Services, diags)
	}
}
