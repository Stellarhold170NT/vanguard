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
	ops, schemas, err := parseOpenAPI(data)
	if err != nil {
		t.Fatalf("parseOpenAPI: %v", err)
	}
	if ops["GET /api/books"] != "listBooks" || ops["POST /api/books"] != "createBook" || ops["GET /api/books/{id}"] != "getBook" {
		t.Errorf("operations = %+v", ops)
	}
	if !schemas["BookDto"] || len(schemas) != 1 {
		t.Errorf("schemas = %+v", schemas)
	}
}

func TestParseOpenAPIIgnoresNonOperationKeys(t *testing.T) {
	data := []byte(`{"paths": {"/a": {"get": {"operationId": "opA", "parameters": []}}, "x-extensions": {}}}`)
	ops, _, err := parseOpenAPI(data)
	if err != nil {
		t.Fatalf("parseOpenAPI: %v", err)
	}
	if len(ops) != 1 || ops["GET /a"] != "opA" {
		t.Errorf("operations = %+v, want only GET /a", ops)
	}
}

func TestParseOpenAPIWithoutOperationID(t *testing.T) {
	// A path entry without an operationId simply contributes nothing —
	// the overlay never invents names.
	ops, _, err := parseOpenAPI([]byte(`{"paths": {"/a": {"get": {"responses": {}}}}}`))
	if err != nil || len(ops) != 0 {
		t.Errorf("operations = %+v err = %v, want empty", ops, err)
	}
}

func TestParseOpenAPIInvalid(t *testing.T) {
	if _, _, err := parseOpenAPI([]byte("{not an openapi doc")); err == nil {
		t.Errorf("invalid JSON must error (the caller turns it into a diagnostic)")
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
