package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

// Client is the HTTP client for the UniFi controller API.
// It supports API key authentication, configurable TLS, and
// per-application base path routing (Network, Protect, etc.).
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client

	// zoneSiteID is the site UUID used by the Integration API.
	// Auto-discovered on first use and cached.
	zoneSiteID   string
	zoneSiteOnce sync.Once
}

// ClientConfig holds configuration for creating a new Client.
type ClientConfig struct {
	BaseURL  string
	APIKey   string
	Insecure bool   // Skip TLS certificate verification
	CACert   string // Path to custom CA certificate file
}

// NewClient creates a new UniFi API client with API key authentication.
func NewClient(cfg ClientConfig) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
	}

	if cfg.CACert != "" {
		caCert, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = pool
	}

	return &Client{
		baseURL: u,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
	}, nil
}

// Application represents a UniFi application namespace.
type Application string

const (
	AppNetwork Application = "network"
	AppProtect Application = "protect"
	AppAccess  Application = "access"
)

// Do executes an HTTP request against the UniFi controller.
// The path should be relative to the application namespace
// (e.g., "/v2/api/site/default/static-dns" for Network app).
func (c *Client) Do(ctx context.Context, method string, app Application, path string, body any, result any) error {
	return c.do(ctx, method, app, path, body, result)
}

func (c *Client) do(ctx context.Context, method string, app Application, path string, body any, result any) error {
	fullPath := fmt.Sprintf("/proxy/%s%s", app, path)
	reqURL := c.baseURL.JoinPath(fullPath).String()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ErrConnection{Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := checkHTTPError(resp.StatusCode, respBody); err != nil {
		return err
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// checkHTTPError maps HTTP status codes to typed errors.
func checkHTTPError(statusCode int, body []byte) error {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return nil
	case statusCode == 401:
		return &ErrAuth{Message: "authentication failed", Body: string(body)}
	case statusCode == 404:
		return &ErrNotFound{Message: "resource not found", Body: string(body)}
	case statusCode == 429:
		return &ErrRateLimit{Message: "rate limit exceeded", Body: string(body)}
	default:
		return &ErrAPI{StatusCode: statusCode, Body: string(body)}
	}
}

// LegacyResponse wraps the legacy REST API response format.
type LegacyResponse[T any] struct {
	Data []T `json:"data"`
	Meta struct {
		RC  string `json:"rc"`
		Msg string `json:"msg,omitempty"`
	} `json:"meta"`
}

// doLegacy executes an HTTP request against the legacy REST API and unwraps the response.
func doLegacy[T any](c *Client, ctx context.Context, method string, path string, body any) ([]T, error) {
	var resp LegacyResponse[T]
	if err := c.do(ctx, method, AppNetwork, path, body, &resp); err != nil {
		return nil, err
	}
	if resp.Meta.RC != "ok" {
		return nil, &ErrAPI{StatusCode: 0, Body: fmt.Sprintf("legacy API error: %s", resp.Meta.Msg)}
	}
	return resp.Data, nil
}

// ZoneSiteID returns the site UUID used by the Integration API.
// It is auto-discovered from the sites endpoint and cached.
func (c *Client) ZoneSiteID(ctx context.Context) (string, error) {
	var retErr error
	c.zoneSiteOnce.Do(func() {
		var resp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		err := c.do(ctx, http.MethodGet, AppNetwork, "/integration/v1/sites", nil, &resp)
		if err != nil {
			retErr = fmt.Errorf("discovering zone site ID: %w", err)
			c.zoneSiteOnce = sync.Once{} // reset so it can be retried
			return
		}
		if len(resp.Data) == 0 {
			retErr = fmt.Errorf("no sites returned from Integration API")
			c.zoneSiteOnce = sync.Once{}
			return
		}
		c.zoneSiteID = resp.Data[0].ID
	})
	if retErr != nil {
		return "", retErr
	}
	return c.zoneSiteID, nil
}
