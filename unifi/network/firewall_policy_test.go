package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jonshaffer/go-unifi/unifi"
)

const (
	policySitesPath = "/proxy/network/integration/v1/sites"
	policiesPath    = "/proxy/network/integration/v1/sites/test-site-uuid/firewall/policies"
)

// policyRouter returns an http.HandlerFunc that handles site discovery and
// delegates policy requests to the provided handler.
//
// Note: Client.Do builds the request URL via url.URL.JoinPath, which
// percent-encodes any "?" in the path segment. This means query parameters
// appended to paths (e.g., pagination offsets, ordering zone IDs) end up
// encoded inside r.URL.Path rather than r.URL.RawQuery. The test handlers
// account for this by matching on the decoded path string.
func policyRouter(t *testing.T, policyHandler http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == policySitesPath:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "test-site-uuid"}},
			})
		case strings.HasPrefix(r.URL.Path, policiesPath):
			policyHandler(w, r)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}
}

func samplePolicyResponse() FirewallPolicyResponse {
	p := FirewallPolicyResponse{
		ID:             "policy-1",
		Name:           "Allow LAN to WAN",
		Enabled:        true,
		Index:          1,
		LoggingEnabled: false,
	}
	p.Action.Type = "ALLOW"
	p.Action.AllowReturnTraffic = true
	p.Source.ZoneID = "zone-lan"
	p.Destination.ZoneID = "zone-wan"
	p.IPProtocolScope.IPVersion = "IPV4"
	p.Metadata.Origin = "USER_DEFINED"
	return p
}

func samplePolicyRequest() FirewallPolicyRequest {
	req := FirewallPolicyRequest{
		Name:           "Allow LAN to WAN",
		Enabled:        true,
		LoggingEnabled: false,
	}
	req.Action.Type = "ALLOW"
	req.Action.AllowReturnTraffic = true
	req.Source.ZoneID = "zone-lan"
	req.Destination.ZoneID = "zone-wan"
	req.IPProtocolScope.IPVersion = "IPV4"
	return req
}

func TestListFirewallPolicies(t *testing.T) {
	p1 := samplePolicyResponse()
	p2 := samplePolicyResponse()
	p2.ID = "policy-2"
	p2.Name = "Block IoT to LAN"
	p2.Action.Type = "BLOCK"
	p2.Index = 2

	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		// ListAll appends ?offset=0&limit=200 as query parameters.
		if r.URL.Query().Get("offset") != "0" || r.URL.Query().Get("limit") != "200" {
			t.Errorf("expected pagination query params, got: %s", r.URL.RawQuery)
		}

		json.NewEncoder(w).Encode(unifi.PaginatedResponse[FirewallPolicyResponse]{
			Data:       []FirewallPolicyResponse{p1, p2},
			Offset:     0,
			Limit:      200,
			Count:      2,
			TotalCount: 2,
		})
	}))
	defer srv.Close()

	app := testApp(srv)
	policies, err := app.ListFirewallPolicies(context.Background())
	if err != nil {
		t.Fatalf("ListFirewallPolicies: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("got %d policies, want 2", len(policies))
	}
	if policies[0].Name != "Allow LAN to WAN" {
		t.Errorf("policies[0].Name = %q, want %q", policies[0].Name, "Allow LAN to WAN")
	}
	if policies[1].Action.Type != "BLOCK" {
		t.Errorf("policies[1].Action.Type = %q, want BLOCK", policies[1].Action.Type)
	}
}

func TestGetFirewallPolicy(t *testing.T) {
	policy := samplePolicyResponse()

	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		expectedPath := policiesPath + "/policy-1"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}
		json.NewEncoder(w).Encode(policy)
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.GetFirewallPolicy(context.Background(), "policy-1")
	if err != nil {
		t.Fatalf("GetFirewallPolicy: %v", err)
	}
	if result.ID != "policy-1" {
		t.Errorf("ID = %q, want policy-1", result.ID)
	}
	if result.Name != "Allow LAN to WAN" {
		t.Errorf("Name = %q, want %q", result.Name, "Allow LAN to WAN")
	}
	if result.Source.ZoneID != "zone-lan" {
		t.Errorf("Source.ZoneID = %q, want zone-lan", result.Source.ZoneID)
	}
	if result.Metadata.Origin != "USER_DEFINED" {
		t.Errorf("Metadata.Origin = %q, want USER_DEFINED", result.Metadata.Origin)
	}
}

func TestCreateFirewallPolicy(t *testing.T) {
	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if r.URL.Path != policiesPath {
			t.Errorf("path = %s, want %s", r.URL.Path, policiesPath)
		}

		// Decode the request body and verify it matches FirewallPolicyRequest
		// (no id, metadata, or index fields).
		var raw map[string]any
		json.NewDecoder(r.Body).Decode(&raw)

		if _, hasID := raw["id"]; hasID {
			t.Error("request body should not contain 'id'")
		}
		if _, hasMeta := raw["metadata"]; hasMeta {
			t.Error("request body should not contain 'metadata'")
		}
		if _, hasIndex := raw["index"]; hasIndex {
			t.Error("request body should not contain 'index'")
		}
		if raw["name"] != "Allow LAN to WAN" {
			t.Errorf("body.name = %v, want %q", raw["name"], "Allow LAN to WAN")
		}

		// Return a response with server-assigned fields.
		resp := samplePolicyResponse()
		resp.ID = "new-policy-id"
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	app := testApp(srv)
	req := samplePolicyRequest()
	result, err := app.CreateFirewallPolicy(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateFirewallPolicy: %v", err)
	}
	if result.ID != "new-policy-id" {
		t.Errorf("ID = %q, want new-policy-id", result.ID)
	}
	if result.Action.Type != "ALLOW" {
		t.Errorf("Action.Type = %q, want ALLOW", result.Action.Type)
	}
}

