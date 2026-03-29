package unifi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectCapabilities(t *testing.T) {
	mux := http.NewServeMux()

	// Mock OpenAPI spec endpoint
	mux.HandleFunc("/proxy/network/api-docs/integration.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"openapi": "3.1.0",
			"info":    map[string]string{"version": "10.1.89"},
			"paths": map[string]any{
				"/v1/sites":                                   map[string]any{},
				"/v1/sites/{siteId}/firewall/zones":            map[string]any{},
				"/v1/sites/{siteId}/firewall/policies":         map[string]any{},
			},
		})
	})

	// Mock feature flags endpoint
	mux.HandleFunc("/proxy/network/v2/api/site/default/features", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{
			"ZONE_BASED_FIREWALL": true,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "key"})
	caps, err := DetectCapabilities(c, context.Background())
	if err != nil {
		t.Fatalf("DetectCapabilities: %v", err)
	}

	if caps.APIVersion != "10.1.89" {
		t.Errorf("APIVersion = %q, want %q", caps.APIVersion, "10.1.89")
	}
	if !caps.Endpoints[EndpointFirewallZones] {
		t.Error("expected firewall zones endpoint to be detected")
	}
	if !caps.Endpoints[EndpointFirewallPolicies] {
		t.Error("expected firewall policies endpoint to be detected")
	}
	if !caps.Features[FeatureZoneBasedFirewall] {
		t.Error("expected ZONE_BASED_FIREWALL feature flag")
	}
}

func TestValidateCapabilities_AllPresent(t *testing.T) {
	caps := &Capabilities{
		APIVersion: "10.1.89",
		Endpoints: map[string]bool{
			EndpointSites:            true,
			EndpointFirewallZones:    true,
			EndpointFirewallPolicies: true,
		},
	}
	if err := ValidateCapabilities(caps); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateCapabilities_MissingEndpoint(t *testing.T) {
	caps := &Capabilities{
		APIVersion: "9.0.0",
		Endpoints: map[string]bool{
			EndpointSites: true,
			// missing zones and policies
		},
	}
	err := ValidateCapabilities(caps)
	if err == nil {
		t.Fatal("expected error for missing endpoints")
	}
	verr, ok := err.(*ErrVersionIncompatible)
	if !ok {
		t.Fatalf("expected ErrVersionIncompatible, got %T", err)
	}
	if len(verr.Missing) != 2 {
		t.Errorf("expected 2 missing endpoints, got %d", len(verr.Missing))
	}
}
