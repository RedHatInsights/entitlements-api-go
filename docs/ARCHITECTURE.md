# Architecture

This document captures institutional knowledge about the entitlements-api-go system — the design decisions, data flows, and operational characteristics that are not obvious from reading individual source files and are not covered in [AGENTS.md](../AGENTS.md) or the guideline documents in `docs/`.

For project structure, package responsibilities, naming conventions, middleware stack, configuration system, and build workflow, see [AGENTS.md](../AGENTS.md).

## System Overview

The entitlements-api-go service is a proxy that sits between console.redhat.com (and other Hybrid Cloud Console frontends) and several Red Hat IT backend services. Its job is to answer a simple question for any authenticated user: "What are you entitled to?"

It occupies a critical position in the platform: every page load on console.redhat.com that needs to know whether a user can access a product (Ansible, OpenShift, Insights, etc.) calls this service. This means:

- **Availability matters more than correctness.** The service is designed to return a 200 even when its dependencies are down, defaulting to "not entitled" (fail-closed) rather than returning errors that would break the console.
- **Latency matters.** The service caches aggressively (30-minute TTL by default) because it is called on nearly every authenticated request to the platform.
- **The service has no database.** All state comes from external services or is derived from static YAML configuration. The only mutable state is the in-memory cache.

### What the Service Is NOT

This service does not manage subscriptions, create entitlements, or modify any backend state (with the obsolete exception of the seats API). It is strictly a read proxy with caching. The actual entitlement logic (which SKUs map to which features) lives in the IT Feature Service; this service merely queries it and interprets the results through the lens of bundle configuration.

## Key Design Decisions

### Why Fail-Closed Caching Instead of Fail-Open

When the Feature Service is unreachable, the service caches an empty `FeatureStatus{}` for the full TTL (default 30 minutes). This means all SKU-based entitlements default to `is_entitled: false`. The alternative — fail-open (granting access when we cannot verify) — was rejected because it would allow unauthorized access to paid products during outages. The trade-off is that legitimate users may temporarily lose access during Feature Service outages, but the response headers (`X-Entitlements-Degraded: true`) allow downstream consumers to detect and communicate this state.

The empty result is cached (rather than retrying on each request) to prevent a thundering herd against a failing dependency. If 10,000 users hit the service while Feature Service is down, only the first request per org (before the cache entry is set) actually contacts the upstream.

### Why No Retry Logic

The codebase intentionally does not retry failed external service calls. The reasoning is:

1. **Feature Service:** Fail-closed caching handles the failure case. Retries would increase latency for the user and add load to an already struggling upstream.
2. **Compliance Service:** Requests are pass-through proxies. The caller (the console frontend) can retry if needed.
3. **AMS/BOP (seats API):** These are write-path operations where automatic retries risk double-execution (e.g., assigning a seat twice).

### Why Two API Styles Coexist

The service predates oapi-codegen adoption at Red Hat. The original `/services` and `/compliance` endpoints were written as plain `http.HandlerFunc` handlers. When the seat management feature was added later, the team chose to use oapi-codegen for type safety. Migrating the existing endpoints to code generation was deemed not worth the effort since they are stable and rarely change. New endpoints should use the generated style (see [api-contracts-guidelines.md](api-contracts-guidelines.md)).

### Why the BOP Client Has No Timeout

The BOP `http.Client` is constructed in `bop.NewClient()` without an explicit `Timeout` field, unlike the shared IT services client which has a configurable timeout (default 10s). This is an asymmetry, not a deliberate design choice. The BOP client is only used by the seats API (which is disabled in production), so the risk exposure is minimal. If the BOP client is ever re-enabled or used for new features, a timeout should be added.

### Why PaidFeatureSuffix Uses Set Instead of SetDefault

The `PaidFeatureSuffix` config key is set with `options.Set()` rather than `options.SetDefault()`. This is intentional: `Set` prevents the value from being overridden by an environment variable. The `_paid` suffix is a contract between this service and the IT Feature Service. If someone accidentally set `ENT_PAID_FEATURE_SUFFIX` in production, it would silently break paid/trial detection for all bundles. Using `Set` makes this impossible.

### Why the Bundle Config Is Gitignored

