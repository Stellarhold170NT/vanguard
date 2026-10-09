package java

import (
	"strings"
	"unicode/utf8"

	"github.com/Stellarhold170NT/vanguard/internal/ir"
	sitter "github.com/smacker/go-tree-sitter"
)

// This file walks the tree-sitter Java tree in exactly one pass per file
// (the brief's performance note: no full-file re-queries). Everything the
// brief asks for — declarations, members, annotations with raw arguments —
// is collected by the single recursive descent below.

// srcLines holds one file's source and its line-start byte offsets so
// tree-sitter points (row, byte column) convert into the IR's locations:
// 1-based line, 1-based column counted in UTF-16 code units (ir.Location
// contract — SARIF declares columnKind "utf16CodeUnits").
type srcLines struct {
	src    []byte
	starts []int // byte offset of the first byte of each line
}

func newSrcLines(src []byte) *srcLines {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &srcLines{src: src, starts: starts}
}

// loc converts one node start into an ir.Location without the file name
// (the extractor fills it in).
func (s *srcLines) loc(n *sitter.Node) ir.Location {
	if n == nil {
		return ir.Location{}
	}
	row := int(n.StartPoint().Row)
	loc := ir.Location{Line: row + 1, Column: 1}
	if row >= len(s.starts) {
		return loc
	}
	lineStart := s.starts[row]
	offset := lineStart + int(n.StartPoint().Column)
	if offset > len(s.src) {
		offset = len(s.src)
	}
	loc.Column = utf16Units(s.src[lineStart:offset]) + 1
	return loc
}

// utf16Units counts the UTF-16 code units of a byte slice: one unit per
// rune below U+10000, two per rune at or above it.
func utf16Units(b []byte) int {
	units := 0
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
		i += size
	}
	return units
}

// extractor walks one parsed file into its File result.
type extractor struct {
	src *srcLines
	rel string // path as passed to Parse; stamped into every location
}

func (e *extractor) loc(n *sitter.Node) ir.Location {
	l := e.src.loc(n)
	l.File = e.rel
	return l
}

// file walks the program root: the package declaration and the top-level
// type declarations, in source order. Imports are deliberately not
// captured — no candidate needs them in v0.1 (w3-02 resolves type
// spellings, not imports).
func (e *extractor) file(root *sitter.Node) *File {
	f := &File{}
	for i := 0; i < int(root.ChildCount()); i++ {
		n := root.Child(i)
		switch n.Type() {
		case "package_declaration":
			f.Package = e.packageName(n)
		case "class_declaration", "interface_declaration", "record_declaration", "enum_declaration":
			if c := e.typeDecl(n); c != nil {
				f.Types = append(f.Types, c)
			}
		}
	}
	return f
}

// packageName reads the qualified name out of a package declaration.
func (e *extractor) packageName(n *sitter.Node) string {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == "identifier" || c.Type() == "scoped_identifier" {
			return c.Content(e.src.src)
		}
	}
	return ""
}

// typeDecl extracts one class/interface/record/enum declaration, recursing
// into its body for members and nested declarations.
func (e *extractor) typeDecl(n *sitter.Node) *Class {
	c := &Class{}
	switch n.Type() {
	case "class_declaration":
		c.Kind = KindClass
	case "interface_declaration":
		c.Kind = KindInterface
	case "record_declaration":
		c.Kind = KindRecord
	case "enum_declaration":
		c.Kind = KindEnum
	default:
		return nil
	}
	if name := n.ChildByFieldName("name"); name != nil {
		c.Name = name.Content(e.src.src)
		c.Location = e.loc(name)
	}
	c.TypeParams = nodeText(n.ChildByFieldName("type_parameters"), e.src.src)
	c.Modifiers, c.Annotations = memberModifiers(childByType(n, "modifiers"), e.src.src)
	if c.Kind == KindRecord {
		// Record components precede the body in the source and are the
		// record's exposed state — collect them as fields first.
		e.recordComponents(c, n.ChildByFieldName("parameters"))
	}
	e.members(c, n.ChildByFieldName("body"))
	return c
}

