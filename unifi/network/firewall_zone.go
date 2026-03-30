package network

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jonshaffer/go-unifi/unifi"
)

// FirewallZoneResponse is the API response for a firewall zone.
type FirewallZoneResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	NetworkIDs []string `json:"networkIds"`
	Metadata   struct {
		Origin       string `json:"origin"`       // SYSTEM_DEFINED or USER_DEFINED
		Configurable bool   `json:"configurable"`
	} `json:"metadata"`
}

// FirewallZoneRequest is the API request for creating/updating a zone.
// Must NOT include id or metadata (400 error).
type FirewallZoneRequest struct {
	Name       string   `json:"name"`
	NetworkIDs []string `json:"networkIds"`
}

func (a *App) zonePath(ctx context.Context) (string, error) {
	siteID, err := a.Client.ZoneSiteID(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/integration/v1/sites/%s/firewall/zones", siteID), nil
}

// ListFirewallZones returns all firewall zones (paginated).
func (a *App) ListFirewallZones(ctx context.Context) ([]FirewallZoneResponse, error) {
	basePath, err := a.zonePath(ctx)
	if err != nil {
		return nil, err
	}
	return unifi.ListAll[FirewallZoneResponse](a.Client, ctx, unifi.AppNetwork, basePath)
}

// GetFirewallZone returns a single zone by ID.
func (a *App) GetFirewallZone(ctx context.Context, id string) (*FirewallZoneResponse, error) {
	basePath, err := a.zonePath(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s", basePath, id)
	var zone FirewallZoneResponse
	if err := a.Client.Do(ctx, http.MethodGet, unifi.AppNetwork, path, nil, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

// CreateFirewallZone creates a new user-defined zone.
func (a *App) CreateFirewallZone(ctx context.Context, req FirewallZoneRequest) (*FirewallZoneResponse, error) {
	basePath, err := a.zonePath(ctx)
	if err != nil {
		return nil, err
	}
	var zone FirewallZoneResponse
	if err := a.Client.Do(ctx, http.MethodPost, unifi.AppNetwork, basePath, req, &zone); err != nil {
		return nil, fmt.Errorf("creating firewall zone: %w", err)
	}
	return &zone, nil
}

// UpdateFirewallZone updates an existing zone. PUT with full body, no id/metadata.
func (a *App) UpdateFirewallZone(ctx context.Context, id string, req FirewallZoneRequest) (*FirewallZoneResponse, error) {
	basePath, err := a.zonePath(ctx)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s", basePath, id)
	var zone FirewallZoneResponse
	if err := a.Client.Do(ctx, http.MethodPut, unifi.AppNetwork, path, req, &zone); err != nil {
		return nil, fmt.Errorf("updating firewall zone: %w", err)
	}
	return &zone, nil
}

// DeleteFirewallZone deletes a user-defined zone. System zones cannot be deleted.
func (a *App) DeleteFirewallZone(ctx context.Context, id string) error {
	basePath, err := a.zonePath(ctx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/%s", basePath, id)
	return a.Client.Do(ctx, http.MethodDelete, unifi.AppNetwork, path, nil, nil)
}
