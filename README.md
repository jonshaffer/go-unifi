# go-unifi

Go SDK for UniFi Network controllers. Auto-generates Go types from the controller's OpenAPI spec (Integration API) and supplements with hand-written methods for legacy REST and v2 API resources.

Built for use with [terraform-provider-unifi](https://github.com/jonshaffer/terraform-provider-unifi).

## UniFi API Landscape

The UniFi controller exposes three generations of API, each with different capabilities and coverage. Understanding which API backs a resource matters for stability expectations and forward-compatibility.

### Integration API v1

**Base path**: `/proxy/network/integration/v1/sites/{siteId}/...`

The modern API, documented via an OpenAPI 3.1 spec served by the controller at `/proxy/network/api-docs/integration.json`. This is where Ubiquiti is actively investing — new features (zone-based firewall, ACL rules, device management) land here first. The SDK generates Go types from this spec via `oapi-codegen`, giving compile-time alignment with the controller's schema.

Characteristics: structured JSON responses, UUID-based IDs, paginated list endpoints, OpenAPI-documented request/response schemas.

### Legacy REST API

**Base path**: `/proxy/network/api/s/{site}/rest/...`

The original UniFi API, still functional for resources not yet migrated to the Integration API (networks, firewall groups, port forwarding, routing, WLAN configs). No OpenAPI spec exists — types are hand-written from observed API responses and the existing bash script implementations. Each resource uses 24-character hex IDs and full-object PUT for updates.

Characteristics: undocumented, stable but no longer receiving new features, `_id` hex identifiers, full-body PUT required for updates.

### v2 API

**Base path**: `/proxy/network/v2/api/site/{site}/...`

An intermediate API generation used for specific resources like DNS records. Partially documented, uses its own conventions (e.g., no GET-by-ID for DNS — must list and filter). Hand-written SDK methods cover these endpoints.

Characteristics: resource-specific, inconsistent conventions between endpoints, 24-character hex IDs.

## API Coverage

| Category | API | SDK Status | Notes |
|----------|-----|------------|-------|
| **Firewall Zones** | Integration v1 | Full CRUD | Generated types from OpenAPI |
| **Firewall Policies** | Integration v1 | Full CRUD | Generated types; PUT strips `id`/`metadata`/`index` |
| **Firewall Policy Ordering** | Integration v1 | Full CRUD | Separate ordering endpoint per zone pair |
| **Firewall Groups** | Legacy REST | Full CRUD | Hand-written; address-group and port-group types |
| **Networks** | Legacy REST | Full CRUD | Hand-written; exposes both `_id` and `external_id` |
| **DNS Records** | v2 | Full CRUD | Hand-written; no GET-by-ID (list + filter) |
| WiFi / SSIDs | Integration v1 | Not yet | 2 endpoints |
| ACL Rules | Integration v1 | Not yet | 3 endpoints (CRUD + ordering) |
| DNS Policies | Integration v1 | Not yet | 2 endpoints |
| VPN Servers | Integration v1 | Not yet | 1 endpoint |
| Site-to-Site Tunnels | Integration v1 | Not yet | 1 endpoint |
| Devices | Integration v1 | Not yet | 4 endpoints (CRUD + stats + port actions) |
| Clients | Integration v1 | Not yet | 2 endpoints (list + actions) |
| WAN Configuration | Integration v1 | Not yet | 1 endpoint |
| Hotspot Vouchers | Integration v1 | Not yet | 2 endpoints |
| Traffic Matching Lists | Integration v1 | Not yet | 2 endpoints |
| RADIUS Profiles | Integration v1 | Not yet | 1 endpoint |
| Device Tags | Integration v1 | Not yet | 1 endpoint |
| Port Forwarding | Legacy REST | Not yet | 1 endpoint |
| Port Profiles | Legacy REST | Not yet | 1 endpoint |
| Static Routes | Legacy REST | Not yet | 1 endpoint |
| Legacy Firewall Rules | Legacy REST | Not yet | 1 endpoint (pre-zone ruleset) |

## Architecture

The SDK is organized by UniFi application, allowing a single client to manage resources across Network, Protect, Access, etc.:

```
unifi/
├── client.go              # HTTP client, API key auth, TLS config
├── errors.go              # Typed errors (ErrAuth, ErrNotFound, ErrRateLimit, etc.)
├── pagination.go          # Transparent pagination helper
├── features.go            # Runtime feature detection (OpenAPI introspection + feature flags)
├── network/               # Network application
│   ├── firewall.generated.go  # Generated from OpenAPI spec (zones, policies, traffic filters)
│   ├── firewall_zone.go       # CRUD methods using generated types
│   ├── firewall_policy.go     # CRUD methods using generated types
│   ├── firewall_group.go      # Hand-written (legacy REST)
│   ├── network.go             # Hand-written (legacy REST)
│   └── dns_record.go          # Hand-written (v2 API)
└── protect/               # Protect application (stub — demonstrates multi-app pattern)
```

## Code Generation

Types for Integration API resources are generated from the controller's OpenAPI spec:

```bash
# Prerequisites
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
export PATH="$HOME/go/bin:$PATH"

# Generate types
go generate ./...
```

This runs the downconversion pipeline (`cmd/generate/`): OpenAPI 3.1 → 3.0 → `oapi-codegen` → `firewall.generated.go`.

The committed spec at `openapi/network-integration.json` is the version contract — its `info.version` field records which controller version the generated types correspond to.

## Drift Detection

The `cmd/validate/` tool compares generated Go struct JSON tags against live API responses:

```bash
go run ./cmd/validate/ --base-url https://192.168.1.1 --api-key $UNIFI_API_KEY
```

Output is a JSON report. Zero mismatches means the generated types match the running controller.

## Feature Detection

At runtime, the SDK can detect controller capabilities via two layers:

1. **OpenAPI introspection** — fetches the spec, parses endpoint paths, builds a capability map
2. **Feature flag queries** — queries controller feature flags (e.g., `ZoneBasedFirewall`), merges into the capability map

This enables the Terraform provider to gracefully handle different controller versions without hard version gates.