// members walks a class/interface/enum body. Enum bodies nest their
// member declarations one level down (enum_body_declarations); enum
// constants are not fields and are skipped.
func (e *extractor) members(c *Class, body *sitter.Node) {
	if body == nil {
		return
	}
	if body.Type() == "enum_body" {
		for i := 0; i < int(body.NamedChildCount()); i++ {
			if sec := body.NamedChild(i); sec.Type() == "enum_body_declarations" {
				e.members(c, sec)
			}
		}
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		m := body.NamedChild(i)
		switch m.Type() {
		case "field_declaration":
			e.fields(c, m)
		case "method_declaration": // bodyless interface methods included
			if md := e.methodDecl(m); md != nil {
				c.Methods = append(c.Methods, *md)
			}
		case "class_declaration", "interface_declaration", "record_declaration", "enum_declaration":
			if nested := e.typeDecl(m); nested != nil {
				c.Nested = append(c.Nested, nested)
			}
		}
	}
}

// recordComponents turns the record header's formal parameters into fields.
func (e *extractor) recordComponents(c *Class, params *sitter.Node) {
	if params == nil {
		return
	}
	for i := 0; i < int(params.NamedChildCount()); i++ {
		p := params.NamedChild(i)
		if p.Type() != "formal_parameter" {
			continue
		}
		name := p.ChildByFieldName("name")
		f := Field{
			Name:     nodeText(name, e.src.src),
			Type:     typeUse(p.ChildByFieldName("type"), e.src.src),
			Location: e.loc(name),
		}
		_, f.Annotations = memberModifiers(childByType(p, "modifiers"), e.src.src)
		c.Fields = append(c.Fields, f)
	}
}

// fields extracts one field_declaration; each declarator becomes a Field.
func (e *extractor) fields(c *Class, n *sitter.Node) {
	tu := typeUse(n.ChildByFieldName("type"), e.src.src)
	mods, anns := memberModifiers(childByType(n, "modifiers"), e.src.src)
	for i := 0; i < int(n.ChildCount()); i++ {
		d := n.Child(i)
		if d.Type() != "variable_declarator" {
			continue
		}
		name := d.ChildByFieldName("name")
		f := Field{
			Name:        nodeText(name, e.src.src),
			Type:        tu,
			Modifiers:   mods,
			Annotations: anns,
			Location:    e.loc(name),
		}
		// C-style declarator dimensions (int a[]) belong to the field's
		// own type spelling.
		if dims := d.ChildByFieldName("dimensions"); dims != nil {
			f.Type.Name += dims.Content(e.src.src)
		}
		c.Fields = append(c.Fields, f)
	}
}

// methodDecl extracts one method or bodyless interface method.
func (e *extractor) methodDecl(n *sitter.Node) *Method {
	name := n.ChildByFieldName("name")
	m := &Method{
		Name:       nodeText(name, e.src.src),
		ReturnType: typeUse(n.ChildByFieldName("type"), e.src.src),
		TypeParams: nodeText(n.ChildByFieldName("type_parameters"), e.src.src),
		Location:   e.loc(name),
	}
	m.Modifiers, m.Annotations = memberModifiers(childByType(n, "modifiers"), e.src.src)
	if params := n.ChildByFieldName("parameters"); params != nil {
		m.Params = e.params(params)
	}
	if dims := n.ChildByFieldName("dimensions"); dims != nil {
		m.ReturnType.Name += dims.Content(e.src.src)
	}
	return m
}

// params extracts a formal_parameters list: plain parameters, varargs
// (spread_parameter wraps its type in an unannotated-type node and names
// it through a variable_declarator) and — deliberately — nothing else;
// receiver parameters are out of the brief's surface.
func (e *extractor) params(n *sitter.Node) []Param {
	var out []Param
	for i := 0; i < int(n.NamedChildCount()); i++ {
		p := n.NamedChild(i)
		switch p.Type() {
		case "formal_parameter":
			name := p.ChildByFieldName("name")
			param := Param{
				Name:     nodeText(name, e.src.src),
				Type:     typeUse(p.ChildByFieldName("type"), e.src.src),
				Location: e.loc(name),
			}
			_, param.Annotations = memberModifiers(childByType(p, "modifiers"), e.src.src)
			out = append(out, param)
		case "spread_parameter":
			if param, ok := e.spreadParam(p); ok {
				out = append(out, param)
			}
		}
	}
	return out
}