func TestUpdateFirewallPolicy(t *testing.T) {
	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}

		expectedPath := policiesPath + "/policy-1"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}

		var raw map[string]any
		json.NewDecoder(r.Body).Decode(&raw)

		if _, hasID := raw["id"]; hasID {
			t.Error("request body should not contain 'id'")
		}
		if _, hasMeta := raw["metadata"]; hasMeta {
			t.Error("request body should not contain 'metadata'")
		}
		if _, hasIndex := raw["index"]; hasIndex {
			t.Error("request body should not contain 'index'")
		}
		if raw["name"] != "Updated Policy Name" {
			t.Errorf("body.name = %v, want %q", raw["name"], "Updated Policy Name")
		}

		resp := samplePolicyResponse()
		resp.Name = "Updated Policy Name"
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	app := testApp(srv)
	req := samplePolicyRequest()
	req.Name = "Updated Policy Name"
	result, err := app.UpdateFirewallPolicy(context.Background(), "policy-1", req)
	if err != nil {
		t.Fatalf("UpdateFirewallPolicy: %v", err)
	}
	if result.Name != "Updated Policy Name" {
		t.Errorf("Name = %q, want %q", result.Name, "Updated Policy Name")
	}
	if result.ID != "policy-1" {
		t.Errorf("ID = %q, want policy-1", result.ID)
	}
}

func TestDeleteFirewallPolicy(t *testing.T) {
	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}

		expectedPath := policiesPath + "/policy-1"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	app := testApp(srv)
	err := app.DeleteFirewallPolicy(context.Background(), "policy-1")
	if err != nil {
		t.Fatalf("DeleteFirewallPolicy: %v", err)
	}
}

func TestGetPolicyOrdering(t *testing.T) {
	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/ordering") {
			t.Errorf("expected /ordering in path, got: %s", r.URL.Path)
		}
		if r.URL.Query().Get("sourceFirewallZoneId") != "zone-lan" {
			t.Errorf("expected sourceFirewallZoneId=zone-lan, got: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("destinationFirewallZoneId") != "zone-wan" {
			t.Errorf("expected destinationFirewallZoneId=zone-wan, got: %s", r.URL.RawQuery)
		}

		var resp PolicyOrdering
		resp.OrderedPolicyIDs.BeforeSystemDefined = []string{"policy-2", "policy-1", "policy-3"}
		resp.OrderedPolicyIDs.AfterSystemDefined = []string{}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	app := testApp(srv)
	ordering, err := app.GetPolicyOrdering(context.Background(), "zone-lan", "zone-wan")
	if err != nil {
		t.Fatalf("GetPolicyOrdering: %v", err)
	}
	before := ordering.OrderedPolicyIDs.BeforeSystemDefined
	if len(before) != 3 {
		t.Fatalf("got %d policy IDs, want 3", len(before))
	}
	if before[0] != "policy-2" {
		t.Errorf("BeforeSystemDefined[0] = %q, want policy-2", before[0])
	}
	if before[1] != "policy-1" {
		t.Errorf("BeforeSystemDefined[1] = %q, want policy-1", before[1])
	}
}

func TestSetPolicyOrdering(t *testing.T) {
	srv := httptest.NewServer(policyRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/ordering") {
			t.Errorf("expected /ordering in path, got: %s", r.URL.Path)
		}
		if r.URL.Query().Get("sourceFirewallZoneId") != "zone-lan" {
			t.Errorf("expected sourceFirewallZoneId=zone-lan, got: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("destinationFirewallZoneId") != "zone-wan" {
			t.Errorf("expected destinationFirewallZoneId=zone-wan, got: %s", r.URL.RawQuery)
		}

		var body PolicyOrdering
		json.NewDecoder(r.Body).Decode(&body)
		before := body.OrderedPolicyIDs.BeforeSystemDefined
		if len(before) != 3 {
			t.Fatalf("body has %d policy IDs, want 3", len(before))
		}
		if before[0] != "policy-3" {
			t.Errorf("body.BeforeSystemDefined[0] = %q, want policy-3", before[0])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	app := testApp(srv)
	var ordering PolicyOrdering
	ordering.OrderedPolicyIDs.BeforeSystemDefined = []string{"policy-3", "policy-1", "policy-2"}
	ordering.OrderedPolicyIDs.AfterSystemDefined = []string{}
	err := app.SetPolicyOrdering(context.Background(), "zone-lan", "zone-wan", ordering)
	if err != nil {
		t.Fatalf("SetPolicyOrdering: %v", err)
	}
}
