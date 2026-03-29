// Command validate compares generated Go struct fields against live API responses
// to detect drift between the committed types and the controller's current schema.
//
// Usage:
//
//	go run ./cmd/validate/ --base-url https://192.168.1.1 --api-key $UNIFI_API_KEY
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/jonshaffer/go-unifi/unifi"
	"github.com/jonshaffer/go-unifi/unifi/network"
)

type FieldMismatch struct {
	Resource  string `json:"resource"`
	Field     string `json:"field"`
	InStruct  bool   `json:"in_struct"`
	InAPI     bool   `json:"in_api"`
	StructTag string `json:"struct_tag,omitempty"`
}

type DriftReport struct {
	APIVersion string          `json:"api_version"`
	Mismatches []FieldMismatch `json:"mismatches"`
	Summary    string          `json:"summary"`
}

func main() {
	baseURL := flag.String("base-url", os.Getenv("UNIFI_BASE_URL"), "UniFi controller base URL")
	apiKey := flag.String("api-key", os.Getenv("UNIFI_API_KEY"), "UniFi API key")
	flag.Parse()

	if *baseURL == "" || *apiKey == "" {
		fmt.Fprintln(os.Stderr, "Usage: validate --base-url URL --api-key KEY")
		os.Exit(1)
	}

	client, err := unifi.NewClient(unifi.ClientConfig{
		BaseURL:  *baseURL,
		APIKey:   *apiKey,
		Insecure: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	report := DriftReport{}

	// Get API version
	caps, err := unifi.DetectCapabilities(client, ctx)
	if err == nil {
		report.APIVersion = caps.APIVersion
	}

	// Compare firewall zone struct vs live response
	zoneMismatches, err := compareResource(client, ctx, "FirewallZone",
		reflect.TypeOf(network.FirewallZone{}),
		"/integration/v1/sites/{siteId}/firewall/zones")
	if err != nil {
		fmt.Fprintf(os.Stderr, "zone comparison: %v\n", err)
	} else {
		report.Mismatches = append(report.Mismatches, zoneMismatches...)
	}

	if len(report.Mismatches) == 0 {
		report.Summary = "No drift detected"
	} else {
		report.Summary = fmt.Sprintf("%d field mismatches found", len(report.Mismatches))
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(report)

	if len(report.Mismatches) > 0 {
		os.Exit(1)
	}
}

func compareResource(c *unifi.Client, ctx context.Context, name string, structType reflect.Type, path string) ([]FieldMismatch, error) {
	// Get zone site ID for path substitution
	siteID, err := c.ZoneSiteID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting site ID: %w", err)
	}
	resolvedPath := strings.Replace(path, "{siteId}", siteID, 1)

	// Fetch raw JSON from the API
	var raw json.RawMessage
	reqURL := fmt.Sprintf("%s?limit=1", resolvedPath)
	err = c.Do(ctx, http.MethodGet, unifi.AppNetwork, reqURL, nil, &raw)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", name, err)
	}

	// Parse the paginated response to get first item's fields
	var page struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &page); err != nil || len(page.Data) == 0 {
		return nil, fmt.Errorf("parsing %s response: no data items", name)
	}

	var apiFields map[string]any
	if err := json.Unmarshal(page.Data[0], &apiFields); err != nil {
		return nil, fmt.Errorf("parsing %s item: %w", name, err)
	}

	// Get struct JSON tags
	structFields := jsonTagsFromType(structType)

	// Compare
	var mismatches []FieldMismatch
	allKeys := mergeKeys(structFields, apiFields)

	for _, key := range allKeys {
		_, inStruct := structFields[key]
		_, inAPI := apiFields[key]
		if inStruct != inAPI {
			m := FieldMismatch{
				Resource: name,
				Field:    key,
				InStruct: inStruct,
				InAPI:    inAPI,
			}
			if tag, ok := structFields[key]; ok {
				m.StructTag = tag
			}
			mismatches = append(mismatches, m)
		}
	}

	return mismatches, nil
}

// jsonTagsFromType extracts json tag names from a struct type.
func jsonTagsFromType(t reflect.Type) map[string]string {
	tags := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		tags[name] = tag
	}
	return tags
}

func mergeKeys(a map[string]string, b map[string]any) []string {
	seen := make(map[string]bool)
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