// spreadParam decodes one varargs parameter. The grammar inlines its
// "_unannotated_type" supertype, so the concrete type node (identifier,
// generic, qualified or array type) is a direct child; the name lives in
// the variable_declarator.
func (e *extractor) spreadParam(p *sitter.Node) (Param, bool) {
	var typNode, decl *sitter.Node
	for i := 0; i < int(p.ChildCount()); i++ {
		c := p.Child(i)
		switch c.Type() {
		case "variable_declarator":
			decl = c
		case "modifiers", "comment":
			// not the type
		default:
			if c.IsNamed() && typNode == nil {
				typNode = c
			}
		}
	}
	name := nodeText(decl.ChildByFieldName("name"), e.src.src)
	if name == "" {
		return Param{}, false
	}
	return Param{
		Name:      name,
		Type:      typeUse(typNode, e.src.src),
		IsVarargs: true,
		Location:  e.loc(decl.ChildByFieldName("name")),
	}, true
}

// typeUse converts a type node into its source spelling: full Name, erasure
// Base and raw top-level type arguments for generics; the element type for
// arrays; verbatim text for everything else (identifiers, primitives,
// void, qualified names).
func typeUse(n *sitter.Node, src []byte) TypeUse {
	if n == nil {
		return TypeUse{}
	}
	tu := TypeUse{Name: n.Content(src), Base: n.Content(src)}
	switch n.Type() {
	case "generic_type":
		var base, args *sitter.Node
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			switch c.Type() {
			case "type_arguments":
				args = c
			case "type_identifier", "scoped_type_identifier", "generic_type":
				base = c
			}
		}
		if base != nil {
			tu.Base = base.Content(src)
		}
		if args != nil {
			for i := 0; i < int(args.NamedChildCount()); i++ {
				tu.Args = append(tu.Args, args.NamedChild(i).Content(src))
			}
		}
	case "array_type":
		if el := n.ChildByFieldName("element"); el != nil {
			tu.Base = el.Content(src)
		}
	}
	return tu
}

// annotationOf extracts one annotation (marker or with arguments): simple
// name, raw text, and the raw arguments in source order — named pairs keep
// their element name, everything else (expressions, arrays, nested
// annotations) is captured verbatim as the unnamed value.
func annotationOf(n *sitter.Node, src []byte) Annotation {
	a := Annotation{Raw: n.Content(src)}
	if name := n.ChildByFieldName("name"); name != nil {
		a.Name = simpleName(name.Content(src))
	}
	if n.Type() != "annotation" {
		return a // marker_annotation: no argument list
	}
	if args := n.ChildByFieldName("arguments"); args != nil {
		for i := 0; i < int(args.NamedChildCount()); i++ {
			el := args.NamedChild(i)
			if el.Type() == "element_value_pair" {
				arg := AnnotationArg{Value: nodeText(el.ChildByFieldName("value"), src)}
				if k := el.ChildByFieldName("key"); k != nil {
					arg.Name = k.Content(src)
				}
				a.Args = append(a.Args, arg)
				continue
			}
			a.Args = append(a.Args, AnnotationArg{Value: el.Content(src)})
		}
	}
	return a
}

// memberModifiers splits a modifiers node into its keyword tokens
// ("public", "static", …) and its annotations, both in source order.
// Comments (named extras) are ignored.
func memberModifiers(n *sitter.Node, src []byte) (keywords []string, anns []Annotation) {
	if n == nil {
		return nil, nil
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		switch {
		case c.Type() == "annotation" || c.Type() == "marker_annotation":
			anns = append(anns, annotationOf(c, src))
		case !c.IsNamed():
			keywords = append(keywords, c.Type())
		}
	}
	return keywords, anns
}

// simpleName returns the last dot-segment of a (possibly qualified) name.
func simpleName(q string) string {
	if i := strings.LastIndexByte(q, '.'); i >= 0 {
		return q[i+1:]
	}
	return q
}

// nodeText is a nil-safe Content.
func nodeText(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Content(src)
}

// childByType returns the first child of the given type, nil when absent.
func childByType(n *sitter.Node, typ string) *sitter.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		if c := n.Child(i); c.Type() == typ {
			return c
		}
	}
	return nil
}
