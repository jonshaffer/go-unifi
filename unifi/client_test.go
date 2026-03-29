package unifi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient(ClientConfig{
		BaseURL:  "https://192.168.1.1",
		APIKey:   "test-key",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL.String() != "https://192.168.1.1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL.String(), "https://192.168.1.1")
	}
}

func TestNewClient_TrailingSlash(t *testing.T) {
	c, err := NewClient(ClientConfig{
		BaseURL:  "https://192.168.1.1/",
		APIKey:   "test-key",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.baseURL.String() != "https://192.168.1.1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL.String(), "https://192.168.1.1")
	}
}

func TestDo_APIKeyHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-KEY"); got != "test-key" {
			t.Errorf("X-API-KEY = %q, want %q", got, "test-key")
		}
		if got := r.URL.Path; got != "/proxy/network/test" {
			t.Errorf("path = %q, want %q", got, "/proxy/network/test")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "test-key"})
	var result map[string]string
	err := c.do(context.Background(), http.MethodGet, AppNetwork, "/test", nil, &result)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("result = %v", result)
	}
}

func TestDo_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "bad-key"})
	err := c.do(context.Background(), http.MethodGet, AppNetwork, "/test", nil, nil)
	if !IsAuth(err) {
		t.Errorf("expected ErrAuth, got %T: %v", err, err)
	}
}

func TestDo_NotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "key"})
	err := c.do(context.Background(), http.MethodGet, AppNetwork, "/missing", nil, nil)
	if !IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestDo_RateLimitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "key"})
	err := c.do(context.Background(), http.MethodGet, AppNetwork, "/test", nil, nil)
	if _, ok := err.(*ErrRateLimit); !ok {
		t.Errorf("expected ErrRateLimit, got %T: %v", err, err)
	}
}

func TestZoneSiteID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/network/integration/v1/sites" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"id": "site-uuid-123"}},
		})
	}))
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "key"})
	id, err := c.ZoneSiteID(context.Background())
	if err != nil {
		t.Fatalf("ZoneSiteID: %v", err)
	}
	if id != "site-uuid-123" {
		t.Errorf("ZoneSiteID = %q, want %q", id, "site-uuid-123")
	}

	// Second call should return cached value (no additional HTTP request).
	id2, err := c.ZoneSiteID(context.Background())
	if err != nil || id2 != id {
		t.Errorf("cached ZoneSiteID = %q, err = %v", id2, err)
	}
}

func TestDoLegacy(t *testing.T) {
	type testItem struct {
		Name string `json:"name"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"name": "item1"}, {"name": "item2"}},
			"meta": map[string]string{"rc": "ok"},
		})
	}))
	defer srv.Close()

	c, _ := NewClient(ClientConfig{BaseURL: srv.URL, APIKey: "key"})
	items, err := doLegacy[testItem](c, context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("doLegacy: %v", err)
	}
	if len(items) != 2 || items[0].Name != "item1" {
		t.Errorf("items = %v", items)
	}
}
