package java

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
)

// fixtureRoot is relative to this package's directory: the shared corpus
// lives in the module-level testdata tree (testdata/README.md).
const fixtureRoot = "../../testdata"

// goodFixtures lists every fixture that must parse cleanly, in the order
// the canonical fixture-set test feeds them to Parse. BrokenSyntax.java is
// deliberately absent — it belongs to the diagnostics tests.
var goodFixtures = []string{
	"BookController.java",
	"PlainService.java",
	"BookDto.java",
	"BookMapper.java",
	"Page.java",
	"Nested.java",
	"Overloads.java",
	"AnnotatedPojo.java",
	"BookStatus.java",
}

func fixturePaths(names ...string) []string {
	rels := make([]string, 0, len(names))
	for _, n := range names {
		rels = append(rels, "java-adapter/"+n)
	}
	return rels
}

// parseFixtures runs the adapter over the given fixture names.
func parseFixtures(t *testing.T, names ...string) (*Result, []ir.Diagnostic) {
	t.Helper()
	return New(fixtureRoot).Parse(fixturePaths(names...))
}

// findClass locates one type declaration anywhere in the result (top level
// or nested) and fails the test when absent.
func findClass(t *testing.T, r *Result, name string) *Class {
	t.Helper()
	var walk func(cs []*Class) *Class
	walk = func(cs []*Class) *Class {
		for _, c := range cs {
			if c.Name == name {
				return c
			}
			if got := walk(c.Nested); got != nil {
				return got
			}
		}
		return nil
	}
	for _, f := range r.Files {
		if c := walk(f.Types); c != nil {
			return c
		}
	}
	t.Fatalf("class %s not found in result", name)
	return nil
}

