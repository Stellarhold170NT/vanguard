package spring

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Stellarhold170NT/vanguard/internal/ir"

	yaml "gopkg.in/yaml.v3"
)

// openAPISpec is the minimal OpenAPI 3 read the overlay merges: the
// operation objects keyed "VERB /path" and the component schema names.
// Everything else in the document is ignored — the overlay merges
// metadata, it does not lint the spec (charter §1, D-column note).
type openAPISpec struct {
	operations map[string]map[string]any
	schemas    map[string]bool
}

// parseOpenAPI reads one OpenAPI document, JSON or YAML (springdoc serves
// JSON at runtime; repos commit either spelling). It never fails the scan:
// a malformed document returns an error the caller turns into a
// diagnostic and the annotation-derived IR stays as it is.
func parseOpenAPI(data []byte) (*openAPISpec, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("not an OpenAPI document (JSON and YAML both fail): %w", err)
		}
		root = normalizeYAML(root)
	}
	m, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("document is not an object")
	}
	spec := &openAPISpec{operations: map[string]map[string]any{}, schemas: map[string]bool{}}
	paths, _ := m["paths"].(map[string]any)
	for p, v := range paths {
		pm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		for method, op := range pm {
			switch strings.ToLower(method) {
			case "get", "post", "put", "patch", "delete":
				om, ok := op.(map[string]any)
				if !ok {
					continue
				}
				// Only operations the spec fully describes (an operationId)
				// participate in the overlay: the overlay merges metadata,
				// it never invents names from half-described entries.
				if id, _ := om["operationId"].(string); id == "" {
					continue
				}
				key := strings.ToUpper(method) + " " + normalizeSpecPath(p)
				if _, exists := spec.operations[key]; !exists {
					spec.operations[key] = om
				}
			}
		}
	}
	if comps, ok := m["components"].(map[string]any); ok {
		if schemas, ok := comps["schemas"].(map[string]any); ok {
			for name := range schemas {
				spec.schemas[name] = true
			}
		}
	}
	return spec, nil
}

// normalizeSpecPath makes a spec path key comparable with the IR's merged
// paths (leading slash; OpenAPI requires one, tolerating readers anyway).
func normalizeSpecPath(p string) string {
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

// normalizeYAML converts the generic yaml.v3 tree into plain
// map[string]any / []any shapes the JSON path produces natively.
func normalizeYAML(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			t[k] = normalizeYAML(e)
		}
		return t
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[fmt.Sprint(k)] = normalizeYAML(e)
		}
		return out
	case []any:
		for i := range t {
			t[i] = normalizeYAML(t[i])
		}
		return t
	default:
		return v
	}
}

// operationSchemas extracts the schema names an operation references via
// $ref in its responses and requestBody (following items/additionalProperties
// into array and map shapes), sorted and deduplicated.
func operationSchemas(op map[string]any) []string {
	seen := map[string]bool{}
	var walkSchema func(schema any)
	walkSchema = func(schema any) {
		m, ok := schema.(map[string]any)
		if !ok {
			return
		}
		if ref, ok := m["$ref"].(string); ok {
			name := schemaNameFromRef(ref)
			if name != "" {
				seen[name] = true
			}
		}
		if items, ok := m["items"]; ok {
			walkSchema(items)
		}
		if ap, ok := m["additionalProperties"]; ok {
			walkSchema(ap)
		}
	}
	walkContent := func(container any) {
		m, ok := container.(map[string]any)
		if !ok {
			return
		}
		content, ok := m["content"].(map[string]any)
		if !ok {
			return
		}
		for _, media := range content {
			if mm, ok := media.(map[string]any); ok {
				walkSchema(mm["schema"])
			}
		}
	}
	if responses, ok := op["responses"].(map[string]any); ok {
		for _, resp := range responses {
			walkContent(resp)
		}
	}
	walkContent(op["requestBody"])
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// schemaNameFromRef reads the schema name out of "#/components/schemas/X".
func schemaNameFromRef(ref string) string {
	const marker = "components/schemas/"
	if i := strings.LastIndex(ref, marker); i >= 0 {
		return ref[i+len(marker):]
	}
	return ""
}

// applyOverlay merges the springdoc/OpenAPI overlay into the surface: for
// every method whose (verb, path) an openapi/swagger file in the walked
// set describes, the spec's operationId and schema names join the method
// annotations and the "overlay" provenance marker records the source
// file. Methods the spec does not describe stay untouched; a spec that
// cannot be parsed costs one diagnostic and nothing else.
func (a *Adapter) applyOverlay(s *ir.ApiSurface, files []string, sink *[]ir.Diagnostic) {
	for _, specFile := range findSpecFiles(files) {
		data, err := readBounded(a.root, specFile)
		if err != nil {
			*sink = append(*sink, ir.Diagnostic{
				Message:  fmt.Sprintf("unresolved overlay %s: %v — overlay skipped", specFile, err),
				Location: ir.Location{File: specFile, Line: 1, Column: 1},
			})
			continue
		}
		spec, err := parseOpenAPI(data)
		if err != nil {
			*sink = append(*sink, ir.Diagnostic{
				Message:  fmt.Sprintf("unresolved overlay %s: %v — overlay skipped", specFile, err),
				Location: ir.Location{File: specFile, Line: 1, Column: 1},
			})
			continue
		}
		for si := range s.Services {
			for mi := range s.Services[si].Methods {
				m := &s.Services[si].Methods[mi]
				key := string(m.Verb) + " " + m.Path
				op, ok := spec.operations[key]
				if !ok {
					continue
				}
				merged := false
				if id, ok := op["operationId"].(string); ok && id != "" {
					m.Annotations["operationId"] = id
					merged = true
				}
				if names := operationSchemas(op); len(names) > 0 {
					m.Annotations["overlaySchemas"] = strings.Join(names, ", ")
					merged = true
				}
				if merged {
					// Provenance marker: this method's extra metadata came
					// from the spec, not from source annotations.
					m.Annotations["overlay"] = specFile
				}
			}
		}
	}
}
