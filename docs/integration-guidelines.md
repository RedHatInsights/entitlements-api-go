# Integration Guidelines

## External Services Overview

This repo integrates with four external services:
- **AMS** (Account Management Service) via `ocm-sdk-go` — `ams/` package
- **BOP** (Back Office Proxy) via raw `net/http` — `bop/` package
- **Feature Service** (IT Services) via raw `net/http` — `controllers/subscriptions.go`
- **Export Compliance Service** via raw `net/http` — `controllers/compliance.go`

## Client Architecture

### Interface + Mock Pattern
Every external client MUST define an interface and a mock implementation in the same package:
- `ams/client.go` defines `AMSInterface`, `ams/mock_client.go` defines `Mock`
- `bop/client.go` defines `Bop` interface, same file defines `Mock`
- Use `var _ InterfaceName = &Client{}` to enforce compile-time interface compliance
- AMS Mock uses exported `var` functions (e.g., `var MockGetQuotaCost = func(...)`) so tests can override behavior per-test; BOP Mock uses a configurable `OrgId` struct field instead

### Constructor Pattern
All clients use a `NewClient(debug bool)` constructor that returns `(Interface, error)`:
- When `debug == true`, return the mock implementation
- When `debug == false`, build a real client from config
- Validate required config before constructing the client (see `bop.validateBOPSettings`)
- Panic at startup in `server/routes.go` if client construction fails

### Dependency Injection
Controllers receive client interfaces via constructor injection:
```go
type SeatManagerApi struct {
    ams ams.AMSInterface
    bop bop.Bop
}
```
Clients are constructed in `server/routes.go` and passed into controllers.

## Configuration

