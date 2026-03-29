package network

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jonshaffer/go-unifi/unifi"
)

// Network represents a VLAN/subnet configuration (legacy REST API).
// LEGACY: Migrate to Integration API when available.
type Network struct {
	ID                      string `json:"_id,omitempty"`
	ExternalID              string `json:"external_id,omitempty"`
	Name                    string `json:"name"`
	Purpose                 string `json:"purpose"`
	VLAN                    int    `json:"vlan"`
	VLANEnabled             bool   `json:"vlan_enabled"`
	IPSubnet                string `json:"ip_subnet"`
	DhcpdEnabled            bool   `json:"dhcpd_enabled"`
	DhcpdStart              string `json:"dhcpd_start,omitempty"`
	DhcpdStop               string `json:"dhcpd_stop,omitempty"`
	MdnsEnabled             bool   `json:"mdns_enabled"`
	InternetAccessEnabled   bool   `json:"internet_access_enabled"`
	NetworkIsolationEnabled bool   `json:"network_isolation_enabled"`
	// Additional fields from API response (preserved for full-object PUT)
	SiteID        string `json:"site_id,omitempty"`
	NetworkGroup  string `json:"networkgroup,omitempty"`
	DomainName    string `json:"domain_name,omitempty"`
	Enabled       bool   `json:"enabled,omitempty"`
	IsNAT         bool   `json:"is_nat,omitempty"`
	AttrNoDelete  bool   `json:"attr_no_delete,omitempty"`
	AttrHiddenID  string `json:"attr_hidden_id,omitempty"`
}

func (a *App) networkPath() string {
	return fmt.Sprintf("/api/s/%s/rest/networkconf", a.Site)
}

// ListNetworks returns all networks.
func (a *App) ListNetworks(ctx context.Context) ([]Network, error) {
	return unifi.DoLegacy[Network](a.Client, ctx, http.MethodGet, a.networkPath(), nil)
}

// GetNetwork returns a single network by ID.
func (a *App) GetNetwork(ctx context.Context, id string) (*Network, error) {
	path := fmt.Sprintf("%s/%s", a.networkPath(), id)
	results, err := unifi.DoLegacy[Network](a.Client, ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, &unifi.ErrNotFound{Message: fmt.Sprintf("network %s not found", id)}
	}
	return &results[0], nil
}

// CreateNetwork creates a new network. Risky on production — creates a VLAN.
func (a *App) CreateNetwork(ctx context.Context, network Network) (*Network, error) {
	network.ID = ""
	results, err := unifi.DoLegacy[Network](a.Client, ctx, http.MethodPost, a.networkPath(), network)
	if err != nil {
		return nil, fmt.Errorf("creating network: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("create network returned no data")
	}
	return &results[0], nil
}

// UpdateNetwork updates an existing network. Full-object PUT required.
func (a *App) UpdateNetwork(ctx context.Context, network Network) (*Network, error) {
	if network.ID == "" {
		return nil, fmt.Errorf("network ID is required for update")
	}
	path := fmt.Sprintf("%s/%s", a.networkPath(), network.ID)
	results, err := unifi.DoLegacy[Network](a.Client, ctx, http.MethodPut, path, network)
	if err != nil {
		return nil, fmt.Errorf("updating network: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("update network returned no data")
	}
	return &results[0], nil
}

// DeleteNetwork deletes a network by ID. Risky — removes the VLAN.
// System networks with attr_no_delete: true cannot be deleted.
func (a *App) DeleteNetwork(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/%s", a.networkPath(), id)
	_, err := unifi.DoLegacy[Network](a.Client, ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting network: %w", err)
	}
	return nil
}
