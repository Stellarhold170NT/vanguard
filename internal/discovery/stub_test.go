package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanguard-lint/vanguard/internal/ir"
)

// validStub is a complete, valid stub document: 3 methods (GET with path
// param, POST with payload+body param, GET with page/size params), 2 types.
const validStub = `{
  "service": "Orders",
  "basePath": "/v1/orders",
  "framework": "stub-http",
  "methods": [
    {"name": "getOrder", "verb": "GET", "path": "/{id}",
     "params": [{"name": "id", "in": "path", "type": "string"}],
     "response": {"type": "Order", "statusCode": 200}},
    {"name": "createOrder", "verb": "POST", "path": "",
     "params": [{"name": "req", "in": "body", "type": "CreateOrderRequest"}],
     "payload": "CreateOrderRequest",
     "response": {"type": "Order", "statusCode": 201}},
    {"name": "listOrders", "verb": "GET", "path": "",
     "params": [{"name": "page", "in": "query", "type": "int32"},
                {"name": "size", "in": "query", "type": "int32", "validation": "@Max(100)"}],
     "response": {"type": "OrderList", "statusCode": 200, "isCollection": true},
     "pagination": "params"}
  ],
  "types": [
    {"name": "Order", "kind": "record", "fields": [
      {"name": "id", "type": "string"},
      {"name": "totalAmount", "jsonName": "total_amount", "type": "int64"}]},
    {"name": "CreateOrderRequest", "kind": "pojo", "fields": [
      {"name": "buyer", "type": "string"}]}
  ]
}`

// stubTree writes stub files under a temp root and returns (root, adapter).
func stubTree(t *testing.T, files map[string]string) (*StubAdapter, string) {
	t.Helper()
	root := writeTree(t, files)
	return NewStubAdapter(root), root
}

// TestStubDetect: the stub claims exactly the *.stub.json files and its
// evidence names them (verbose debuggability, charter §5.3).
func TestStubDetect(t *testing.T) {
	a, _ := stubTree(t, nil)
	ok, ev := a.Detect([]string{"main.go", "README.md"})
	if ok || ev.Reason == "" {
		t.Fatalf("detect = %v with evidence %+v, want false with a reason", ok, ev)
	}
	ok, ev = a.Detect([]string{"api/orders.stub.json", "main.go"})
	if !ok || len(ev.Details) != 1 || ev.Details[0] != "api/orders.stub.json" {
		t.Fatalf("detect = %v, evidence %+v, want the stub file listed", ok, ev)
	}
}

// TestStubParseBuildsValidIR — the core stub test: one stub file becomes a
// valid ir.ApiSurface (services, methods, params, payload, response,
// pagination, types), locations are file-level, and the surface passes the
// w2-02 structural validator.
func TestStubParseBuildsValidIR(t *testing.T) {
	a, _ := stubTree(t, map[string]string{"api/orders.stub.json": validStub})
	surface, diags := a.Parse([]string{"api/orders.stub.json", "main.go"})
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %+v, want none", diags)
	}
	if surface.Source != (ir.Source{Lang: StubLanguage, Framework: "stub-http"}) {
		t.Fatalf("source = %+v", surface.Source)
	}
	if len(surface.Services) != 1 {
		t.Fatalf("services = %+v", surface.Services)
	}
	svc := surface.Services[0]
	if svc.Name != "Orders" || svc.BasePath != "/v1/orders" {
		t.Fatalf("service = %+v", svc)
	}
	if len(svc.Methods) != 3 {
		t.Fatalf("methods = %+v", svc.Methods)
	}

	get := svc.Methods[0]
	if get.Verb != ir.VerbGet || get.Path != "/v1/orders/{id}" || len(get.Params) != 1 {
		t.Fatalf("getOrder = %+v", get)
	}
	if get.Params[0].In != ir.ParamInPath || get.Params[0].Type.Name != "string" {
		t.Fatalf("get params = %+v", get.Params)
	}

	post := svc.Methods[1]
	if post.Verb != ir.VerbPost || post.Path != "/v1/orders" {
		t.Fatalf("post = %+v", post)
	}
	if post.Payload == nil || post.Payload.Name != "CreateOrderRequest" {
		t.Fatalf("post payload = %+v", post.Payload)
	}
	if post.Response.StatusCode != 201 || post.Response.Type.Name != "Order" {
		t.Fatalf("post response = %+v", post.Response)
	}

	list := svc.Methods[2]
	if list.Pagination.Style != ir.PaginationParams {
		t.Fatalf("list pagination = %+v", list.Pagination)
	}
	if list.Params[1].Validation != "@Max(100)" {
		t.Fatalf("size param = %+v", list.Params[1])
	}
	if !list.Response.IsCollection || list.Response.Type.Name != "OrderList" {
		t.Fatalf("list response = %+v", list.Response)
	}

	if len(surface.Types) != 2 {
		t.Fatalf("types = %+v", surface.Types)
	}
	order := surface.Types[0]
	if order.Kind != ir.KindRecord || len(order.Fields) != 2 || order.Fields[1].JSONName != "total_amount" {
		t.Fatalf("Order type = %+v", order)
	}

	// Locations are file-level (documented stub convention).
	if get.Location.File != "api/orders.stub.json" || get.Location.Line != 1 || get.Location.Column != 1 {
		t.Fatalf("location = %+v, want file-level line 1 col 1", get.Location)
	}

	// The w2-02 structural contract must hold for adapter output.
	if v := surface.Validate(); len(v) != 0 {
		t.Fatalf("stub surface fails ir.Validate: %v", v)
	}
}

