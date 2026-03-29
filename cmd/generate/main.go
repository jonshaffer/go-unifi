// Command generate converts the OpenAPI 3.1 spec to 3.0 and runs oapi-codegen.
//
// Usage:
//
//	go run ./cmd/generate/
//
// This produces unifi/network/firewall.generated.go with typed structs
// for firewall zones and policies from the Integration API spec.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	root := findModuleRoot()

	specIn := filepath.Join(root, "openapi", "network-integration.json")
	specOut := filepath.Join(root, "openapi", "network-integration-3.0.json")
	configPath := filepath.Join(root, "oapi-codegen.yaml")

	fmt.Println("Downconverting OpenAPI 3.1 → 3.0...")
	if err := downconvertSpec(specIn, specOut); err != nil {
		fmt.Fprintf(os.Stderr, "downconvert failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running oapi-codegen...")
	cmd := exec.Command("oapi-codegen", "--config", configPath, specOut)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "oapi-codegen failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done. Generated types in unifi/network/firewall.generated.go")
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintln(os.Stderr, "could not find go.mod")
			os.Exit(1)
		}
		dir = parent
	}
}
