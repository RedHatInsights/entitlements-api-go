# Security Guidelines — entitlements-api-go

## Identity and Authentication

- All business endpoints MUST use the `identity.EnforceIdentityWithLogger` middleware from `platform-go-middlewares/v2/identity`. Apply it via `r.With(enforceIdentity)` on each route group.
- The `/status` and `/metrics` endpoints are intentionally unauthenticated. Never add business logic to these routes.
- The `/api/entitlements/v1/openapi.json` endpoint is intentionally unauthenticated. Do not gate spec endpoints behind identity enforcement.
- Extract the caller's identity exclusively via `identity.GetIdentity(req.Context()).Identity`. Never parse `x-rh-identity` headers manually.

## Authorization

- Org admin checks: always check `idObj.User == nil` before accessing `idObj.User.OrgAdmin`. Service Accounts have a nil `User` field and must be treated as non-admin.
```go
if idObj.User == nil || !idObj.User.OrgAdmin {
    doError(w, http.StatusForbidden, ...)
    return
}
```
- Cross-org seat operations: always verify that the subscription's AMS org matches the caller's org via `ConvertUserOrgId` before allowing delete. Same pattern for PostSeats — verify `user.OrgId != idObj.Internal.OrgID` via BOP lookup.
- Compliance screening is not supported for Service Accounts (nil User field). Return 400, not 500.

## Input Validation

- Org IDs passed to AMS queries MUST be validated with `^[a-zA-Z0-9]+$` before use (see `validateOrgIdPattern`). This prevents injection into AMS search queries.
- Use the `QueryBuilder` for all AMS search queries. Never construct AMS query strings via raw string concatenation — the builder wraps values in single quotes.
- Pagination params (`limit`, `offset`) must be validated: limit > 0, offset >= 0. Reject with 400 on invalid values.
- Boolean query params use `strconv.ParseBool` and default to `false` on parse error — never panic on bad input.
- The compliance endpoint validates that `userIdentity.User.Username` is non-empty and non-whitespace before forwarding.

## Secrets and Configuration

- All secrets are injected via environment variables with the `ENT_` prefix (Viper `AutomaticEnv` with `SetEnvPrefix("ENT")`). Never hardcode secrets.
- Sensitive env vars: `ENT_OIDC_CLIENT_ID`, `ENT_OIDC_CLIENT_SECRET`, `ENT_BOP_TOKEN`, `ENT_BOP_CLIENT_ID`, `ENT_CW_KEY`, `ENT_CW_SECRET`.
- The `PaidFeatureSuffix` is set via `options.Set` (not `SetDefault`) to prevent override by environment. Use this pattern for values that must not be externally configurable.
- `GLITCHTIP_DSN` is read directly from `os.Getenv`, not through Viper — it is the only exception to the ENT_ prefix convention.

## TLS and HTTP Clients

- All outbound HTTP clients to IT services (Feature Service, Compliance Service) MUST use mutual TLS configured with the application's certificate-key pair and custom CA bundle from `config.GetConfig()`.
- The BOP client uses TLS with custom RootCAs but not client certificates (it authenticates via `x-rh-apitoken` and `x-rh-clientid` headers instead).
- The AMS client authenticates via OIDC client credentials (`ClientID`/`ClientSecret`) through the OCM SDK — do not configure TLS manually for AMS.
- HTTP client timeouts are set via `IT_SERVICES_TIMEOUT_SECONDS` (default 10s). Never create an `http.Client` without a timeout.
- In production, certificates are loaded from the `/certificates` volume mount. Test certificates in `test_data/` are only used when the volume is absent.

## Error Handling

- Never expose internal error details in HTTP responses to external callers. Use structured error types:
  - `types.DependencyErrorResponse` for upstream service failures (includes service name, status, endpoint).
  - `types.RequestErrorResponse` for bad client requests.
  - `api.Error` for seats API errors (maps AMS/BOP errors to appropriate status codes via `SeatsErrorMapper`).
- Log the full error server-side with `logger.Log` and report to Sentry via `sentry.CaptureException`. The response body should contain only the mapped/sanitized message.
- AMS error codes (e.g., `ACCT-MGMT-11`) are mapped to user-friendly messages via config. Add new mappings in `seats_error_mapper.go`.
- `doError` is defined in `controllers/seats.go` and delegates to `SeatsErrorMapper.MapResponse()` for all seat-manager endpoint errors.

## Fail-Closed Behavior

- When the Feature Service is unreachable or returns non-200, the system caches an empty `FeatureStatus{}` for the configured TTL. This means entitlements default to NOT entitled (fail-closed).
- Degraded responses include `X-Entitlements-Degraded: true` and `X-Entitlements-Degraded-Status` headers. Downstream consumers should check these.
- The `EntitleAll` config bypasses all entitlement checks — it must never be `true` in production.

## Container Security

- The Dockerfile runs the final image as `USER 1001` (non-root). Never add `USER root` to the runtime stage.
- Only the compiled binary, API spec, bundles config, and licenses are copied to the runtime image. Source code and build tools are excluded.
- Base images use pinned tags from the Hummingbird image registry (`registry.access.redhat.com/hi`). The builder uses `hi/go` (FIPS-enabled Go toolchain) and the runtime uses `hi/core-runtime` (minimal FIPS-enabled runtime).

## Debug Mode

- When `DEBUG=true`, AMS and BOP clients are replaced with mock implementations (`ams.Mock`, `bop.Mock`). This must never be enabled in production — it bypasses all external service calls and authorization checks.

## Logging

- Use JSON-formatted structured logging via `logrus` with field maps. Never use `fmt.Println` or `log.Println` for application logs. Known exceptions: `config/certificates.go` uses stdlib `log.Println` during certificate loading, and `bundle_sync/main.go` (a standalone CLI tool) uses `fmt.Println`.
- Never log raw identity headers, tokens, or secrets. The identity validation logger in routes.go logs the header content on failure — this is intentional for debugging auth issues but should not be extended.
- Prometheus metrics are exposed on `/metrics` without authentication. Metric names and labels must not contain PII or secrets (org IDs in labels are acceptable).

## Caching

- The subscription cache (`ccache`) is keyed by org ID with configurable TTL, max size, and prune percentage. Cache poisoning is mitigated by the identity middleware validating the caller before org ID extraction.
- AMS org ID conversion results are cached for 30 minutes. Both the input and output org IDs are validated against the alphanumeric pattern before caching.
