package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonshaffer/go-unifi/unifi"
)

func TestListFirewallGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(legacyWrap([]FirewallGroup{
			{ID: "g1", Name: "RFC1918", GroupType: "address-group", GroupMembers: []string{"10.0.0.0/8", "172.16.0.0/12"}},
			{ID: "g2", Name: "NFS Ports", GroupType: "port-group", GroupMembers: []string{"111", "2049"}},
		}))
	}))
	defer srv.Close()

	app := testApp(srv)
	groups, err := app.ListFirewallGroups(context.Background())
	if err != nil {
		t.Fatalf("ListFirewallGroups: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if groups[0].GroupType != "address-group" {
		t.Errorf("groups[0].GroupType = %q", groups[0].GroupType)
	}
}

func TestGetFirewallGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(legacyWrap([]FirewallGroup{
			{ID: "g1", ExternalID: "ext-uuid-1", Name: "RFC1918", GroupType: "address-group"},
		}))
	}))
	defer srv.Close()

	app := testApp(srv)
	group, err := app.GetFirewallGroup(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GetFirewallGroup: %v", err)
	}
	if group.ExternalID != "ext-uuid-1" {
		t.Errorf("ExternalID = %q, want ext-uuid-1", group.ExternalID)
	}
}

func TestGetFirewallGroup_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(legacyWrap([]FirewallGroup{}))
	}))
	defer srv.Close()

	app := testApp(srv)
	_, err := app.GetFirewallGroup(context.Background(), "missing")
	if !unifi.IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestCreateFirewallGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var body FirewallGroup
		json.NewDecoder(r.Body).Decode(&body)
		body.ID = "new-g1"
		body.ExternalID = "new-ext-uuid"
		json.NewEncoder(w).Encode(legacyWrap([]FirewallGroup{body}))
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.CreateFirewallGroup(context.Background(), FirewallGroup{
		Name: "Test Group", GroupType: "port-group", GroupMembers: []string{"80", "443"},
	})
	if err != nil {
		t.Fatalf("CreateFirewallGroup: %v", err)
	}
	if result.ID != "new-g1" {
		t.Errorf("ID = %q", result.ID)
	}
}
