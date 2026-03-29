package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// downconvertSpec converts an OpenAPI 3.1.x spec to 3.0.3 for oapi-codegen compatibility.
//
// The UniFi controller serves a 3.1.0 spec but currently doesn't use 3.1-only features.
// This function handles the known differences:
//   - Version string: "3.1.0" → "3.0.3"
//   - Type arrays: ["string", "null"] → "string" + nullable: true
//   - const → enum with single value
func downconvertSpec(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("parsing spec: %w", err)
	}

	version, _ := spec["openapi"].(string)
	if !strings.HasPrefix(version, "3.1") {
		// Already 3.0.x, just copy
		return os.WriteFile(outputPath, data, 0644)
	}

	spec["openapi"] = "3.0.3"
	walkAndConvert(spec)

	out, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling converted spec: %w", err)
	}
	return os.WriteFile(outputPath, out, 0644)
}

// walkAndConvert recursively walks the spec and converts 3.1 features to 3.0.
func walkAndConvert(v any) {
	switch val := v.(type) {
	case map[string]any:
		// Convert type arrays: ["string", "null"] → type: "string", nullable: true
		if typeVal, ok := val["type"]; ok {
			if typeArr, ok := typeVal.([]any); ok {
				var nonNull string
				hasNull := false
				for _, t := range typeArr {
					s, _ := t.(string)
					if s == "null" {
						hasNull = true
					} else {
						nonNull = s
					}
				}
				if nonNull != "" {
					val["type"] = nonNull
				}
				if hasNull {
					val["nullable"] = true
				}
			}
		}

		// Convert const → enum with single value
		if constVal, ok := val["const"]; ok {
			val["enum"] = []any{constVal}
			delete(val, "const")
		}

		for _, child := range val {
			walkAndConvert(child)
		}

	case []any:
		for _, item := range val {
			walkAndConvert(item)
		}
	}
}
