package network

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jonshaffer/go-unifi/unifi"
)

// FirewallGroup represents a named IP address or port group (legacy REST API).
// LEGACY: Migrate to Integration API when available.
type FirewallGroup struct {
	ID           string   `json:"_id,omitempty"`
	ExternalID   string   `json:"external_id,omitempty"`
	SiteID       string   `json:"site_id,omitempty"`
	Name         string   `json:"name"`
	GroupType    string   `json:"group_type"`    // "address-group" or "port-group"
	GroupMembers []string `json:"group_members"`
}

func (a *App) firewallGroupPath() string {
	return fmt.Sprintf("/api/s/%s/rest/firewallgroup", a.Site)
}

// ListFirewallGroups returns all firewall groups.
func (a *App) ListFirewallGroups(ctx context.Context) ([]FirewallGroup, error) {
	return unifi.DoLegacy[FirewallGroup](a.Client, ctx, http.MethodGet, a.firewallGroupPath(), nil)
}

// GetFirewallGroup returns a single firewall group by ID.
func (a *App) GetFirewallGroup(ctx context.Context, id string) (*FirewallGroup, error) {
	path := fmt.Sprintf("%s/%s", a.firewallGroupPath(), id)
	results, err := unifi.DoLegacy[FirewallGroup](a.Client, ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, &unifi.ErrNotFound{Message: fmt.Sprintf("firewall group %s not found", id)}
	}
	return &results[0], nil
}

// CreateFirewallGroup creates a new firewall group.
func (a *App) CreateFirewallGroup(ctx context.Context, group FirewallGroup) (*FirewallGroup, error) {
	group.ID = ""
	results, err := unifi.DoLegacy[FirewallGroup](a.Client, ctx, http.MethodPost, a.firewallGroupPath(), group)
	if err != nil {
		return nil, fmt.Errorf("creating firewall group: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("create firewall group returned no data")
	}
	return &results[0], nil
}

// UpdateFirewallGroup updates an existing firewall group. Full-object PUT required.
func (a *App) UpdateFirewallGroup(ctx context.Context, group FirewallGroup) (*FirewallGroup, error) {
	if group.ID == "" {
		return nil, fmt.Errorf("firewall group ID is required for update")
	}
	path := fmt.Sprintf("%s/%s", a.firewallGroupPath(), group.ID)
	results, err := unifi.DoLegacy[FirewallGroup](a.Client, ctx, http.MethodPut, path, group)
	if err != nil {
		return nil, fmt.Errorf("updating firewall group: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("update firewall group returned no data")
	}
	return &results[0], nil
}

// DeleteFirewallGroup deletes a firewall group by ID.
func (a *App) DeleteFirewallGroup(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/%s", a.firewallGroupPath(), id)
	_, err := unifi.DoLegacy[FirewallGroup](a.Client, ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting firewall group: %w", err)
	}
	return nil
}