// TestStubParseBestEffort — the charter §5.3 core: every broken input
// becomes a diagnostic naming its file; the healthy files still parse; the
// method never errors or aborts.
func TestStubParseBestEffort(t *testing.T) {
	a, _ := stubTree(t, map[string]string{
		"good.stub.json":       validStub,
		"bad-json.stub.json":   `{"service": "Broken",`, // truncated JSON
		"no-service.stub.json": `{"methods": []}`,       // no service name
		"bad-verb.stub.json":   `{"service": "V", "methods": [{"name": "m", "verb": "FETCH"}]}`,
		"bad-param.stub.json":  `{"service": "P", "methods": [{"name": "m", "verb": "GET", "path": "/", "params": [{"name": "c", "in": "cookie", "type": "string"}]}]}`,
		"bad-page.stub.json":   `{"service": "Q", "methods": [{"name": "m", "verb": "GET", "path": "/", "pagination": "cursor"}]}`,
		"bad-kind.stub.json":   `{"service": "K", "methods": [], "types": [{"name": "T", "kind": "singleton"}]}`,
	})
	files := []string{
		"good.stub.json", "bad-json.stub.json", "no-service.stub.json",
		"bad-verb.stub.json", "bad-param.stub.json", "bad-page.stub.json",
		"bad-kind.stub.json",
	}
	surface, diags := a.Parse(files)

	// good parses fully; V/P/Q/K keep their (valid) service shells with
	// their bad methods/types dropped; bad-json and no-service contribute
	// nothing.
	if len(surface.Services) != 5 {
		t.Fatalf("services = %+v, want 5 (Orders, V, P, Q, K)", surface.Services)
	}
	svcNames := map[string]int{}
	for _, s := range surface.Services {
		svcNames[s.Name] = len(s.Methods)
	}
	if n, ok := svcNames["Orders"]; !ok || n != 3 {
		t.Fatalf("Orders missing or wrong: %+v", svcNames)
	}
	if n, ok := svcNames["V"]; !ok || n != 0 {
		t.Fatalf("bad-verb service = %+v, want shell with 0 methods", svcNames)
	}
	if _, ok := svcNames["Broken"]; ok {
		t.Error("bad-json must not produce a service")
	}
	if _, ok := svcNames["ShouldNotExist"]; ok {
		t.Error("no-service must not produce a service")
	}

	byFile := map[string]int{}
	for _, d := range diags {
		byFile[d.Location.File]++
	}
	for _, f := range []string{"bad-json.stub.json", "no-service.stub.json", "bad-verb.stub.json", "bad-param.stub.json", "bad-page.stub.json", "bad-kind.stub.json"} {
		if byFile[f] == 0 {
			t.Errorf("no diagnostic for %s in %+v", f, diags)
		}
	}
	if byFile["good.stub.json"] != 0 {
		t.Errorf("good file diagnosed: %+v", diags)
	}
}

// TestStubParseRejectsEscapePath — security groundwork: a path leaving the
// scan root is refused before any file open (w4-05 audits again).
func TestStubParseRejectsEscapePath(t *testing.T) {
	a, _ := stubTree(t, map[string]string{"outside.stub.json": validStub})
	surface, diags := a.Parse([]string{"../outside.stub.json"})
	if len(surface.Services) != 0 {
		t.Fatalf("services = %+v, want none", surface.Services)
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "escapes") {
		t.Fatalf("diagnostics = %+v, want an escape refusal", diags)
	}
}

// TestStubParseIgnoresForeignFiles: Parse never claims what Detect did not.
func TestStubParseIgnoresForeignFiles(t *testing.T) {
	a, _ := stubTree(t, map[string]string{"main.go": "package main"})
	surface, diags := a.Parse([]string{"main.go", "README.md"})
	if len(surface.Services) != 0 || len(surface.Types) != 0 || len(diags) != 0 {
		t.Fatalf("surface %+v diags %v, want all empty", surface, diags)
	}
}

// TestStubParseReadCap: a stub file above the read cap is never loaded into
// memory — one diagnostic instead (the brief's memory guard).
func TestStubParseReadCap(t *testing.T) {
	a, root := stubTree(t, nil)
	big := filepath.Join(root, "big.stub.json")
	if err := os.WriteFile(big, make([]byte, StubMaxFileBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	surface, diags := a.Parse([]string{"big.stub.json"})
	if len(surface.Services) != 0 {
		t.Fatalf("services = %+v, want none", surface.Services)
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "cap") {
		t.Fatalf("diagnostics = %+v, want a read-cap diagnostic", diags)
	}
}

// TestStubParseUnreadableFile: a missing file (deleted between walk and
// parse) is a diagnostic, not an error.
func TestStubParseUnreadableFile(t *testing.T) {
	a, _ := stubTree(t, nil)
	surface, diags := a.Parse([]string{"ghost.stub.json"})
	if len(surface.Services) != 0 || len(diags) != 1 {
		t.Fatalf("surface %+v diags %+v, want one diagnostic", surface, diags)
	}
}
