# Performance Guidelines

## Caching

### Feature Status Cache (ccache)
- The Feature Status (subscriptions) cache uses `karlseguin/ccache/v3`, a concurrent LRU cache.
- Cache is keyed by `orgID` with a configurable TTL (`ENT_SUBS_CACHE_DURATION_SECONDS`, default 1800s).
- Max size (`ENT_SUBS_CACHE_MAX_SIZE`, default 500) and prune percentage (`ENT_SUBS_CACHE_ITEM_PRUNE`, default 10%) are set at init.
- **Fail-closed caching**: on upstream error or non-200 from Feature Service, an empty `FeatureStatus{}` is cached for the full TTL to prevent thundering herd against a failing dependency. The response is marked degraded via `X-Entitlements-Degraded` header.
- The `ForceFreshData` flag (triggered by `trial_activated=true`) bypasses the cache for that request but still populates it on response.

### AMS Org ID Cache (ccache)
- The AMS client caches `userOrgId -> amsOrgId` mappings for 30 minutes using a separate `ccache` instance with default sizing.
- Every `GetQuotaCost` and `GetSubscriptions` call benefits from this cache; `DeleteSubscription` operates on a subscription ID directly and does not go through `ConvertUserOrgId`.
- When adding new AMS operations that require org ID conversion, always go through `ConvertUserOrgId` to leverage this cache.

### Cache Configuration
- All cache tunables are environment variables prefixed with `ENT_` (viper `AutomaticEnv` with prefix `ENT`).
- Do not hardcode cache durations; use `config.Keys.SubsCacheDuration` and related keys.

## HTTP Clients

### Singleton mTLS Client
- `controllers.getClient()` returns a lazily-initialized singleton `*http.Client` with mTLS and a configurable timeout (`ENT_IT_SERVICES_TIMEOUT_SECONDS`, default 10s).
- This client is shared across all Feature Service and Compliance Service requests. Do not create new `http.Client` instances for these services.
- The BOP client creates its own `http.Client` in `bop.NewClient()` with TLS but no explicit timeout — be aware of this asymmetry.

### Timeout Handling
- The Compliance controller explicitly checks for `url.Error.Timeout()` to distinguish timeout errors from other failures.
- The Feature Service controller does not differentiate timeout errors. Both paths cache the fail-closed state.

### AMS SDK Connection
- The AMS client uses `ocm-sdk-go` with OAuth2 client credentials. The SDK manages its own token refresh and connection pooling internally.
- The SDK connection is created once in `ams.NewClient()` via `BuildContext(context.Background())`.

## Prometheus Metrics

### Naming Conventions
All metrics follow this pattern:
| Scope | Histogram (latency) | Counter (failures) |
|-------|---------------------|--------------------|
| HTTP layer | `entitlements_api_duration_seconds` (by path) | `entitlements_api_response_status` (by code, path) |
| Feature Service | `it_feature_service_time_taken` | `it_feature_service_failure` (by code) |
| Compliance Service | `it_export_compliance_service_time_taken` | `it_export_compliance_service_failure` (by code) |
| AMS operations | `quota_cost_service_request_time_taken`, `org_list_service_request_time_taken`, `get_subscription_service_request_time_taken`, `get_subscriptions_service_request_time_taken`, `delete_subscription_service_request_time_taken`, `quota_authorization_service_request_time_taken` | (none) |
| BOP | `bop_service_request_time_taken` | `back_office_proxy_service_failure` (by code) |

### Histogram Buckets
- All histograms use identical bucket config: `prometheus.LinearBuckets(0.25, 0.25, 20)` — 20 buckets from 0.25s to 5.0s in 0.25s increments.
- When adding new histograms, use the same bucket configuration for dashboard consistency.

### Registration Pattern
- Use `promauto.NewHistogram` / `promauto.NewCounterVec` for automatic registration (used in controllers, ams, bop, and for `requestDuration` in server/).
- Exception: `server/metrics.go` uses manual `prometheus.NewCounterVec` + explicit `prometheus.Register` in `init()` for `responseStatus`. New counter metrics in `server/` should follow that manual pattern; new histograms in `server/` may use `promauto` as `requestDuration` does.

### Instrumentation Pattern
```go
start := time.Now()
// ... do work ...
myHistogram.Observe(time.Since(start).Seconds())
```
- Always capture `start` before the operation and observe after.
- For the HTTP middleware, use `prometheus.NewTimer` instead.
- Failure counters use status code as the label: `myFailure.WithLabelValues(strconv.Itoa(statusCode)).Inc()`.

### Dashboard
- The Grafana dashboard is in `dashboards/grafana-dashboard-insights-entitlement-operations.yaml`.
- It queries metrics by `namespace` and `service="entitlements-api-go-service"`.
- When adding new metrics, update this dashboard to maintain observability.

## Concurrency

### No Goroutines or Channels
- This codebase does not use goroutines, channels, or `sync` primitives directly. Concurrency is handled entirely by the `net/http` server (one goroutine per request) and the `ccache` library (internally thread-safe).
- Do not introduce goroutines without careful consideration — the current design relies on request-scoped processing with no fan-out.

### Thread Safety
- `ccache` is safe for concurrent reads/writes. No additional locking is needed around cache access.
- The singleton `http.Client` is safe for concurrent use per Go stdlib guarantees.
- Package-level `var` for `bundleInfo`, `featuresQuery`, and `paidFeatureSuffix` are set once at startup before serving requests. Do not mutate them after `server.Launch()`.

## Configuration Defaults That Affect Performance

| Key | Default | Impact |
|-----|---------|--------|
| `ENT_SUBS_CACHE_DURATION_SECONDS` | 1800 | How long cached entitlements are valid |
| `ENT_SUBS_CACHE_MAX_SIZE` | 500 | Max entries before LRU eviction + pruning |
| `ENT_SUBS_CACHE_ITEM_PRUNE` | 10 | Percent of cache pruned when full |
| `ENT_IT_SERVICES_TIMEOUT_SECONDS` | 10 | HTTP client timeout for Feature/Compliance calls |
| `ENT_LOG_LEVEL` | info | Higher verbosity (debug) adds per-request log overhead |

## Dependency Resilience

- Feature Service failures are fail-closed: empty entitlements cached, response headers indicate degradation, Sentry captures the error.
- Compliance Service failures are not cached; each failed request hits the upstream again.
- BOP failures are not cached and propagate as 500s to the caller.
- AMS org ID lookup failures are not cached (only successful conversions are cached).
- When adding new external service integrations, follow the Feature Service pattern: cache failure state, set degradation headers, increment failure counters, and capture to Sentry.

## Build and Runtime

- Multi-stage Docker build: `go-toolset` builder, `ubi9-minimal` runtime.
- The binary runs as non-root (USER 1001).
- The server uses `http.ListenAndServe` with no graceful shutdown. Termination relies on the container orchestrator's SIGTERM handling.
- CloudWatch log batching is configured with a 10-second flush interval in the logger.