The actual `bundles/bundles.yml` file is gitignored. In production, it comes from a ConfigMap sourced from a separate repository ([entitlements-config](https://github.com/RedHatInsights/entitlements-config)). This separation exists because:

1. **Deployment independence.** SKU-to-bundle mappings change frequently as products are added or SKUs are updated. These changes should not require a code deployment.
2. **Security.** The SKU list is considered business-sensitive information.
3. **Operational control.** The entitlements-config repo has its own review process and deployment pipeline.

The committed `bundles.example.yml` exists only for local development and testing.

## Data Flow

### GET /api/entitlements/v1/services

This is the primary endpoint. It answers "what is this user entitled to?" for every bundle.

```
Client (with x-rh-identity header)
  |
  v
Identity Middleware -- decodes and validates the x-rh-identity header
  |
  v
Services Handler (controllers/subscriptions.go)
  |
  |-- Extract org ID from identity
  |-- Check in-memory cache (ccache, keyed by org ID)
  |     |
  |     |-- Cache HIT (not expired, not force-fresh): use cached FeatureStatus
  |     |-- Cache MISS or force-fresh (trial_activated=true):
  |           |
  |           v
  |         Build Feature Service URL with feature query params
  |         (features derived from bundles.yml at startup)
  |           |
  |           v
  |         GET https://<SUBS_HOST>/features/v2/featureStatus?features=X&features=Y&accountId=<orgId>
  |         (mTLS with enterprise cert)
  |           |
  |           |-- Success (200): parse response, cache result for TTL
  |           |-- Error or non-200: cache empty FeatureStatus{} (fail-closed),
  |           |   set degraded=true, log + Sentry
  |
  v
For each bundle in bundles.yml:
  |-- If EntitleAll is true: entitled=true (dev/test only)
  |-- If bundle has SKUs: check if feature name exists in Feature Service response
  |     |-- If bundle has paid_skus: also check for "<name>_paid" feature
  |     |   to determine trial vs paid status
  |-- If bundle has use_valid_acc_num: require non-empty, non-"-1" account number
  |-- If bundle has use_valid_org_id: require non-empty, non-"-1" org ID
  |-- If bundle has use_is_internal: require valid account + internal user + @redhat.com email
  |
  v
Return JSON map: { "bundle_name": { "is_entitled": bool, "is_trial": bool }, ... }
If degraded: add X-Entitlements-Degraded headers
```

The response is a dynamic map, not a fixed schema. Adding a new bundle to `bundles.yml` automatically adds it to the response without code changes.

**Non-SKU-based bundles** (like `insights` with `use_valid_acc_num: true`) never call the Feature Service. They are resolved purely from identity header attributes.

### GET /api/entitlements/v1/compliance

This is a pass-through proxy to the Red Hat Export Compliance screening service.

```
Client (with x-rh-identity header)
  |
  v
Identity Middleware
  |
  v
Compliance Handler (controllers/compliance.go)
  |-- Validate: must be a User identity (not Service Account)
  |-- Validate: username must be non-empty and non-whitespace
  |-- Construct ComplianceScreeningRequest with username
  |
  v
POST https://<COMPLIANCE_HOST>/v1/screening
(mTLS with enterprise cert, shared HTTP client with timeout)
  |
  v
Pass through the response status code and body unchanged
```

Unlike `/services`, the compliance endpoint does not cache results and does not implement degraded mode. Each request hits the upstream. Failures return 500 with a `DependencyErrorResponse`.

### Seats API (Obsolete, Disabled by Default)

The seats endpoints (`GET /seats`, `POST /seats`, `DELETE /seats/{id}`) manage Ansible Wisdom subscription seat assignments through AMS (Account Management Service). They are disabled by default (`DisableSeatManager: true`) and are not enabled in production.

The data flow involves two external services working together:

- **AMS** (via ocm-sdk-go): Manages subscriptions, quota, and org ID translation
- **BOP** (Back Office Proxy): Looks up user details to verify org membership before seat assignment

Key authorization flow for DELETE:
```
1. Verify caller is org admin (from identity header)
2. Fetch the subscription from AMS
3. Convert caller's org ID to AMS org ID (with caching)
4. Verify subscription's org matches caller's AMS org
5. Only then delete the subscription
```

## External Dependencies

### IT Feature Service (Subscription Service)

- **Purpose:** Returns which features/bundles an organization is entitled to based on their SKU subscriptions.
- **Protocol:** HTTPS with mutual TLS (enterprise certificate).
- **Coupling:** Medium. The service constructs a feature query at startup from `bundles.yml` and the `FEATURES` config. The query is static for the lifetime of the process.
- **Failure mode:** Fail-closed caching with degraded response headers.
- **API path:** `GET /features/v2/featureStatus?features=X&features=Y&accountId=<orgId>`

### Export Compliance Service

- **Purpose:** Screens users for export compliance (trade sanctions, embargoes).
- **Protocol:** HTTPS with mutual TLS (same enterprise certificate).
- **Coupling:** Low. Pure proxy — the service forwards the request and returns the response unchanged.
- **Failure mode:** Returns 500 to caller. No caching of failures.
- **API path:** `POST /v1/screening`

### AMS (Account Management Service)

- **Purpose:** Manages OpenShift subscriptions and seat assignments. Used only by the (disabled) seats API.
- **Protocol:** HTTPS with OAuth2 client credentials, via the `ocm-sdk-go` library.
- **Coupling:** High. AMS-specific query syntax, org ID format (requiring translation), and error codes. The SDK manages its own connection pooling and token refresh.
- **Failure mode:** Errors propagate directly to caller via error mapper.

### BOP (Back Office Proxy)

- **Purpose:** Looks up user details to verify org membership for seat operations.
- **Protocol:** HTTPS with API token authentication (`x-rh-apitoken`, `x-rh-clientid` headers). No client certificate.
- **Coupling:** Low. Single `POST /v1/users` endpoint.
- **Failure mode:** Errors propagate directly to caller. No timeout configured.

### CloudWatch (Logging)

- **Purpose:** Centralized log aggregation.
- **Coupling:** Low. Configured as a logrus hook with 10-second batch flush. If credentials are absent, the hook is simply not added and logs go to stdout only.
- **Note:** In Clowder-managed environments, CloudWatch config is overridden from Clowder's provided configuration.

### Glitchtip/Sentry (Error Tracking)

- **Purpose:** Captures unexpected errors and panics for alerting and debugging.
- **Coupling:** Very low. Initialized from `GLITCHTIP_DSN` env var (the only env var that does not use the `ENT_` prefix). If absent, error tracking is silently disabled.

## Deployment Model

### Clowder and OpenShift

The service runs as a `ClowdApp` on OpenShift, managed by the Clowder operator (`deployment/clowdapp.yml`). Key characteristics:

- **Single deployment** named `service` with a public web service on port 8000.
- **Health checks:** Liveness and readiness probes both hit `/status`. Liveness has a 20s initial delay; readiness has 30s.

### Init Container: bundle-sync

Before the main API server starts, an init container runs the `bundle-sync` binary:

1. Reads bundle SKU configuration from `bundles.yml` (mounted from the `entitlements-config` ConfigMap).
2. For each feature in the `FEATURES` config, compares the SKU list against what is registered in the IT Feature Service.
3. If there are differences, POSTs the updated SKU list to the Feature Service's `/features/v1` endpoint.
4. For paid bundles, creates two features: `<name>` (eval + paid SKUs) and `<name>_paid` (paid SKUs only).

This ensures the Feature Service has correct SKU-to-feature mappings before the API starts. The sync can be disabled with `RUN_BUNDLE_SYNC=false`.

### Certificate Management

Two certificate delivery mechanisms:

1. **Volume mount (default for production):** Delivered via the `it-key-pair` OpenShift secret, mounted at `/certificates/`. AppSRE handles automatic renewal. Requires `ENT_CERTS_FROM_ENV=false`.
2. **Environment variables:** When `ENT_CERTS_FROM_ENV=true`, certificates are read from `ENT_CA_CERT`, `ENT_CERT`, and `ENT_KEY`. Primarily for local development.

If the `/certificates` directory does not exist and `CERTS_FROM_ENV` is false, the service falls back to test certificates in `test_data/`. Development only.

Certificate loading happens during config initialization. If certificates cannot be loaded, the process panics — there is no point starting a service that cannot reach its dependencies.

### CI/CD Pipeline

- **Konflux/Tekton** (`.tekton/`): Primary CI and deployment pipeline. Triggered on PR and push to main. Builds the container image, runs tests, generates SBOMs, and deploys to stage automatically on merge.
- **Production deployment:** Requires updating the image tag in the app-interface deployment configuration to the desired commit SHA. Manual step.
- **GitHub Actions** (`.github/workflows/`): Supplementary checks only — security scanning, JSON/YAML validation, PR labeling. Does not run Go tests.

## Known Constraints and Trade-offs

### No Graceful Shutdown

The server uses `http.ListenAndServe` without graceful shutdown. When the process receives SIGTERM during rolling updates, in-flight requests may be terminated abruptly. The `defer sentry.Flush(2 * time.Second)` in `main.go` only executes if `ListenAndServe` returns an error, so error reports may be lost during normal SIGTERM-based shutdown.

### The Features Query Is Built Once and Cached Forever

The `featuresQuery` variable (the URL query string sent to the Feature Service) is built on the first request and never updated. If `bundles.yml` changes at runtime (e.g., ConfigMap update), the process must be restarted. This is acceptable because ConfigMap changes in OpenShift trigger pod restarts.

### AMS Org ID vs Platform Org ID

The platform uses one org ID format (from the `x-rh-identity` header) while AMS uses a different internal org ID. The `ConvertUserOrgId` method translates between them, with results cached for 30 minutes (hardcoded, not configurable unlike the Feature Service cache).

### The Seats API Is Obsolete But Not Removed

The seats API endpoints are disabled by default (`DisableSeatManager: true`). The code remains because:

1. It is the only user of oapi-codegen in this service, and removing it would also remove the code generation infrastructure.
2. The AMS and BOP clients are only constructed if the seat manager is enabled, so disabled seats add zero runtime overhead.

### No Rate Limiting

The service has no application-level rate limiting. It relies on the platform's API gateway (3scale) and OpenShift network policies. The in-memory cache provides some natural protection against downstream overload, but a burst of requests with unique org IDs could still overwhelm the Feature Service.

### Prometheus Metric Naming Inconsistency

HTTP-layer metrics use the `entitlements_api_` prefix while service-specific metrics use varied prefixes (`it_feature_service_`, `bop_service_`, `quota_cost_service_`, etc.). AMS operations have latency histograms but no failure counters, unlike other services. This inconsistency is historical and does not affect functionality.