### Config Access Pattern
- All external service config comes from `config.GetConfig().Options` (a `*viper.Viper` instance)
- Config keys are defined as constants in `config.EntitlementsConfigKeysType`
- Environment variables use `ENT_` prefix (set via `options.SetEnvPrefix("ENT")`)
- Prefer setting a default value in `config/main.go:initialize()` for every config key. Exception: secret-type keys (`OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `BOP_CLIENT_ID`, `BOP_TOKEN`) and `FEATURES` intentionally have no default and must be supplied at runtime.

### Key Config Values Per Service
| Service | Host Key | Other Required Keys |
|---------|----------|-------------------|
| AMS | `AMS_HOST` | `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OAUTH_TOKEN_URL` |
| BOP | `BOP_URL` | `BOP_CLIENT_ID`, `BOP_TOKEN`, `BOP_ENV` |
| Feature Service | `SUBS_HOST` | `FEATURE_STATUS_API_PATH`, `FEATURES` |
| Compliance | `COMPLIANCE_HOST` | `COMP_API_BASE_PATH` |

## HTTP Client Conventions

### Shared HTTP Client for IT Services
Feature Service and Compliance Service share a singleton `*http.Client` from `controllers/client.go`:
- Uses mutual TLS with certs from `config.GetConfig().Certs` and `config.GetConfig().RootCAs`
- Timeout is configurable via `IT_SERVICES_TIMEOUT_SECONDS` (default: 10s)
- The client is lazily initialized and reused across requests

### BOP HTTP Client
BOP builds its own `http.Client` with TLS (RootCAs only, no client cert) in `bop.NewClient`.

### AMS Client
AMS uses `ocm-sdk-go` connection builder, not raw HTTP. Auth is OAuth2 client credentials.

## Metrics

### Histogram Naming Convention
Every external call MUST have a latency histogram. Use this pattern:
```go
var myServiceTime = promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "<service>_service_request_time_taken",
    Help:    "<service> service latency distributions.",
    Buckets: prometheus.LinearBuckets(0.25, 0.25, 20),
})
```
All histograms use identical bucket config: `LinearBuckets(0.25, 0.25, 20)`.

### Timing Pattern
Capture timing with `time.Now()` before the call and `Observe(time.Since(start).Seconds())` after:
```go
start := time.Now()
resp, err := client.Do(req)
myServiceTime.Observe(time.Since(start).Seconds())
```

### Failure Counters
Prefer adding a failure counter with a status code label for services that can fail:
```go
var myFailure = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "<service>_service_failure",
    Help: "Total number of <service> failures",
}, []string{"code"})
```
Increment with: `myFailure.WithLabelValues(strconv.Itoa(statusCode)).Inc()`

Known exception: AMS operations use only latency histograms and do not have failure counters.

## Error Handling

### Custom Error Types
Each client package defines its own error type with a `StatusCode` field:
- `ams.ClientError{Message, StatusCode, OrgId, AmsOrgId}`
- `bop.UserDetailError{Message, StatusCode, UserName}`

### Error Mapper for AMS
`controllers/seats_error_mapper.go` maps upstream errors to API responses using `errors.As`:
- `ocmErrors.Error` (from ocm-sdk-go) -> mapped with AMS error codes
- `ams.ClientError` -> passed through
- `bop.UserDetailError` -> passed through
- Known AMS error codes (e.g., `ACCT-MGMT-11`) get augmented messages via config

### Dependency Error Response Format
When an external service fails, return a `DependencyErrorResponse`:
```go
types.DependencyErrorResponse{
    Error: types.DependencyErrorDetails{
        DependencyFailure: true,
        Service:           "Service Name",
        Status:            statusCode,
        Endpoint:          url,
        Message:           errMsg,
    },
}
```

### Sentry Integration
- Call `sentry.CaptureException(err)` for all unexpected external service errors
- Use `sentry.WithScope` to attach contextual tags (response body, status, URL)
- Do NOT capture expected client errors (e.g., 400 Bad Request)

## Caching

### OrgID Cache (AMS)
`ams/client.go` uses `ccache` to cache user-org-id to AMS-org-id mappings:
- TTL: 30 minutes (hardcoded)
- Check `item.Expired()` before using cached value

### Feature Status Cache (Subscriptions)
`controllers/subscriptions.go` uses `ccache` with configurable parameters:
- `SUBS_CACHE_DURATION_SECONDS` (default: 1800)
- `SUBS_CACHE_MAX_SIZE` (default: 500)
- `SUBS_CACHE_ITEM_PRUNE` (default: 10%)
- **Fail-closed caching**: on downstream failure, cache an empty `FeatureStatus{}` to prevent repeated failing calls
- When cached fail-closed data is served, set `X-Entitlements-Degraded: true` header

## Resilience Patterns

### No Retry Logic
This codebase does NOT implement retries for any external service call. Failures are handled by:
1. Caching fail-closed state (Feature Service)
2. Returning errors immediately to the caller (AMS, BOP, Compliance)

### Timeout Handling
- IT Services (Feature/Compliance): configurable via `IT_SERVICES_TIMEOUT_SECONDS` on the shared HTTP client
- AMS: managed by ocm-sdk-go internally
- BOP: no explicit timeout configured on the HTTP client
- Compliance controller explicitly checks for `url.Error.Timeout()` to differentiate timeout errors

### Degraded Mode
When Feature Service is unavailable, the `/services` endpoint:
1. Caches empty feature data (fail-closed)
2. Sets `X-Entitlements-Degraded: true` response header
3. Sets `X-Entitlements-Degraded-Status` header with the upstream status code
4. Still returns a 200 with all SKU-based bundles set to `is_entitled: false`

## Testing Conventions

- Test framework: Ginkgo v2 + Gomega
- Each package has a `*_suite_test.go` with `RegisterFailHandler` and `RunSpecs`
- Mock external calls by replacing exported `var` functions (e.g., `ams.MockGetQuotaCost = func(...)`)
- The `controllers/subscriptions.go` `GetFeatureStatus` is a `var` function to allow test overriding

## AMS Query Builder

Use `ams.NewQueryBuilder()` for constructing AMS search queries:
```go
query := NewQueryBuilder().
    Like("plan.id", "AnsibleWisdom").
    And().
    Equals("organization_id", orgId).
    Build()
```
Supports: `Like`, `Equals`, `In`, `And`. Always chain with `And()` between conditions.

## Input Validation

- Validate org IDs with `validateOrgIdPattern` (alphanumeric only) before sending to AMS
- Validate user identity fields (nil checks on `User`, whitespace checks on `Username`) before calling compliance
- Service Accounts (`User == nil`) are explicitly blocked from compliance screening and seat management
