package network

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jonshaffer/go-unifi/unifi"
)

const (
	sitesPath = "/proxy/network/integration/v1/sites"
	zonesPath = "/proxy/network/integration/v1/sites/test-site-uuid/firewall/zones"
)

// zoneRouter returns an http.HandlerFunc that handles site discovery and
// delegates zone requests to the provided handler.
func zoneRouter(t *testing.T, zoneHandler http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == sitesPath:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "test-site-uuid"}},
			})
		case strings.HasPrefix(r.URL.Path, zonesPath):
			zoneHandler(w, r)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}
}

func TestListFirewallZones(t *testing.T) {
	zones := []FirewallZoneResponse{
		{
			ID:         "zone-1",
			Name:       "Internal",
			NetworkIDs: []string{"net-1", "net-2"},
		},
		{
			ID:         "zone-2",
			Name:       "External",
			NetworkIDs: []string{},
		},
	}
	zones[0].Metadata.Origin = "SYSTEM_DEFINED"
	zones[0].Metadata.Configurable = false
	zones[1].Metadata.Origin = "USER_DEFINED"
	zones[1].Metadata.Configurable = true

	srv := httptest.NewServer(zoneRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		json.NewEncoder(w).Encode(unifi.PaginatedResponse[FirewallZoneResponse]{
			Data:       zones,
			Offset:     0,
			Limit:      200,
			Count:      2,
			TotalCount: 2,
		})
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.ListFirewallZones(context.Background())
	if err != nil {
		t.Fatalf("ListFirewallZones: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d zones, want 2", len(result))
	}
	if result[0].Name != "Internal" {
		t.Errorf("result[0].Name = %q, want Internal", result[0].Name)
	}
	if result[1].Name != "External" {
		t.Errorf("result[1].Name = %q, want External", result[1].Name)
	}
	if result[0].Metadata.Origin != "SYSTEM_DEFINED" {
		t.Errorf("result[0].Metadata.Origin = %q, want SYSTEM_DEFINED", result[0].Metadata.Origin)
	}
	if len(result[0].NetworkIDs) != 2 {
		t.Errorf("result[0].NetworkIDs length = %d, want 2", len(result[0].NetworkIDs))
	}
}

func TestGetFirewallZone(t *testing.T) {
	zone := FirewallZoneResponse{
		ID:         "zone-1",
		Name:       "Internal",
		NetworkIDs: []string{"net-1"},
	}
	zone.Metadata.Origin = "SYSTEM_DEFINED"
	zone.Metadata.Configurable = false

	srv := httptest.NewServer(zoneRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		expectedPath := zonesPath + "/zone-1"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}
		json.NewEncoder(w).Encode(zone)
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.GetFirewallZone(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("GetFirewallZone: %v", err)
	}
	if result.ID != "zone-1" {
		t.Errorf("ID = %q, want zone-1", result.ID)
	}
	if result.Name != "Internal" {
		t.Errorf("Name = %q, want Internal", result.Name)
	}
	if result.Metadata.Origin != "SYSTEM_DEFINED" {
		t.Errorf("Metadata.Origin = %q, want SYSTEM_DEFINED", result.Metadata.Origin)
	}
}

func TestCreateFirewallZone(t *testing.T) {
	srv := httptest.NewServer(zoneRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != zonesPath {
			t.Errorf("path = %s, want %s", r.URL.Path, zonesPath)
		}

		body, _ := io.ReadAll(r.Body)
		var req FirewallZoneRequest
		json.Unmarshal(body, &req)

		if req.Name != "DMZ" {
			t.Errorf("request Name = %q, want DMZ", req.Name)
		}
		if len(req.NetworkIDs) != 1 || req.NetworkIDs[0] != "net-dmz" {
			t.Errorf("request NetworkIDs = %v, want [net-dmz]", req.NetworkIDs)
		}

		// Verify no id or metadata in the request body.
		var raw map[string]any
		json.Unmarshal(body, &raw)
		if _, ok := raw["id"]; ok {
			t.Error("request body should not contain id")
		}
		if _, ok := raw["metadata"]; ok {
			t.Error("request body should not contain metadata")
		}

		resp := FirewallZoneResponse{
			ID:         "zone-new",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		}
		resp.Metadata.Origin = "USER_DEFINED"
		resp.Metadata.Configurable = true
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.CreateFirewallZone(context.Background(), FirewallZoneRequest{
		Name:       "DMZ",
		NetworkIDs: []string{"net-dmz"},
	})
	if err != nil {
		t.Fatalf("CreateFirewallZone: %v", err)
	}
	if result.ID != "zone-new" {
		t.Errorf("ID = %q, want zone-new", result.ID)
	}
	if result.Name != "DMZ" {
		t.Errorf("Name = %q, want DMZ", result.Name)
	}
	if result.Metadata.Origin != "USER_DEFINED" {
		t.Errorf("Metadata.Origin = %q, want USER_DEFINED", result.Metadata.Origin)
	}
}

func TestUpdateFirewallZone(t *testing.T) {
	srv := httptest.NewServer(zoneRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		expectedPath := zonesPath + "/zone-1"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}

		body, _ := io.ReadAll(r.Body)
		var req FirewallZoneRequest
		json.Unmarshal(body, &req)

		if req.Name != "Internal-Updated" {
			t.Errorf("request Name = %q, want Internal-Updated", req.Name)
		}
		if len(req.NetworkIDs) != 2 {
			t.Errorf("request NetworkIDs length = %d, want 2", len(req.NetworkIDs))
		}

		// Verify no id or metadata in the request body.
		var raw map[string]any
		json.Unmarshal(body, &raw)
		if _, ok := raw["id"]; ok {
			t.Error("request body should not contain id")
		}
		if _, ok := raw["metadata"]; ok {
			t.Error("request body should not contain metadata")
		}

		resp := FirewallZoneResponse{
			ID:         "zone-1",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		}
		resp.Metadata.Origin = "USER_DEFINED"
		resp.Metadata.Configurable = true
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.UpdateFirewallZone(context.Background(), "zone-1", FirewallZoneRequest{
		Name:       "Internal-Updated",
		NetworkIDs: []string{"net-1", "net-3"},
	})
	if err != nil {
		t.Fatalf("UpdateFirewallZone: %v", err)
	}
	if result.ID != "zone-1" {
		t.Errorf("ID = %q, want zone-1", result.ID)
	}
	if result.Name != "Internal-Updated" {
		t.Errorf("Name = %q, want Internal-Updated", result.Name)
	}
}

func TestDeleteFirewallZone(t *testing.T) {
	srv := httptest.NewServer(zoneRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		expectedPath := zonesPath + "/zone-1"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	app := testApp(srv)
	err := app.DeleteFirewallZone(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("DeleteFirewallZone: %v", err)
	}
}

func TestGetFirewallZone_NotFound(t *testing.T) {
	srv := httptest.NewServer(zoneRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/nonexistent") {
			t.Errorf("path = %s, expected suffix /nonexistent", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "zone not found"}`))
	}))
	defer srv.Close()

	app := testApp(srv)
	_, err := app.GetFirewallZone(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !unifi.IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}
