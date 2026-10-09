package rules

import (
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/engine"
	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// The R3xx family tests (w3-05): pagination shape rules over hand-built IR
// surfaces, following the demo-set pattern — the checks read ONLY the IR
// (Method/Param nodes plus ctx.Surface), never source files.

// r3xxController is the controller shell every case mounts its method on.
func r3xxController(methods ...ir.Method) *ir.ApiSurface {
	for i := range methods {
		if methods[i].Location == (ir.Location{}) {
			methods[i].Location = ir.Location{File: "src/OrderController.java", Line: 10 + i, Column: 10}
		}
		if methods[i].Pagination == (ir.Pagination{}) {
			// Adapters always state the shape explicitly (w2-02 contract);
			// tests spell none where the source has no paging input.
			methods[i].Pagination = ir.Pagination{Style: ir.PaginationNone, Location: methods[i].Location}
		}
	}
	return &ir.ApiSurface{
		Source: ir.Source{Lang: "java", Framework: "spring-boot"},
		Services: []ir.Service{{
			Name:     "OrderController",
			BasePath: "/api/v1/orders",
			Location: ir.Location{File: "src/OrderController.java", Line: 8, Column: 7},
			Methods:  methods,
		}},
	}
}

func listMethod(name, path string) ir.Method {
	return ir.Method{
		OperationName: name,
		Verb:          ir.VerbGet,
		Path:          path,
		Response: ir.Response{
			Type:         ir.TypeRef{Name: "List<OrderResponse>", IsCollection: true},
			IsCollection: true,
			Location:     ir.Location{File: "src/OrderController.java", Line: 11, Column: 19},
		},
	}
}

func runPaginationRules(t *testing.T, surface *ir.ApiSurface) []engine.Finding {
	t.Helper()
	rs, err := PaginationRules()
	if err != nil {
		t.Fatalf("PaginationRules: %v", err)
	}
	reg := NewRegistry()
	for _, r := range rs {
		if err := reg.Register(r); err != nil {
			t.Fatalf("register %s: %v", r.ID, err)
		}
	}
	report, err := engine.NewLinter(reg).Run(surface, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return report.Findings
}

// TestR3xxRulesFromData pins the data-driven identity of the family: every
// member comes from an embedded YAML record and passes full validation.
func TestR3xxRulesFromData(t *testing.T) {
	rs, err := PaginationRules()
	if err != nil {
		t.Fatalf("PaginationRules: %v", err)
	}
	want := map[string]engine.Severity{
		"R3xx-01": engine.SeverityWarn,
		"R3xx-02": engine.SeverityInfo,
		"R3xx-03": engine.SeverityInfo,
	}
	if len(rs) != len(want) {
		t.Fatalf("rules = %+v, want %d members", rs, len(want))
	}
	seen := map[string]bool{}
	for _, r := range rs {
		if r.Severity != want[r.ID] {
			t.Errorf("%s severity = %s, want %s", r.ID, r.Severity, want[r.ID])
		}
		if r.Category != "pagination" {
			t.Errorf("%s category = %q, want pagination", r.ID, r.Category)
		}
		if r.Slug == "" || r.DocPath == "" || r.Summary == "" || r.ExampleGood == "" || r.ExampleBad == "" {
			t.Errorf("%s metadata incomplete: %+v", r.ID, r)
		}
		seen[r.ID] = true
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("family missing %s", id)
		}
	}
}

// TestR3xx01ListPaginated: an unpaginated GET collection endpoint is a WARN
// anchored on the method; every paginated / non-list shape is silent.
func TestR3xx01ListPaginated(t *testing.T) {
	tests := []struct {
		name   string
		method ir.Method
		want   bool
	}{
		{
			name:   "unpaginated GET collection fires",
			method: listMethod("listOrders", "/api/v1/orders"),
			want:   true,
		},
		{
			name: "Pageable style is silent",
			method: func() ir.Method {
				m := listMethod("pageOrders", "/api/v1/orders")
				m.Pagination = ir.Pagination{Style: ir.PaginationPageable}
				return m
			}(),
			want: false,
		},
		{
			name: "page/size params style is silent",
			method: func() ir.Method {
				m := listMethod("pageOrders", "/api/v1/orders")
				m.Pagination = ir.Pagination{Style: ir.PaginationParams}
				return m
			}(),
			want: false,
		},
		{
			name: "item path /{id} is not a list endpoint",
			method: func() ir.Method {
				m := listMethod("getOrder", "/api/v1/orders/{id}")
				m.Response.IsCollection = false
				m.Response.Type = ir.TypeRef{Name: "OrderResponse"}
				return m
			}(),
			want: false,
		},
		{
			name: "non-GET verb is out of scope",
			method: func() ir.Method {
				m := listMethod("exportOrders", "/api/v1/orders/export")
				m.Verb = ir.VerbPost
				return m
			}(),
			want: false,
		},
		{
			name: "scalar response is out of scope",
			method: func() ir.Method {
				m := listMethod("countOrders", "/api/v1/orders/count")
				m.Response = ir.Response{Type: ir.TypeRef{Name: "long"}, Location: m.Response.Location}
				return m
			}(),
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := runPaginationRules(t, r3xxController(tc.method))
			var got []engine.Finding
			for _, f := range findings {
				if f.RuleID == "R3xx-01" {
					got = append(got, f)
				}
			}
			if !tc.want {
				if len(got) != 0 {
					t.Fatalf("R3xx-01 fired on negative shape: %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("R3xx-01 findings = %+v, want exactly 1", got)
			}
			f := got[0]
			if f.Severity != engine.SeverityWarn {
				t.Errorf("severity = %s, want WARN (engine-stamped)", f.Severity)
			}
			if !strings.Contains(f.Message, "unpaginated") || !strings.Contains(f.Message, "/api/v1/orders") {
				t.Errorf("message %q should name the endpoint and the gap (reason first)", f.Message)
			}
			if !strings.Contains(f.Suggestion, "Pageable") || !strings.Contains(f.Suggestion, "page/size") {
				t.Errorf("suggestion %q should name both pagination spellings", f.Suggestion)
			}
		})
	}
}

// TestR3xx02ListEnvelope: a bare java.util collection container as the
// response of a collection endpoint is INFO; Spring Page/Slice and custom
// envelope records already carry metadata and stay silent.
func TestR3xx02ListEnvelope(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{"bare List fires", "List<OrderResponse>", true},
		{"bare Set fires", "Set<OrderResponse>", true},
		{"array spelling fires", "OrderResponse[]", true},
		{"Spring Page is an envelope", "Page<OrderResponse>", false},
		{"Spring Slice is an envelope", "Slice<OrderResponse>", false},
		{"custom envelope record", "OrderPageResponse", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := listMethod("listOrders", "/api/v1/orders")
			m.Response.Type = ir.TypeRef{Name: tc.typeName}
			findings := runPaginationRules(t, r3xxController(m))
			var got []engine.Finding
			for _, f := range findings {
				if f.RuleID == "R3xx-02" {
					got = append(got, f)
				}
			}
			if tc.want != (len(got) == 1) {
				t.Fatalf("R3xx-02 on %q: findings = %+v, want fired=%v", tc.typeName, got, tc.want)
			}
			if tc.want && (!strings.Contains(got[0].Message, tc.typeName) || !strings.Contains(got[0].Suggestion, "metadata")) {
				t.Errorf("message/suggestion should name the bare type and the envelope fix: %+v", got[0])
			}
		})
	}
}

