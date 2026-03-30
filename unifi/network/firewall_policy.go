package network

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jonshaffer/go-unifi/unifi"
)

// FirewallPolicyResponse is the API response for a firewall policy.
type FirewallPolicyResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Index    int    `json:"index"` // evaluation order (read-only, managed via ordering endpoint)
	Action   struct {
		Type              string `json:"type"` // ALLOW, BLOCK, REJECT
		AllowReturnTraffic bool   `json:"allowReturnTraffic,omitempty"`
	} `json:"action"`
	Source struct {
		ZoneID        string `json:"zoneId"`
		TrafficFilter any    `json:"trafficFilter,omitempty"`
	} `json:"source"`
	Destination struct {
		ZoneID        string `json:"zoneId"`
		TrafficFilter any    `json:"trafficFilter,omitempty"`
	} `json:"destination"`
	IPProtocolScope struct {
		IPVersion string `json:"ipVersion"` // IPV4, IPV6, IPV4_AND_IPV6
	} `json:"ipProtocolScope"`
	LoggingEnabled bool `json:"loggingEnabled"`
	Metadata       struct {
		Origin string `json:"origin"` // SYSTEM_DEFINED or USER_DEFINED
	} `json:"metadata"`
}

// FirewallPolicyRequest is the API request for creating/updating a policy.
// Must NOT include id, metadata, or index (400 error).
type FirewallPolicyRequest struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Action  struct {
		Type              string `json:"type"`
		AllowReturnTraffic bool   `json:"allowReturnTraffic,omitempty"`
	} `json:"action"`
	Source struct {
		ZoneID        string `json:"zoneId"`
		TrafficFilter any    `json:"trafficFilter,omitempty"`
	} `json:"source"`
	Destination struct {
		ZoneID        string `json:"zoneId"`
		TrafficFilter any    `json:"trafficFilter,omitempty"`
	} `json:"destination"`
	IPProtocolScope struct {
		IPVersion string `json:"ipVersion"`
	} `json:"ipProtocolScope"`
	LoggingEnabled bool `json:"loggingEnabled"`
}

func (a *App) policyPath(ctx context.Context) (string, error) {
	siteID, err := a.Client.ZoneSiteID(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/integration/v1/sites/%s/firewall/policies", siteID), nil
}

// ListFirewallPolicies returns all firewall policies (paginated).
func (a *App) ListFirewallPolicies(ctx context.Context) ([]FirewallPolicyResponse, error) {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return nil, err
	}
	return unifi.ListAll[FirewallPolicyResponse](a.Client, ctx, unifi.AppNetwork, basePath)
}

// GetFirewallPolicy returns a single policy by ID.
func (a *App) GetFirewallPolicy(ctx context.Context, id string) (*FirewallPolicyResponse, error) {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s", basePath, id)
	var policy FirewallPolicyResponse
	if err := a.Client.Do(ctx, http.MethodGet, unifi.AppNetwork, path, nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// CreateFirewallPolicy creates a new policy.
func (a *App) CreateFirewallPolicy(ctx context.Context, req FirewallPolicyRequest) (*FirewallPolicyResponse, error) {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return nil, err
	}
	var policy FirewallPolicyResponse
	if err := a.Client.Do(ctx, http.MethodPost, unifi.AppNetwork, basePath, req, &policy); err != nil {
		return nil, fmt.Errorf("creating firewall policy: %w", err)
	}
	return &policy, nil
}

// UpdateFirewallPolicy updates an existing policy. PUT with body stripped of id/metadata/index.
func (a *App) UpdateFirewallPolicy(ctx context.Context, id string, req FirewallPolicyRequest) (*FirewallPolicyResponse, error) {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s", basePath, id)
	var policy FirewallPolicyResponse
	if err := a.Client.Do(ctx, http.MethodPut, unifi.AppNetwork, path, req, &policy); err != nil {
		return nil, fmt.Errorf("updating firewall policy: %w", err)
	}
	return &policy, nil
}

// DeleteFirewallPolicy deletes a policy by ID.
func (a *App) DeleteFirewallPolicy(ctx context.Context, id string) error {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s", basePath, id)
	return a.Client.Do(ctx, http.MethodDelete, unifi.AppNetwork, path, nil, nil)
}

// PolicyOrdering represents the evaluation order for a zone pair.
type PolicyOrdering struct {
	OrderedPolicyIDs []string `json:"orderedFirewallPolicyIds"`
}

// GetPolicyOrdering returns the evaluation order for policies between two zones.
func (a *App) GetPolicyOrdering(ctx context.Context, srcZoneID, dstZoneID string) (*PolicyOrdering, error) {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/ordering?sourceFirewallZoneId=%s&destinationFirewallZoneId=%s", basePath, srcZoneID, dstZoneID)
	var ordering PolicyOrdering
	if err := a.Client.Do(ctx, http.MethodGet, unifi.AppNetwork, path, nil, &ordering); err != nil {
		return nil, err
	}
	return &ordering, nil
}

// SetPolicyOrdering sets the evaluation order for policies between two zones.
func (a *App) SetPolicyOrdering(ctx context.Context, srcZoneID, dstZoneID string, ordering PolicyOrdering) error {
	basePath, err := a.policyPath(ctx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/ordering?sourceFirewallZoneId=%s&destinationFirewallZoneId=%s", basePath, srcZoneID, dstZoneID)
	return a.Client.Do(ctx, http.MethodPut, unifi.AppNetwork, path, ordering, nil)
}
