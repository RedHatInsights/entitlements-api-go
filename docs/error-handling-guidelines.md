# Error Handling Guidelines

## Logging Library

- Use `logger.Log` (the global `*logrus.Logger` singleton from `logger` package) for all application logging.
- Import alias convention varies by file: `l "github.com/RedHatInsights/entitlements-api-go/logger"` in `compliance.go` and `subscriptions.go`; no alias (`"github.com/RedHatInsights/entitlements-api-go/logger"`) in `seats.go`; `log` alias in `server/routes.go`. Use `l` as the alias in new controller files; use no alias or `log` for non-controller files.
- Always use structured fields via `logrus.Fields`:
  ```go
  l.Log.WithFields(logrus.Fields{"error": err, "org_id": orgId}).Error("descriptive message")
  ```
- The `"error"` field key is the standard key for attaching an error to a log entry. Do not use `"err"` or other variants.
- JSON formatter is configured globally with remapped field keys (`ts`, `caller`, `logLevel`, `msg`). Do not configure additional formatters.

## Sentry / Glitchtip Integration

- Sentry SDK is initialized via the `GLITCHTIP_DSN` environment variable in `main.go`. It may not be active in all environments.
- Use `sentry.CaptureException(err)` for all unexpected errors that indicate a bug or infrastructure failure.
- Use `sentry.WithScope` when you need to attach contextual tags (response body, status code, URL) to a specific exception:
  ```go
  sentry.WithScope(func(scope *sentry.Scope) {
      scope.SetTag("response_body", body)
      scope.SetTag("response_status", strconv.Itoa(statusCode))
      scope.SetTag("url", url)
      sentry.CaptureException(wrappedErr)
  })
  ```
- Prefer not calling `sentry.CaptureException` for client errors (400-level). Known exception: `failOnBadRequest` in the compliance controller calls `sentry.CaptureException` for 400 responses (Service Account and missing-username cases) because those indicate a misconfigured or unexpected caller identity.
- The Sentry HTTP middleware (`sentryhttp`) with `Repanic: true` is installed on the router. It captures panics and re-panics so `middleware.Recoverer` (registered earlier in the middleware stack, and therefore outer) can also handle them.

## Custom Error Types

### `ams.ClientError`
- Used for AMS client-layer errors that carry an HTTP status code, org ID context, and optional AMS org ID.
- Implements `error` interface. Includes contextual identifiers in the message string.

### `bop.UserDetailError`
- Used for BOP user-lookup failures. Carries `Message`, `StatusCode`, and `UserName`.
- Implements `error` interface with a formatted message including all fields.

### `ocmErrors.Error` (external)
- Errors from the OCM SDK (`github.com/openshift-online/ocm-sdk-go/errors`).
- Imported with alias `ocmErrors` in the error mapper.

## Error Mapping (Seats API)

- All seat-manager endpoint errors go through `doError(w, httpStatusCode, err, source)` in `controllers/seats.go`.
- `doError` delegates to `SeatsErrorMapper.MapResponse()` which uses `errors.As` to match custom error types in priority order:
  1. `*ocmErrors.Error` — extracts AMS error code, reason, operation ID; may append config-driven message for known codes (e.g., `ACCT-MGMT-11`).
  2. `*ams.ClientError` — uses message and status from the client error.
  3. `*bop.UserDetailError` — uses message and status from BOP error.
  4. Fallback — wraps `err.Error()` with the provided HTTP status code.
- `doError` logs at `Error` level for 500s and `Debug` level for non-500s.
- The `source` parameter is a free-text label identifying the upstream call (e.g., `"AMS GetSubscription"`, `"BOP GetUser"`). Pass `""` for locally-generated errors.

## Error Response Formats

### Seats API (`api.Error` — generated from OpenAPI spec)
- Fields: `Error *string`, `Code *string`, `Identifier *string`, `OperationId *string`, `Status *int`.
- All fields are pointers. Use the `toPtr[T]` generic helper to set them.

### Subscriptions/Services endpoint (`types.DependencyErrorResponse`)
- Used when an external dependency (Feature Service) fails.
- Structure: `{ "error": { "dependency_failure": true, "service": "...", "status": N, "endpoint": "...", "message": "..." } }`.
- Returned with HTTP 500 via `failOnDependencyError`.

### Compliance endpoint (`types.DependencyErrorResponse` and `types.RequestErrorResponse`)
- `failOnBadRequest` — returns 400 with `RequestErrorResponse` for invalid input (e.g., service accounts).
- `failOnComplianceError` — returns 500 with `DependencyErrorResponse` for compliance service failures.
- `failOnServiceError` — returns 500 with plain text for internal marshaling errors.

## Error Wrapping

- Use `fmt.Errorf("context: %w", err)` to wrap errors that will be unwrapped with `errors.As` downstream.
- The `%w` verb is used in BOP client (`"Error from trying to send BOP GetUser request [%w]"`), compliance controller, and seats controller.
- Use `[%w]` bracket style for wrapping in user-facing error messages (e.g., `fmt.Errorf("PostSeats [%w]", err)`).
- Use `: %w` style for internal/config errors (e.g., `fmt.Errorf("unable to load certificates: %w", err)`).
- Do NOT use `errors.Wrap` from `pkg/errors` — this repo uses only stdlib `fmt.Errorf` with `%w`.

## Panics

- Panics are acceptable ONLY during startup initialization for unrecoverable configuration failures:
  - `config.initialize()` — panics if certificates cannot be loaded.
  - `logger.InitLogger()` — panics if log level is unparseable.
  - `server.DoRoutes()` — panics if AMS or BOP clients fail to construct.
- Never panic in request handlers. The `middleware.Recoverer` middleware is a safety net, not a design pattern.

## Fatal Logging

- `logger.Log.Fatal()` is used in `main.go` for startup failures (e.g., bundle info loading) and in `server.Launch()` when the HTTP server stops.
- `log.Fatalf()` (stdlib) is used in `bundle_sync/main.go` (a standalone CLI tool, not the API server).
- Do not use `log.Fatalf` in the API server; use `logger.Log.Fatal` instead.

## Degraded Mode (Fail-Closed Pattern)

- When the Feature Service returns an error or non-200 status, the `/services` endpoint does NOT return an error to the caller.
- Instead, it caches an empty feature set (fail-closed), sets `X-Entitlements-Degraded: true` and `X-Entitlements-Degraded-Status` headers, and returns 200 with restricted entitlements.
- The error is still logged and sent to Sentry.
- The `isCachedFailClosed` function detects cache hits with empty feature data, which also triggers degraded mode.

## Prometheus Metrics for Errors

- Failure counters track error rates by HTTP status code string label:
  - `it_feature_service_failure` (label: `code`) — Feature Service errors.
  - `it_export_compliance_service_failure` (label: `code`) — Compliance service errors.
  - `back_office_proxy_service_failure` (label: `code`) — BOP errors.
- Always increment the appropriate counter when returning or logging an error from an external dependency.
- Use `strconv.Itoa(statusCode)` for the label value.

## Conventions Summary

1. Return `error` from functions; handle at the call site. Do not log-and-return (pick one) except in HTTP handlers where you must do both.
2. HTTP handlers: log the error, optionally send to Sentry, then write the appropriate error response format.
3. Service/client layers: return errors (potentially wrapped with context) without logging. Let the handler decide.
4. Silently discarding errors (`body, _ := io.ReadAll(...)`) is acceptable only for best-effort operations like reading an error response body for logging.
5. Always `defer resp.Body.Close()` after checking for nil response — the current codebase consistently uses `defer` for body close in all response-handling paths.
