package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDownconvertSpec_31To30(t *testing.T) {
	spec := map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "Test", "version": "1.0"},
		"paths":   map[string]any{},
		"components": map[string]any{
			"schemas": map[string]any{
				"TestType": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
						"nullable_field": map[string]any{
							"type": []any{"string", "null"},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	inPath := filepath.Join(dir, "spec-in.json")
	outPath := filepath.Join(dir, "spec-out.json")

	writeJSON(t, inPath, spec)

	if err := downconvertSpec(inPath, outPath); err != nil {
		t.Fatalf("downconvertSpec: %v", err)
	}

	result := readJSON(t, outPath)

	// Check version was downconverted
	if v, _ := result["openapi"].(string); v != "3.0.3" {
		t.Errorf("openapi = %q, want 3.0.3", v)
	}

	// Check type array was converted to scalar + nullable
	schemas := result["components"].(map[string]any)["schemas"].(map[string]any)
	testType := schemas["TestType"].(map[string]any)
	props := testType["properties"].(map[string]any)

	nullableField := props["nullable_field"].(map[string]any)
	if typ, _ := nullableField["type"].(string); typ != "string" {
		t.Errorf("nullable_field.type = %v, want string", nullableField["type"])
	}
	if nullable, _ := nullableField["nullable"].(bool); !nullable {
		t.Error("nullable_field.nullable should be true")
	}

	// Non-nullable field should be unchanged
	nameField := props["name"].(map[string]any)
	if typ, _ := nameField["type"].(string); typ != "string" {
		t.Errorf("name.type = %v, want string", nameField["type"])
	}
	if _, hasNullable := nameField["nullable"]; hasNullable {
		t.Error("name should not have nullable field")
	}
}

func TestDownconvertSpec_ConstToEnum(t *testing.T) {
	spec := map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "Test", "version": "1.0"},
		"paths":   map[string]any{},
		"components": map[string]any{
			"schemas": map[string]any{
				"ActionType": map[string]any{
					"const": "ALLOW",
				},
			},
		},
	}

	dir := t.TempDir()
	inPath := filepath.Join(dir, "spec-in.json")
	outPath := filepath.Join(dir, "spec-out.json")

	writeJSON(t, inPath, spec)

	if err := downconvertSpec(inPath, outPath); err != nil {
		t.Fatalf("downconvertSpec: %v", err)
	}

	result := readJSON(t, outPath)
	schemas := result["components"].(map[string]any)["schemas"].(map[string]any)
	actionType := schemas["ActionType"].(map[string]any)

	if _, hasConst := actionType["const"]; hasConst {
		t.Error("const should have been removed")
	}
	enumVal, ok := actionType["enum"].([]any)
	if !ok || len(enumVal) != 1 || enumVal[0] != "ALLOW" {
		t.Errorf("enum = %v, want [ALLOW]", actionType["enum"])
	}
}

func TestDownconvertSpec_AlreadyV30(t *testing.T) {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "Test", "version": "1.0"},
		"paths":   map[string]any{},
	}

	dir := t.TempDir()
	inPath := filepath.Join(dir, "spec-in.json")
	outPath := filepath.Join(dir, "spec-out.json")

	writeJSON(t, inPath, spec)

	if err := downconvertSpec(inPath, outPath); err != nil {
		t.Fatalf("downconvertSpec: %v", err)
	}

	result := readJSON(t, outPath)
	if v, _ := result["openapi"].(string); v != "3.0.3" {
		t.Errorf("openapi = %q, want 3.0.3 (unchanged)", v)
	}
}

func TestDownconvertSpec_MissingInput(t *testing.T) {
	dir := t.TempDir()
	err := downconvertSpec(filepath.Join(dir, "nonexistent.json"), filepath.Join(dir, "out.json"))
	if err == nil {
		t.Error("expected error for missing input file")
	}
}

func TestWalkAndConvert_NestedTypeArrays(t *testing.T) {
	// Verify deeply nested type arrays are converted
	spec := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"type": []any{"integer", "null"},
			},
		},
	}

	walkAndConvert(spec)

	inner := spec["level1"].(map[string]any)["level2"].(map[string]any)
	if typ, _ := inner["type"].(string); typ != "integer" {
		t.Errorf("nested type = %v, want integer", inner["type"])
	}
	if nullable, _ := inner["nullable"].(bool); !nullable {
		t.Error("nested nullable should be true")
	}
}

func TestFindModuleRoot(t *testing.T) {
	// findModuleRoot walks up looking for go.mod.
	// Since we're running inside the module, it should find one.
	root := findModuleRoot()
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Errorf("findModuleRoot returned %q but go.mod not found there", root)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return result
}