// TestR3xx03UnboundedPageSize: a size/limit query parameter without an
// @Max-style constraint is INFO anchored on the PARAM; capped params and
// other parameter names stay silent.
func TestR3xx03UnboundedPageSize(t *testing.T) {
	paramLoc := ir.Location{File: "src/OrderController.java", Line: 14, Column: 44}
	tests := []struct {
		name       string
		paramName  string
		validation string
		want       bool
	}{
		{"size without cap fires", "size", "", true},
		{"limit without cap fires", "limit", "", true},
		{"pageSize without cap fires", "pageSize", "", true},
		{"size with @Max is silent", "size", "@Max(100)", false},
		{"size with @Min only still fires", "size", "@Min(1)", true},
		{"page param is not size-carrying", "page", "", false},
		{"offset param is not size-carrying", "offset", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := listMethod("searchOrders", "/api/v1/orders")
			m.Pagination = ir.Pagination{Style: ir.PaginationParams}
			m.Params = []ir.Param{{
				Name:       tc.paramName,
				In:         ir.ParamInQuery,
				Type:       ir.TypeRef{Name: "int"},
				Validation: tc.validation,
				Location:   paramLoc,
			}}
			findings := runPaginationRules(t, r3xxController(m))
			var got []engine.Finding
			for _, f := range findings {
				if f.RuleID == "R3xx-03" {
					got = append(got, f)
				}
			}
			if tc.want != (len(got) == 1) {
				t.Fatalf("R3xx-03 on %s (%q): findings = %+v, want fired=%v", tc.paramName, tc.validation, got, tc.want)
			}
			if tc.want {
				f := got[0]
				if f.Location != paramLoc {
					t.Errorf("anchor = %+v, want the parameter itself %+v", f.Location, paramLoc)
				}
				if !strings.Contains(f.Message, tc.paramName) || !strings.Contains(f.Suggestion, "@Max") {
					t.Errorf("message/suggestion should name the param and the @Max fix: %+v", f)
				}
			}
		})
	}
}

// TestR3xx03IgnoresNonQueryBinding pins the scope boundary: the cap rule
// reads query parameters only (a header named size is not page sizing).
func TestR3xx03IgnoresNonQueryBinding(t *testing.T) {
	m := listMethod("searchOrders", "/api/v1/orders")
	m.Params = []ir.Param{{
		Name:     "size",
		In:       ir.ParamInHeader,
		Type:     ir.TypeRef{Name: "int"},
		Location: paramLocHelper(),
	}}
	findings := runPaginationRules(t, r3xxController(m))
	for _, f := range findings {
		if f.RuleID == "R3xx-03" {
			t.Fatalf("R3xx-03 fired on a header param: %+v", f)
		}
	}
}

func paramLocHelper() ir.Location {
	return ir.Location{File: "src/OrderController.java", Line: 14, Column: 44}
}