func findCandidate(r *Result, name string) *Class {
	for _, c := range r.Candidates() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func method(t *testing.T, c *Class, name string, occurrence int) Method {
	t.Helper()
	n := -1
	for _, m := range c.Methods {
		if m.Name == name {
			n++
			if n == occurrence {
				return m
			}
		}
	}
	t.Fatalf("class %s has no occurrence %d of method %s", c.Name, occurrence, name)
	return Method{}
}

func TestLanguage(t *testing.T) {
	if got := New(t.TempDir()).Language(); got != Language {
		t.Fatalf("Language() = %q, want %q", got, Language)
	}
}

// TestParseFixtureSet pins the whole-fixture outcome: every good file is
// extracted (10 inputs → 9 files, BrokenSyntax excluded), exactly one
// diagnostic names the broken file, and nothing crashes (the test failing
// to complete at all would be the crash).
func TestParseFixtureSet(t *testing.T) {
	res, diags := parseFixtures(t, goodFixtures...)
	if len(diags) != 0 {
		t.Fatalf("clean fixture set produced diagnostics: %+v", diags)
	}
	if len(res.Files) != len(goodFixtures) {
		t.Fatalf("got %d extracted files, want %d", len(res.Files), len(goodFixtures))
	}
	for i, f := range res.Files {
		if want := "java-adapter/" + goodFixtures[i]; f.Path != want {
			t.Fatalf("Files[%d].Path = %q, want %q", i, f.Path, want)
		}
		if f.Package != "com.example.books" {
			t.Fatalf("%s: Package = %q, want com.example.books", f.Path, f.Package)
		}
		if len(f.Types) == 0 {
			t.Fatalf("%s: no top-level types extracted", f.Path)
		}
	}
}

// TestCandidateSet pins the API-candidate view: exactly the annotated
// classes, in file × declaration order, with qualifying nested classes
// after their parent's position in the walk.
func TestCandidateSet(t *testing.T) {
	res, _ := parseFixtures(t, goodFixtures...)
	var got []string
	for _, c := range res.Candidates() {
		got = append(got, c.Name)
	}
	want := []string{
		"BookController",
		"BookDto",
		"Page",
		"StaticNested",
		"PublicNested",
		"Overloads",
		"AnnotatedPojo",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("candidates = %v, want %v", got, want)
	}
}

func TestBookControllerExtraction(t *testing.T) {
	res, _ := parseFixtures(t, "BookController.java")
	c := findClass(t, res, "BookController")
	if c.Kind != KindClass {
		t.Fatalf("Kind = %q, want %q", c.Kind, KindClass)
	}
	if c.Location.File != "java-adapter/BookController.java" || c.Location.Line != 10 {
		t.Fatalf("class Location = %+v, want line 10 in BookController.java", c.Location)
	}
	if len(c.Annotations) != 2 {
		t.Fatalf("class annotations = %d, want 2", len(c.Annotations))
	}
	ra := c.Annotations[0]
	if ra.Name != "RestController" || ra.Raw != "@RestController" || len(ra.Args) != 0 {
		t.Fatalf("marker annotation wrong: %+v", ra)
	}
	rm := c.Annotations[1]
	if rm.Name != "RequestMapping" || rm.Raw != `@RequestMapping("/api/v1/books")` {
		t.Fatalf("RequestMapping raw wrong: %+v", rm)
	}
	if len(rm.Args) != 1 || rm.Args[0].Name != "" || rm.Args[0].Value != `"/api/v1/books"` {
		t.Fatalf("RequestMapping args wrong: %+v", rm.Args)
	}

	if len(c.Methods) != 4 {
		t.Fatalf("methods = %d, want 4 (list, create, get, delete)", len(c.Methods))
	}
	list := method(t, c, "list", 0)
	if list.ReturnType.Name != "ResponseEntity<List<BookDto>>" ||
		list.ReturnType.Base != "ResponseEntity" ||
		strings.Join(list.ReturnType.Args, "|") != "List<BookDto>" {
		t.Fatalf("list return type wrong: %+v", list.ReturnType)
	}
	if len(list.Params) != 0 || len(list.Annotations) != 1 || list.Annotations[0].Name != "GetMapping" {
		t.Fatalf("list method wrong: params=%d annotations=%+v", len(list.Params), list.Annotations)
	}
	create := method(t, c, "create", 0)
	if len(create.Params) != 1 || create.Params[0].Name != "book" ||
		create.Params[0].Type.Name != "BookDto" ||
		len(create.Params[0].Annotations) != 1 ||
		create.Params[0].Annotations[0].Name != "RequestBody" {
		t.Fatalf("create parameter wrong: %+v", create.Params)
	}
	get := method(t, c, "get", 0)
	if len(get.Params) != 1 || get.Params[0].Name != "id" || get.Params[0].Type.Name != "Long" ||
		get.Params[0].Annotations[0].Name != "PathVariable" {
		t.Fatalf("get parameter wrong: %+v", get.Params)
	}
	del := method(t, c, "delete", 0)
	if del.ReturnType.Name != "void" {
		t.Fatalf("delete return = %q, want void", del.ReturnType.Name)
	}
	for _, m := range c.Methods {
		if strings.Join(m.Modifiers, ",") != "public" {
			t.Fatalf("method %s modifiers = %v, want [public]", m.Name, m.Modifiers)
		}
	}
}

func TestRecordComponents(t *testing.T) {
	res, _ := parseFixtures(t, "BookDto.java")
	c := findClass(t, res, "BookDto")
	if c.Kind != KindRecord {
		t.Fatalf("Kind = %q, want %q", c.Kind, KindRecord)
	}
	ji := c.Annotations[0]
	if ji.Name != "JsonInclude" ||
		ji.Args[0].Name != "" ||
		ji.Args[0].Value != "JsonInclude.Include.NON_NULL" {
		t.Fatalf("JsonInclude annotation wrong: %+v", ji)
	}
	if len(c.Fields) != 3 {
		t.Fatalf("record components = %d, want 3", len(c.Fields))
	}
	id := c.Fields[0]
	if id.Name != "id" || id.Type.Name != "Long" || len(id.Modifiers) != 0 {
		t.Fatalf("component id wrong: %+v", id)
	}
	if id.Annotations[0].Raw != `@JsonProperty("book_id")` ||
		id.Annotations[0].Args[0].Value != `"book_id"` {
		t.Fatalf("component id annotation wrong: %+v", id.Annotations)
	}
	summary := c.Fields[2]
	if summary.Name != "summary" || summary.Type.Name != "String" {
		t.Fatalf("component summary wrong: %+v", summary)
	}
	if summary.Annotations[0].Raw != `@Size(max = 200)` ||
		summary.Annotations[0].Args[0].Name != "max" ||
		summary.Annotations[0].Args[0].Value != "200" {
		t.Fatalf("component summary annotation wrong: %+v", summary.Annotations)
	}
	if findCandidate(res, "BookDto") == nil {
		t.Fatalf("annotated record must be a candidate")
	}
}

func TestPlainServiceNeverCandidate(t *testing.T) {
	res, _ := parseFixtures(t, "PlainService.java")
	svc := findClass(t, res, "PlainService")
	if len(svc.Annotations) != 0 {
		t.Fatalf("PlainService has annotations: %+v", svc.Annotations)
	}
	if len(svc.Methods) != 2 {
		t.Fatalf("PlainService inventory methods = %d, want 2", len(svc.Methods))
	}
	if findCandidate(res, "PlainService") != nil {
		t.Fatalf("PlainService must not be a candidate")
	}
}

func TestInterfaceMembers(t *testing.T) {
	res, _ := parseFixtures(t, "BookMapper.java")
	c := findClass(t, res, "BookMapper")
	if c.Kind != KindInterface {
		t.Fatalf("Kind = %q, want %q", c.Kind, KindInterface)
	}
	if len(c.Methods) != 2 {
		t.Fatalf("interface methods = %d, want 2", len(c.Methods))
	}
	toDto := method(t, c, "toDto", 0)
	if len(toDto.Modifiers) != 0 || toDto.ReturnType.Name != "BookDto" {
		t.Fatalf("toDto wrong: %+v", toDto)
	}
	legacy := method(t, c, "legacyName", 0)
	if len(legacy.Annotations) != 1 || legacy.Annotations[0].Name != "Deprecated" {
		t.Fatalf("legacyName annotations wrong: %+v", legacy.Annotations)
	}
	if len(legacy.Params) != 1 || legacy.Params[0].Annotations[0].Name != "Deprecated" {
		t.Fatalf("legacyName parameter wrong: %+v", legacy.Params)
	}
	if findCandidate(res, "BookMapper") != nil {
		t.Fatalf("un-annotated interface must not be a candidate")
	}
}

func TestGenericExtraction(t *testing.T) {
	res, _ := parseFixtures(t, "Page.java")
	c := findClass(t, res, "Page")
	if c.TypeParams != "<T>" {
		t.Fatalf("TypeParams = %q, want <T>", c.TypeParams)
	}
	jp := c.Annotations[0]
	if jp.Name != "JsonIgnoreProperties" || jp.Args[0].Name != "ignoreUnknown" || jp.Args[0].Value != "true" {
		t.Fatalf("JsonIgnoreProperties wrong: %+v", jp)
	}
	items := c.Fields[0]
	if items.Name != "items" || items.Type.Name != "List<T>" ||
		items.Type.Base != "List" || strings.Join(items.Type.Args, "|") != "T" {
		t.Fatalf("field items wrong: %+v", items)
	}
	if strings.Join(items.Modifiers, ",") != "private" {
		t.Fatalf("items modifiers = %v, want [private]", items.Modifiers)
	}
	getItems := method(t, c, "getItems", 0)
	if getItems.ReturnType.Name != "List<T>" || getItems.ReturnType.Base != "List" {
		t.Fatalf("getItems return wrong: %+v", getItems.ReturnType)
	}
}

func TestNestedClasses(t *testing.T) {
	res, _ := parseFixtures(t, "Nested.java")
	outer := findClass(t, res, "Nested")
	if len(outer.Annotations) != 0 || len(outer.Fields) != 1 || outer.Fields[0].Name != "outerField" {
		t.Fatalf("outer class wrong: annotations=%d fields=%+v", len(outer.Annotations), outer.Fields)
	}
	if len(outer.Nested) != 3 {
		t.Fatalf("nested classes = %d, want 3", len(outer.Nested))
	}
	static_ := outer.Nested[0]
	if static_.Name != "StaticNested" || strings.Join(static_.Modifiers, ",") != "static" ||
		len(static_.Annotations) != 1 || static_.Annotations[0].Name != "Deprecated" {
		t.Fatalf("StaticNested wrong: %+v", static_)
	}
	inner := outer.Nested[1]
	if inner.Name != "Inner" || len(inner.Modifiers) != 0 {
		t.Fatalf("Inner wrong: %+v", inner)
	}
	publicNested := outer.Nested[2]
	if publicNested.Name != "PublicNested" ||
		strings.Join(publicNested.Modifiers, ",") != "public,static" {
		t.Fatalf("PublicNested wrong: %+v", publicNested)
	}
	if findCandidate(res, "Nested") != nil || findCandidate(res, "Inner") != nil {
		t.Fatalf("un-annotated classes must not be candidates")
	}
	if findCandidate(res, "StaticNested") == nil || findCandidate(res, "PublicNested") == nil {
		t.Fatalf("annotated nested classes must be candidates")
	}
}

func TestOverloads(t *testing.T) {
	res, _ := parseFixtures(t, "Overloads.java")
	c := findClass(t, res, "Overloads")
	if len(c.Methods) != 4 {
		t.Fatalf("methods = %d, want 4", len(c.Methods))
	}
	first := method(t, c, "find", 0)
	if len(first.Params) != 1 || first.Params[0].Type.Name != "Long" {
		t.Fatalf("find(Long) wrong: %+v", first.Params)
	}
	second := method(t, c, "find", 1)
	if len(second.Params) != 1 || second.Params[0].Type.Name != "String" {
		t.Fatalf("find(String) wrong: %+v", second.Params)
	}
	third := method(t, c, "find", 2)
	if len(third.Params) != 2 ||
		third.Params[0].Type.Name != "String" || third.Params[1].Type.Name != "int" {
		t.Fatalf("find(String,int) wrong: %+v", third.Params)
	}
	titles := method(t, c, "titles", 0)
	if titles.ReturnType.Name != "java.util.List<String>" ||
		titles.ReturnType.Base != "java.util.List" ||
		strings.Join(titles.ReturnType.Args, "|") != "String" {
		t.Fatalf("titles return wrong: %+v", titles.ReturnType)
	}
}

func TestFieldAnnotationsAndModifiers(t *testing.T) {
	res, _ := parseFixtures(t, "AnnotatedPojo.java")
	c := findClass(t, res, "AnnotatedPojo")
	if len(c.Fields) != 3 {
		t.Fatalf("fields = %d, want 3", len(c.Fields))
	}
	fullName := c.Fields[0]
	if fullName.Annotations[0].Raw != `@JsonProperty("full_name")` {
		t.Fatalf("fullName annotation wrong: %+v", fullName.Annotations)
	}
	limit := c.Fields[1]
	if limit.Name != "limit" || limit.Type.Name != "int" ||
		strings.Join(limit.Modifiers, ",") != "public" {
		t.Fatalf("limit field wrong: %+v", limit)
	}
	if limit.Annotations[0].Raw != "@Max(100)" ||
		limit.Annotations[0].Args[0].Value != "100" {
		t.Fatalf("limit annotation wrong: %+v", limit.Annotations)
	}
	code := c.Fields[2]
	sz := code.Annotations[0]
	if sz.Name != "List" || sz.Raw != "@Size.List({@Size(max = 10), @Size(min = 1)})" {
		t.Fatalf("code container annotation wrong: %+v", sz)
	}
	if len(sz.Args) != 1 || sz.Args[0].Value != "{@Size(max = 10), @Size(min = 1)}" {
		t.Fatalf("code annotation args wrong: %+v", sz.Args)
	}
	getter := method(t, c, "getFullName", 0)
	if strings.Join(getter.Modifiers, ",") != "public" {
		t.Fatalf("getFullName modifiers = %v, want [public]", getter.Modifiers)
	}
}

func TestEnumCapture(t *testing.T) {
	res, _ := parseFixtures(t, "BookStatus.java")
	c := findClass(t, res, "BookStatus")
	if c.Kind != KindEnum {
		t.Fatalf("Kind = %q, want %q", c.Kind, KindEnum)
	}
	if findCandidate(res, "BookStatus") != nil {
		t.Fatalf("un-annotated enum must not be a candidate")
	}
}

// TestSyntaxErrorIsolatesFiles: the broken file yields a diagnostic naming
// it, is excluded from the extraction, and every other file still parses.
func TestSyntaxErrorIsolatesFiles(t *testing.T) {
	res, diags := parseFixtures(t, "PlainService.java", "BrokenSyntax.java", "Page.java")
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %d (%+v), want exactly 1", len(diags), diags)
	}
	d := diags[0]
	if d.Location.File != "java-adapter/BrokenSyntax.java" {
		t.Fatalf("diagnostic anchored at %q, want the broken file", d.Location.File)
	}
	if !strings.Contains(d.Message, "syntax error") ||
		!strings.Contains(d.Message, "BrokenSyntax.java") {
		t.Fatalf("diagnostic message must name the file and the reason: %q", d.Message)
	}
	if d.Location.Line != 7 {
		t.Fatalf("diagnostic line = %d, want 7 (the broken method)", d.Location.Line)
	}
	if len(res.Files) != 2 {
		t.Fatalf("extracted files = %d, want 2 — broken file excluded, others keep scanning", len(res.Files))
	}
	if findClass(t, res, "Page") == nil || findClass(t, res, "PlainService") == nil {
		t.Fatalf("healthy files must survive the broken one")
	}
}

