package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonshaffer/go-unifi/unifi"
)

func testApp(srv *httptest.Server) *App {
	c, _ := unifi.NewClient(unifi.ClientConfig{BaseURL: srv.URL, APIKey: "test"})
	return NewApp(c, "default")
}

func TestListDNSRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/network/v2/api/site/default/static-dns" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]DNSRecord{
			{ID: "abc123", Key: "nas.hyperfluid.dev", Value: "192.168.10.39", RecordType: "A", Enabled: true},
			{ID: "def456", Key: "talos.hyperfluid.dev", Value: "192.168.10.55", RecordType: "A", Enabled: true},
		})
	}))
	defer srv.Close()

	app := testApp(srv)
	records, err := app.ListDNSRecords(context.Background())
	if err != nil {
		t.Fatalf("ListDNSRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Key != "nas.hyperfluid.dev" {
		t.Errorf("records[0].Key = %q", records[0].Key)
	}
}

func TestGetDNSRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]DNSRecord{
			{ID: "abc123", Key: "nas.hyperfluid.dev", Value: "192.168.10.39", RecordType: "A", Enabled: true},
			{ID: "def456", Key: "talos.hyperfluid.dev", Value: "192.168.10.55", RecordType: "A", Enabled: true},
		})
	}))
	defer srv.Close()

	app := testApp(srv)
	record, err := app.GetDNSRecord(context.Background(), "def456")
	if err != nil {
		t.Fatalf("GetDNSRecord: %v", err)
	}
	if record.Key != "talos.hyperfluid.dev" {
		t.Errorf("Key = %q, want talos.hyperfluid.dev", record.Key)
	}
}

func TestGetDNSRecord_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]DNSRecord{})
	}))
	defer srv.Close()

	app := testApp(srv)
	_, err := app.GetDNSRecord(context.Background(), "nonexistent")
	if !unifi.IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestCreateDNSRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var body DNSRecord
		json.NewDecoder(r.Body).Decode(&body)
		if body.ID != "" {
			t.Error("expected ID to be empty on create")
		}
		body.ID = "new123"
		json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()

	app := testApp(srv)
	result, err := app.CreateDNSRecord(context.Background(), DNSRecord{
		Key: "test.hyperfluid.dev", Value: "192.168.10.99", RecordType: "A", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateDNSRecord: %v", err)
	}
	if result.ID != "new123" {
		t.Errorf("ID = %q, want new123", result.ID)
	}
}

func TestDeleteDNSRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/proxy/network/v2/api/site/default/static-dns/abc123" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	app := testApp(srv)
	err := app.DeleteDNSRecord(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("DeleteDNSRecord: %v", err)
	}
}
