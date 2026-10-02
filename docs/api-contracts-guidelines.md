# API Contracts Guidelines

## Source of Truth

- The OpenAPI 3.0 spec lives at `apispec/api.spec.json`. All API contract changes start here.
- The spec is served at runtime via `/api/entitlements/v1/openapi.json` (read from disk, path configured via `OpenAPISpecPath` config key).
- Base URL for all endpoints: `/api/entitlements/v1/`.

## Code Generation (oapi-codegen)

- Only the `seats` tag endpoints use oapi-codegen. The `/services` and `/compliance` endpoints are hand-written.
- Two config files in `controllers/` drive generation:
  - `types.cfg.yaml` — generates `api/types.gen.go` (models only, filtered to `seats` tag).
  - `server.cfg.yaml` — generates `api/server.gen.go` (chi server + strict server + embedded spec, filtered to `seats` tag).
- Generator directives live as `//go:generate` comments at the top of `controllers/seats.go`, not in a separate `generate.go` file.
- Run `make generate` or `go generate ./...` to regenerate. Generated `*.gen.go` files are gitignored.
- Generator version is pinned: `github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@v2.0.0`. Do not change without coordinating.
- The seat manager feature is disabled by default (`DisableSeatManager` defaults to `true` in `config/main.go`). Generated seat endpoints are only registered when this flag is `false`.

### Adding a new oapi-codegen endpoint

1. Add the path and schemas to `apispec/api.spec.json`.
2. Tag the operation with `seats` to include it in generation, or add a new `include-tags` entry to both cfg files.
3. Run `make generate`.
4. Implement the new method on `SeatManagerApi` to satisfy `api.ServerInterface`.

## Two API Styles in One Codebase

### Generated (seats)

- `SeatManagerApi` implements `api.ServerInterface` (compile-time check: `var _ api.ServerInterface = &SeatManagerApi{}`).
- Registered via `api.HandlerFromMuxWithBaseURL(seatManagerApi, r.With(enforceIdentity), "/api/entitlements/v1")` — identity enforcement is applied inline at registration, not as a separate middleware step.
- Request/response types come from `api` package (generated). Use pointer fields with the `toPtr[T]` helper.
- Query params arrive as generated `api.GetSeatsParams` struct; apply `fillDefaults()` for nil optional fields.
- Errors use `doError()` which maps through `SeatsErrorMapper` to produce `api.Error` JSON responses.

### Hand-written (services, compliance)

- Registered as plain `http.HandlerFunc` on the chi router in `server/routes.go`.
- Query params are parsed manually via `req.URL.Query().Get()` helpers (`filtersFromParams`, `boolFromParams`).
- Request/response types are defined in `types/` package with `json` struct tags using `snake_case`.
- Errors use endpoint-specific helpers (`failOnDependencyError`, `failOnBadRequest`, `failOnComplianceError`).

## Schema Conventions

- All JSON field names use `snake_case` (e.g., `is_entitled`, `account_username`, `subscription_id`).
- Query parameter names are `snake_case` for hand-written endpoints (e.g., `include_bundles`, `trial_activated`) but `camelCase` for generated seats endpoints (e.g., `accountUsername`, `firstName`, `lastName`). Follow whichever convention the endpoint style uses.
- Enum values use `PascalCase` for status values (`Active`, `Deprovisioned`) and `snake_case` for sort fields.
- Use `x-enum-varnames` in the spec to control generated Go constant names.
- Array query params use `style: form` with `explode: false` (comma-separated).

## Pagination

- Pagination uses `offset`/`limit` pattern (not cursor-based).
- Default limit: 10. Min: 1. Max: 1000. Default offset: 0. Min: 0.
- Paginated responses use `ListPagination` (contains `meta.count` and `links.{first,previous,next,last}`).
- Pagination links follow the format `/api/entitlements/v1/seats/?limit=N&offset=M`.
- `last` link is omitted when the upstream (AMS) does not provide total count.

## Error Response Shapes

Three distinct error shapes exist — use the correct one for context:

| Schema | When to use | Fields |
|---|---|---|
| `api.Error` (generated) | Seats endpoints | `error`, `status`, `identifier`, `code`, `operation_id` |
| `DependencyErrorResponse` | Upstream service failure | `error.{dependency_failure, service, status, endpoint, message}` |
| `RequestErrorResponse` | Bad request to entitlements | `error.{status, message}` |

Do not introduce new error shapes. Map upstream errors (AMS, BOP) through `SeatsErrorMapper`.

## Response Headers

- Prefer setting `Content-Type: application/json` before writing the response body. Note: the hand-written error helpers (`failOnDependencyError`, `failOnBadRequest`, `failOnComplianceError`) use `http.Error()`, which sets `Content-Type: text/plain; charset=utf-8` instead. New error helpers should set the header explicitly rather than relying on `http.Error()`.
- `/services` sets `X-Entitlements-Degraded: true` and `X-Entitlements-Degraded-Status: <code>` when upstream calls fail but the request still returns 200 with degraded data.

## Identity and Authorization

- All authenticated routes use `identity.EnforceIdentityWithLogger` middleware (from `platform-go-middlewares/v2`).
- Identity is extracted via `identity.GetIdentity(req.Context()).Identity`.
- Service Accounts (`idObj.User == nil`) are handled explicitly — they cannot perform org-admin actions or compliance screening.
- Org-admin checks (`idObj.User.OrgAdmin`) gate write operations on seats (POST, DELETE).
- DELETE `/seats/{id}` additionally verifies the subscription's AMS org matches the caller's org.

## Bundle Configuration

- Bundle definitions live in `bundles/bundles.yml` (gitignored; `bundles.example.yml` is committed).
- Bundle YAML schema: `name`, `use_valid_acc_num`, `use_valid_org_id`, `use_is_internal`, `skus`, `eval_skus`, `paid_skus`.
- Adding a new bundle does NOT require spec changes — the `/services` response is a dynamic map keyed by bundle name.
- The `Service` schema uses `additionalProperties` referencing `ServiceDetails`, making it an open-ended map.

## Adding a New Endpoint Checklist

1. Add path, parameters, and schemas to `apispec/api.spec.json`.
2. If using codegen: tag appropriately, update cfg files if needed, run `make generate`, implement interface.
3. If hand-written: add types to `types/` package, add handler in `controllers/`, register route in `server/routes.go`.
4. Wrap route with `enforceIdentity` middleware unless it is public (only `/status`, `/metrics`, and `/api/entitlements/v1/openapi.json` are unauthenticated).
5. Use existing error response shapes and helpers — do not create new ones.

## Testing Conventions

- Controller tests use Ginkgo/Gomega (`controllers_suite_test.go`).
- Seats tests mock `ams.AMSInterface` and `bop.Bop` interfaces.
- Subscription tests replace `GetFeatureStatus` (package-level var) for mocking.