// TestParseSkipsAndGuards covers the defensive paths: non-.java files are
// silently ignored, escaping paths are refused with a diagnostic, and an
// unreadable file becomes a diagnostic, never an error or a crash.
func TestParseSkipsAndGuards(t *testing.T) {
	root := t.TempDir()
	a := New(root)

	res, diags := a.Parse([]string{"README.md", "notes.txt"})
	if len(res.Files) != 0 || len(diags) != 0 {
		t.Fatalf("non-java input must be ignored silently, got files=%d diags=%d", len(res.Files), len(diags))
	}

	res, diags = a.Parse([]string{"../escape.java"})
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "escapes the scan root") {
		t.Fatalf("escaping path must be refused with a diagnostic, got %+v", diags)
	}
	if len(res.Files) != 0 {
		t.Fatalf("escaping path must not be extracted")
	}

	res, diags = a.Parse([]string{"missing.java"})
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "missing.java") {
		t.Fatalf("unreadable file must produce a diagnostic naming it, got %+v", diags)
	}
}

// TestOversizedFileSkipped: a file above the discovery size cap is refused
// with a diagnostic and never parsed.
func TestOversizedFileSkipped(t *testing.T) {
	root := t.TempDir()
	big := filepath.Join(root, "Big.java")
	payload := strings.Repeat("// x\n", (1<<20)/5+3)
	if err := os.WriteFile(big, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	res, diags := New(root).Parse([]string{"Big.java"})
	if len(res.Files) != 0 {
		t.Fatalf("oversized file must not be extracted")
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "exceeds") {
		t.Fatalf("oversized file must produce a cap diagnostic, got %+v", diags)
	}
}

// TestColumnsAreUTF16Units pins the IR location contract (ir.go: Column
// counts UTF-16 code units, not bytes): the second parameter starts after
// three CJK characters — 9 UTF-8 bytes but 3 UTF-16 units.
func TestColumnsAreUTF16Units(t *testing.T) {
	src := "class U {\n" +
		"    void f(String 日本語, int x) {}\n" +
		"}\n"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "U.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, diags := New(root).Parse([]string{"U.java"})
	if len(diags) != 0 {
		t.Fatalf("unicode source must parse cleanly: %+v", diags)
	}
	m := findClass(t, res, "U").Methods[0]
	if len(m.Params) != 2 {
		t.Fatalf("params = %d, want 2", len(m.Params))
	}
	cjk := m.Params[0]
	if cjk.Name != "日本語" || cjk.Location.Line != 2 || cjk.Location.Column != 19 {
		t.Fatalf("CJK param location wrong: %+v (name %q)", cjk.Location, cjk.Name)
	}
	x := m.Params[1]
	if x.Name != "x" || x.Location.Line != 2 || x.Location.Column != 28 {
		t.Fatalf("param x location wrong: %+v (want UTF-16 column 28, not byte column 34)", x.Location)
	}
}

// TestParseIsDeterministic: parsing the same files twice yields
// byte-identical JSON (charter B-6 — candidate types carry no maps, so the
// marshal order is the source order).
func TestParseIsDeterministic(t *testing.T) {
	r1, d1 := parseFixtures(t, goodFixtures...)
	r2, d2 := parseFixtures(t, goodFixtures...)
	b1, err := json.Marshal(r1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(r2)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) {
		t.Fatalf("two parses of the same input differ")
	}
	if len(d1) != len(d2) {
		t.Fatalf("diagnostics differ between runs: %d vs %d", len(d1), len(d2))
	}
}

// TestVarargsParameter pins spread-parameter extraction (type + name from
// the variable_declarator, IsVarargs set).
func TestVarargsParameter(t *testing.T) {
	src := "class V {\n" +
		"    void log(String prefix, Object... rest) {}\n" +
		"}\n"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "V.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, diags := New(root).Parse([]string{"V.java"})
	if len(diags) != 0 {
		t.Fatalf("varargs source must parse cleanly: %+v", diags)
	}
	m := findClass(t, res, "V").Methods[0]
	if len(m.Params) != 2 {
		t.Fatalf("params = %d, want 2", len(m.Params))
	}
	va := m.Params[1]
	if va.Name != "rest" || !va.IsVarargs || va.Type.Name != "Object" {
		t.Fatalf("varargs param wrong: %+v", va)
	}
}
