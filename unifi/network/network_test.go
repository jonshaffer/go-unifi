package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonshaffer/go-unifi/unifi"
)

func legacyWrap(data any) map[string]any {
	return map[string]any{"data": data, "meta": map[string]string{"rc": "ok"}}
}

func TestListNetworks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/network/api/s/default/rest/networkconf" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(legacyWrap([]Network{
			{ID: "net1", Name: "Servers", VLAN: 10, Purpose: "corporate"},
			{ID: "net2", Name: "IoT", VLAN: 20, Purpose: "corporate"},
		}))
	}))
	defer srv.Close()

	app := testApp(srv)
	nets, err := app.ListNetworks(context.Background())
	if err != nil {
		t.Fatalf("ListNetworks: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("got %d networks, want 2", len(nets))
	}
	if nets[0].Name != "Servers" {
		t.Errorf("nets[0].Name = %q", nets[0].Name)
	}
}

func TestGetNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/network/api/s/default/rest/networkconf/net1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(legacyWrap([]Network{
			{ID: "net1", ExternalID: "uuid-1", Name: "Servers", VLAN: 10},
		}))
	}))
	defer srv.Close()

	app := testApp(srv)
	net, err := app.GetNetwork(context.Background(), "net1")
	if err != nil {
		t.Fatalf("GetNetwork: %v", err)
	}
	if net.ExternalID != "uuid-1" {
		t.Errorf("ExternalID = %q, want uuid-1", net.ExternalID)
	}
}

func TestGetNetwork_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(legacyWrap([]Network{}))
	}))
	defer srv.Close()

	app := testApp(srv)
	_, err := app.GetNetwork(context.Background(), "missing")
	if !unifi.IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestUpdateNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		var body Network
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(legacyWrap([]Network{body}))
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.UpdateNetwork(context.Background(), Network{
		ID: "net1", Name: "Servers", MdnsEnabled: true,
	})
	if err != nil {
		t.Fatalf("UpdateNetwork: %v", err)
	}
	if !result.MdnsEnabled {
		t.Error("expected MdnsEnabled = true")
	}
}
