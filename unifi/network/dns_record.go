package network

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jonshaffer/go-unifi/unifi"
)

// DNSRecord represents a static DNS record (v2 API).
type DNSRecord struct {
	ID         string `json:"_id,omitempty"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	RecordType string `json:"record_type"`
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port,omitempty"`
	Priority   int    `json:"priority,omitempty"`
	TTL        int    `json:"ttl,omitempty"`
	Weight     int    `json:"weight,omitempty"`
}

func (a *App) dnsPath() string {
	return fmt.Sprintf("/v2/api/site/%s/static-dns", a.Site)
}

// ListDNSRecords returns all static DNS records.
func (a *App) ListDNSRecords(ctx context.Context) ([]DNSRecord, error) {
	var records []DNSRecord
	err := a.Client.Do(ctx, http.MethodGet, unifi.AppNetwork, a.dnsPath(), nil, &records)
	if err != nil {
		return nil, fmt.Errorf("listing DNS records: %w", err)
	}
	return records, nil
}

// GetDNSRecord returns a single DNS record by ID.
// The v2 DNS API has no GET-by-ID endpoint (405), so this lists all and filters.
func (a *App) GetDNSRecord(ctx context.Context, id string) (*DNSRecord, error) {
	records, err := a.ListDNSRecords(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, &unifi.ErrNotFound{Message: fmt.Sprintf("DNS record %s not found", id)}
}

// CreateDNSRecord creates a new static DNS record.
func (a *App) CreateDNSRecord(ctx context.Context, record DNSRecord) (*DNSRecord, error) {
	record.ID = "" // controller assigns ID
	var result DNSRecord
	err := a.Client.Do(ctx, http.MethodPost, unifi.AppNetwork, a.dnsPath(), record, &result)
	if err != nil {
		return nil, fmt.Errorf("creating DNS record: %w", err)
	}
	return &result, nil
}

// UpdateDNSRecord updates an existing DNS record. Full-object PUT with _id in body.
func (a *App) UpdateDNSRecord(ctx context.Context, record DNSRecord) (*DNSRecord, error) {
	if record.ID == "" {
		return nil, fmt.Errorf("DNS record ID is required for update")
	}
	path := fmt.Sprintf("%s/%s", a.dnsPath(), record.ID)
	var result DNSRecord
	err := a.Client.Do(ctx, http.MethodPut, unifi.AppNetwork, path, record, &result)
	if err != nil {
		return nil, fmt.Errorf("updating DNS record: %w", err)
	}
	return &result, nil
}

// DeleteDNSRecord deletes a DNS record by ID.
func (a *App) DeleteDNSRecord(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/%s", a.dnsPath(), id)
	err := a.Client.Do(ctx, http.MethodDelete, unifi.AppNetwork, path, nil, nil)
	if err != nil {
		return fmt.Errorf("deleting DNS record: %w", err)
	}
	return nil
}
