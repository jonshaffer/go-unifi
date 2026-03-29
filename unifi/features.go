package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Capabilities represents the detected capabilities of a UniFi controller.
type Capabilities struct {
	// APIVersion is the version string from the OpenAPI spec's info.version field.
	APIVersion string

	// Endpoints maps known API endpoint paths to whether they exist in the spec.
	Endpoints map[string]bool

	// Features maps feature flag names to their enabled status.
	Features map[string]bool
}

// Well-known endpoints that the provider checks for.
const (
	EndpointFirewallZones    = "/v1/sites/{siteId}/firewall/zones"
	EndpointFirewallPolicies = "/v1/sites/{siteId}/firewall/policies"
	EndpointSites            = "/v1/sites"
)

// Well-known feature flags.
const (
	FeatureZoneBasedFirewall = "ZONE_BASED_FIREWALL"
)

// DetectCapabilities discovers what the controller supports by:
// 1. Fetching the OpenAPI spec and parsing available endpoints (FR-017 layer 1)
// 2. Querying feature flags (FR-017 layer 2)
func DetectCapabilities(c *Client, ctx context.Context) (*Capabilities, error) {
	caps := &Capabilities{
		Endpoints: make(map[string]bool),
		Features:  make(map[string]bool),
	}

	// Layer 1: OpenAPI spec introspection
	if err := detectFromOpenAPISpec(c, ctx, caps); err != nil {
		// Non-fatal: spec might not be available on older controllers
		caps.APIVersion = "unknown"
	}

	// Layer 2: Feature flag queries
	if err := detectFeatureFlags(c, ctx, caps); err != nil {
		// Non-fatal: feature flags endpoint might not exist
		_ = err
	}

	return caps, nil
}

// detectFromOpenAPISpec fetches the Integration API spec and extracts endpoint info.
func detectFromOpenAPISpec(c *Client, ctx context.Context, caps *Capabilities) error {
	var spec struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		Paths map[string]any `json:"paths"`
	}

	err := c.do(ctx, http.MethodGet, AppNetwork, "/api-docs/integration.json", nil, &spec)
	if err != nil {
		return err
	}

	caps.APIVersion = spec.Info.Version

	for path := range spec.Paths {
		caps.Endpoints[path] = true
	}

	return nil
}

// detectFeatureFlags queries the controller's feature flags.
func detectFeatureFlags(c *Client, ctx context.Context, caps *Capabilities) error {
	var resp json.RawMessage
	err := c.do(ctx, http.MethodGet, AppNetwork, "/v2/api/site/default/features", nil, &resp)
	if err != nil {
		return err
	}

	// Feature flags response can be a map or array depending on controller version.
	// Try parsing as map first.
	var flagMap map[string]bool
	if json.Unmarshal(resp, &flagMap) == nil {
		for k, v := range flagMap {
			caps.Features[k] = v
		}
		return nil
	}

	// Try as array of objects with name/enabled fields
	var flagList []struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if json.Unmarshal(resp, &flagList) == nil {
		for _, f := range flagList {
			caps.Features[f.Name] = f.Enabled
		}
	}

	return nil
}

// ValidateCapabilities checks that the controller has the required endpoints
// and features. Returns ErrVersionIncompatible with diagnostic hints if
// critical capabilities are missing (NFR-001: no hard version gate).
func ValidateCapabilities(caps *Capabilities) error {
	var missing []string

	required := []struct {
		endpoint string
		desc     string
	}{
		{EndpointSites, "Integration API sites"},
		{EndpointFirewallZones, "firewall zones"},
		{EndpointFirewallPolicies, "firewall policies"},
	}

	for _, r := range required {
		if !caps.Endpoints[r.endpoint] {
			missing = append(missing, fmt.Sprintf("%s (%s)", r.endpoint, r.desc))
		}
	}

	if len(missing) > 0 {
		return &ErrVersionIncompatible{
			Message: fmt.Sprintf(
				"Required endpoints not available: %s. "+
					"Network Application 10.x+ is typically required. "+
					"Detected API version: %s",
				strings.Join(missing, ", "),
				caps.APIVersion,
			),
			Missing: missing,
		}
	}

	return nil
}
