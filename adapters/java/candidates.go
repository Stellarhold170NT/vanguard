package java

import "github.com/vanguard-lint/vanguard/internal/ir"

// ClassKind classifies a type declaration the adapter extracted.
type ClassKind string

const (
	KindClass     ClassKind = "class"
	KindInterface ClassKind = "interface"
	KindRecord    ClassKind = "record"
	KindEnum      ClassKind = "enum"
)

// TypeUse is a type as written in the source. Name is the full spelling
// including generics ("ResponseEntity<List<BookDto>>"); Base is the erasure
// without type arguments ("ResponseEntity"); Args lists the raw top-level
// type arguments in source order ("List<BookDto>") — nil when the type is
// not generic. Raw spellings are kept verbatim so w3-02 can unwrap
// transport wrappers without re-parsing.
type TypeUse struct {
	Name string   `json:"name"`
	Base string   `json:"base,omitempty"`
	Args []string `json:"args,omitempty"`
}

// AnnotationArg is one argument of an annotation. Name is the element name
// ("max" for @Size(max = 200)) and "" for the single unnamed value form;
// Value is the raw source text of the value — strings keep their quotes,
// arrays keep their braces, nested annotations keep their whole text
// (@Size.List({@Size(max = 10)}) survives verbatim).
type AnnotationArg struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}

// Annotation is one declared annotation: Name is the simple name (last
// segment — "List" for the container @Size.List), Raw the full source text
// including "@" and arguments, Args the raw arguments in source order.
// The adapter attaches no meaning to annotations — deciding that
// @GetMapping means GET is w3-02's mapping work.
type Annotation struct {
	Name string          `json:"name"`
	Raw  string          `json:"raw"`
	Args []AnnotationArg `json:"args,omitempty"`
}

// Param is one declared parameter of a method: name, type as written, the
// annotations on it, and whether it is a varargs (... ) parameter.
type Param struct {
	Name        string       `json:"name"`
	Type        TypeUse      `json:"type"`
	IsVarargs   bool         `json:"isVarargs,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
	Location    ir.Location  `json:"location"`
}

// Method is one declared method (or bodyless interface method): name,
// return type, type parameters ("<T>" raw, "" when absent), modifier
// keywords in source order (public, static, …), parameters and
// annotations. Constructors are deliberately not captured — the brief's
// extraction surface is methods; a constructor that maps an endpoint is a
// w3-02 concern.
type Method struct {
	Name        string       `json:"name"`
	ReturnType  TypeUse      `json:"returnType"`
	TypeParams  string       `json:"typeParams,omitempty"`
	Modifiers   []string     `json:"modifiers,omitempty"`
	Params      []Param      `json:"params,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
	Location    ir.Location  `json:"location"`
}

// Field is one declared field or record component: name, type, modifier
// keywords (empty for record components, which carry no source modifiers)
// and annotations. A multi-declarator field (int a, b) becomes one Field
// per declarator.
type Field struct {
	Name        string       `json:"name"`
	Type        TypeUse      `json:"type"`
	Modifiers   []string     `json:"modifiers,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
	Location    ir.Location  `json:"location"`
}

// Class is one extracted type declaration: class, interface, record or
// enum, at top level or nested. Package lives on File — every declaration
// of a file shares it. Methods and Fields hold the full inventory (private
// included); the API-candidate view (Result.Candidates) applies the
// brief's public/annotated filter on top.
type Class struct {
	Name        string       `json:"name"`
	Kind        ClassKind    `json:"kind"`
	TypeParams  string       `json:"typeParams,omitempty"`
	Modifiers   []string     `json:"modifiers,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
	Methods     []Method     `json:"methods,omitempty"`
	Fields      []Field      `json:"fields,omitempty"`
	Nested      []*Class     `json:"nested,omitempty"`
	Location    ir.Location  `json:"location"`
}

// File is the extraction result of one .java source file: its path (as
// passed to Adapter.Parse, relative to the scan root), its package
// declaration ("" under the default package) and its top-level type
// declarations in source order.
type File struct {
	Path    string   `json:"path"`
	Package string   `json:"package,omitempty"`
	Types   []*Class `json:"types,omitempty"`
}

// Result is one adapter pass over a file list. Files appear in the input
// order; files that could not be extracted (syntax errors, unreadable,
// oversized) are absent — their reason is in the diagnostics Parse
// returned. The candidate types carry no maps, so marshalling a Result is
// byte-deterministic for the same input (charter B-6).
type Result struct {
	Files []*File `json:"files,omitempty"`
}

// Candidates returns the adapter's API-candidate view of the extraction:
// every type declaration that carries at least one annotation, in file ×
// declaration order with qualifying nested classes included (depth-first,
// parent before its nested classes). Per the brief the members are the
// public ones — methods and fields whose modifiers spell "public" in the
// source. Record components are the record's exposed state (they carry no
// source modifier) and always count as public. Un-annotated classes stay
// out of the view but remain in the Files inventory — w3-02 extracts DTO
// records and POJOs that no annotation marks.
//
// The returned Class values are filtered copies: mutating them never
// affects the inventory, and their Nested list is empty (candidates are a
// flat enumeration; walk Result.Files for hierarchy).
func (r *Result) Candidates() []*Class {
	var out []*Class
	for _, f := range r.Files {
		collectCandidates(f.Types, &out)
	}
	return out
}

func collectCandidates(cs []*Class, out *[]*Class) {
	for _, c := range cs {
		if len(c.Annotations) > 0 {
			*out = append(*out, c.candidateView())
		}
		collectCandidates(c.Nested, out)
	}
}

// candidateView copies one class and keeps only the candidate members.
func (c *Class) candidateView() *Class {
	view := *c
	view.Methods = nil
	view.Fields = nil
	view.Nested = nil
	for _, m := range c.Methods {
		if isPublic(m.Modifiers) {
			view.Methods = append(view.Methods, m)
		}
	}
	for _, f := range c.Fields {
		if c.Kind == KindRecord || isPublic(f.Modifiers) {
			view.Fields = append(view.Fields, f)
		}
	}
	return &view
}

func isPublic(modifiers []string) bool {
	for _, m := range modifiers {
		if m == "public" {
			return true
		}
	}
	return false
}
