package spring

import (
	"reflect"
	"testing"
)

func TestParseOpenAPIJSON(t *testing.T) {
	data := []byte(`{
	  "openapi": "3.0.1",
	  "paths": {
	    "/api/books": {
	      "get": {"operationId": "listBooks"},
	      "post": {"operationId": "createBook"}
	    },
	    "/api/books/{id}": {
	      "get": {"operationId": "getBook"}
	    }
	  },
	  "components": {"schemas": {"BookDto": {"type": "object"}}}
	}`)
	spec, err := parseOpenAPI(data)
	if err != nil {
		t.Fatalf("parseOpenAPI: %v", err)
	}
	if op := spec.operations["GET /api/books"]; op == nil || op["operationId"] != "listBooks" {
		t.Errorf("GET /api/books = %+v", spec.operations["GET /api/books"])
	}
	if op := spec.operations["POST /api/books"]; op == nil || op["operationId"] != "createBook" {
		t.Errorf("POST /api/books = %+v", spec.operations["POST /api/books"])
	}
	if op := spec.operations["GET /api/books/{id}"]; op == nil || op["operationId"] != "getBook" {
		t.Errorf("GET /api/books/{id} = %+v", spec.operations["GET /api/books/{id}"])
	}
	if !spec.schemas["BookDto"] || len(spec.schemas) != 1 {
		t.Errorf("schemas = %+v", spec.schemas)
	}
}

func TestParseOpenAPIIgnoresNonOperationKeys(t *testing.T) {
	spec, err := parseOpenAPI([]byte(`{"paths": {"/a": {"get": {"operationId": "opA", "parameters": []}}, "x-extensions": {}}}`))
	if err != nil {
		t.Fatalf("parseOpenAPI: %v", err)
	}
	if len(spec.operations) != 1 || spec.operations["GET /a"] == nil {
		t.Errorf("operations = %+v, want only GET /a", spec.operations)
	}
}

func TestParseOpenAPIWithoutOperationID(t *testing.T) {
	// A path entry without an operationId simply contributes nothing —
	// the overlay never invents names.
	spec, err := parseOpenAPI([]byte(`{"paths": {"/a": {"get": {"responses": {}}}}}`))
	if err != nil || len(spec.operations) != 0 {
		t.Errorf("operations = %+v err = %v, want empty", spec.operations, err)
	}
}

func TestParseOpenAPIInvalid(t *testing.T) {
	if _, err := parseOpenAPI([]byte("{not an openapi doc")); err == nil {
		t.Errorf("invalid document must error (the caller turns it into a diagnostic)")
	}
}

func TestParseOpenAPIYAML(t *testing.T) {
	spec, err := parseOpenAPI([]byte("openapi: 3.0.1\npaths:\n  /api/x:\n    get:\n      operationId: opX\n"))
	if err != nil {
		t.Fatalf("parseOpenAPI yaml: %v", err)
	}
	if op := spec.operations["GET /api/x"]; op == nil || op["operationId"] != "opX" {
		t.Errorf("GET /api/x = %+v", spec.operations["GET /api/x"])
	}
}

func TestOperationSchemas(t *testing.T) {
	op := map[string]any{
		"responses": map[string]any{
			"200": map[string]any{
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"items": map[string]any{"$ref": "#/components/schemas/BookDto"},
						},
					},
				},
			},
		},
		"requestBody": map[string]any{
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/CreateBookRequest"},
				},
			},
		},
	}
	got := operationSchemas(op)
	if !reflect.DeepEqual(got, []string{"BookDto", "CreateBookRequest"}) {
		t.Errorf("operationSchemas = %v, want sorted refs", got)
	}
	if s := operationSchemas(map[string]any{}); s != nil {
		t.Errorf("operationSchemas(empty) = %v, want nil", s)
	}
}
