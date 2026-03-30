// Acceptance tests for the UniFi Network SDK.
// These tests require a live UniFi controller and are gated on UNIFI_ACC=1.
//
// Required environment variables:
//   - UNIFI_ACC=1 (enables these tests)
//   - UNIFI_API_KEY (API key for the controller)
//   - UNIFI_BASE_URL (e.g., https://192.168.1.1)
//   - UNIFI_SITE (optional, defaults to "default")
package network

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jonshaffer/go-unifi/unifi"
)

func skipUnlessAcc(t *testing.T) {
	t.Helper()
	if os.Getenv("UNIFI_ACC") != "1" {
		t.Skip("Set UNIFI_ACC=1 to run acceptance tests")
	}
}

func accApp(t *testing.T) *App {
	t.Helper()

	apiKey := os.Getenv("UNIFI_API_KEY")
	baseURL := os.Getenv("UNIFI_BASE_URL")
	site := os.Getenv("UNIFI_SITE")
	if site == "" {
		site = "default"
	}

	if apiKey == "" || baseURL == "" {
		t.Fatal("UNIFI_API_KEY and UNIFI_BASE_URL must be set")
	}

	c, err := unifi.NewClient(unifi.ClientConfig{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}

	return NewApp(c, site)
}

func TestAccAuthentication(t *testing.T) {
	skipUnlessAcc(t)
	app := accApp(t)
	ctx := context.Background()

	// Verify we can discover the site UUID (proves auth and connectivity)
	siteID, err := app.Client.ZoneSiteID(ctx)
	if err != nil {
		t.Fatalf("ZoneSiteID: %v", err)
	}
	if siteID == "" {
		t.Fatal("ZoneSiteID returned empty string")
	}
	t.Logf("Site UUID: %s", siteID)
}

func TestAccListFirewallZones(t *testing.T) {
	skipUnlessAcc(t)
	app := accApp(t)
	ctx := context.Background()

	zones, err := app.ListFirewallZones(ctx)
	if err != nil {
		t.Fatalf("ListFirewallZones: %v", err)
	}

	// Every controller has at least the system-defined zones
	if len(zones) == 0 {
		t.Fatal("expected at least one firewall zone")
	}

	t.Logf("Found %d zones", len(zones))
	for _, z := range zones {
		t.Logf("  Zone: %s (origin=%s, networks=%d)", z.Name, z.Metadata.Origin, len(z.NetworkIDs))
	}
}

func TestAccDNSRecordCRUD(t *testing.T) {
	skipUnlessAcc(t)
	app := accApp(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("acc-test-%d.hyperfluid.dev", os.Getpid())

	// Create
	created, err := app.CreateDNSRecord(ctx, DNSRecord{
		Key:        testKey,
		Value:      "192.168.99.99",
		RecordType: "A",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created record has no ID")
	}
	t.Logf("Created DNS record: %s -> %s (ID: %s)", created.Key, created.Value, created.ID)

	// Cleanup on exit
	defer func() {
		if err := app.DeleteDNSRecord(ctx, created.ID); err != nil {
			t.Errorf("cleanup DeleteDNSRecord: %v", err)
		} else {
			t.Logf("Cleaned up DNS record %s", created.ID)
		}
	}()

	// Read back
	record, err := app.GetDNSRecord(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetDNSRecord: %v", err)
	}
	if record.Key != testKey {
		t.Errorf("Key = %q, want %q", record.Key, testKey)
	}
	if record.Value != "192.168.99.99" {
		t.Errorf("Value = %q, want 192.168.99.99", record.Value)
	}

	// Update
	updated, err := app.UpdateDNSRecord(ctx, DNSRecord{
		ID:         created.ID,
		Key:        testKey,
		Value:      "192.168.99.100",
		RecordType: "A",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("UpdateDNSRecord: %v", err)
	}
	if updated.Value != "192.168.99.100" {
		t.Errorf("updated Value = %q, want 192.168.99.100", updated.Value)
	}
}

func TestAccPagination(t *testing.T) {
	skipUnlessAcc(t)
	app := accApp(t)
	ctx := context.Background()

	// List firewall policies — exercises the pagination helper
	policies, err := app.ListFirewallPolicies(ctx)
	if err != nil {
		t.Fatalf("ListFirewallPolicies: %v", err)
	}

	t.Logf("Found %d policies", len(policies))
	for i, p := range policies {
		t.Logf("  [%d] %s (action=%s, enabled=%v)", i, p.Name, p.Action.Type, p.Enabled)
	}
}
