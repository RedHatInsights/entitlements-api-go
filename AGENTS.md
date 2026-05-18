# AGENTS.md

## Docs Index

| Document | Description |
|----------|-------------|
| [docs/security-guidelines.md](docs/security-guidelines.md) | Identity/auth middleware, input validation, TLS configuration, secrets handling, fail-closed behavior, and container security |
| [docs/performance-guidelines.md](docs/performance-guidelines.md) | Caching strategy (ccache), HTTP client singletons, Prometheus metrics naming/registration, concurrency model, and dependency resilience |
| [docs/error-handling-guidelines.md](docs/error-handling-guidelines.md) | Logging conventions (logrus), Sentry/Glitchtip integration, custom error types, error mapping, wrapping patterns, and degraded mode |
| [docs/api-contracts-guidelines.md](docs/api-contracts-guidelines.md) | OpenAPI spec management, oapi-codegen workflow, two API styles (generated vs hand-written), schema conventions, pagination, and error response shapes |
| [docs/testing-guidelines.md](docs/testing-guidelines.md) | Ginkgo/Gomega framework, suite setup, mock patterns (interface-based and function variable), HTTP testing helpers, and common matchers |
| [docs/integration-guidelines.md](docs/integration-guidelines.md) | External service clients (AMS, BOP, Feature Service, Compliance), client architecture, HTTP client conventions, resilience patterns, and AMS query builder |

## Cross-Cutting Conventions

### Project Structure

```
entitlements-api-go/
  main.go                  # Application entry point: logger init, Sentry init, bundle loading, server launch
  ams/                     # AMS client, interface, mock, and query builder
  api/                     # Generated code from oapi-codegen (*.gen.go, gitignored)
  apispec/                 # OpenAPI 3.0 spec and spec-serving handler
  bop/                     # Back Office Proxy client, interface, and mock
  bundle_sync/             # Standalone CLI tool for syncing bundle SKU config to Feature Service
  bundles/                 # Bundle YAML config (bundles.yml gitignored; bundles.example.yml committed)
  config/                  # Viper-based configuration singleton and certificate loading
  controllers/             # HTTP handlers, shared HTTP client, error mapper, oapi-codegen config files
  dashboards/              # Grafana dashboard YAML
  deployment/              # ClowdApp OpenShift deployment template
  docs/                    # Domain guideline documents
  logger/                  # Global logrus logger singleton with CloudWatch hook
  server/                  # Router setup (chi), Prometheus middleware, and server launcher
  types/                   # Shared domain types (bundles, features, error responses, compliance)
  test_data/               # Static test fixtures (certs, bundle YAML files)
  scripts/                 # Helper scripts
  .tekton/                 # Konflux/Tekton pipeline definitions
```

### Naming Conventions

- **Packages**: Single lowercase words matching directory names (`ams`, `bop`, `config`, `types`, `logger`). The `api` package is reserved for generated code.
- **Files**: Snake_case for multi-word filenames (`mock_client.go`, `seats_error_mapper.go`). The entry point for each package is `main.go` or `client.go`.
- **Interfaces**: Named after their role, not prefixed with `I`. Examples: `AMSInterface`, `Bop`, `SeatsErrorMapper`.
- **Structs**: `Client` for real implementations, `Mock` for test doubles — both in the same package.
- **Config keys**: `UPPER_SNAKE_CASE` strings in `config.Keys`, mapped to `ENT_`-prefixed env vars.

### Code Style Patterns

- **Singleton initialization**: `config.GetConfig()`, `logger.InitLogger()`, and `controllers.getClient()` use lazy-init package-level vars.
- **Functional options**: The `MakeRequest` test helper uses an `opt` function type to override identity fields.
- **Generic helper**: `toPtr[T]` in `controllers/seats.go` converts values to pointers for generated API types.
- **Import aliases**: `l` for logger in controller files, `log` in `server/routes.go`, `v1` for OCM SDK types, `ocmErrors` for OCM error types.

### Build and Development Workflow

- **`make generate`**: Regenerates `api/types.gen.go` and `api/server.gen.go`. Run before tests or builds after modifying the spec.
- **`make test`** / **`make test-all`**: `test-all` adds race detector, serial execution, and coverage. Both run `generate` first.
- **`make debug-run`**: Runs with `ENT_DEBUG=1` (enables mock AMS/BOP clients — local dev only).
- Generated files (`*.gen.go`) are gitignored. Never commit them and never edit them directly.

### Common Pitfalls

- **Forgetting `make generate`**: Tests fail if generated files are missing. Always run `make generate` after cloning or changing the spec.
- **Mutating package-level vars after startup**: `bundleInfo`, `featuresQuery`, `paidFeatureSuffix`, and the HTTP client singleton are set once — mutating them post-launch causes races.
- **Nil `User` on identity**: Service Accounts have `idObj.User == nil`. Always nil-check before accessing any `User.*` field.
- **`http.Client` creation**: Use `controllers.getClient()` for Feature Service and Compliance — don't create new clients.
- **`errors.Wrap` from `pkg/errors`**: Not a dependency. Use only `fmt.Errorf` with `%w`.

### CI/CD Pipeline

- **Konflux/Tekton** (`.tekton/`): Primary CI — runs `make generate` then `make test-all` with race detector.
- **GitHub Actions** (`.github/workflows/`): Security scanning (Grype/Syft), JSON/YAML validation, PR labeling. Does not run Go tests.
- **PR template**: Includes a secure coding checklist. Address relevant items in your PR.

## Architectural Context

### Two Binaries, Shared Packages

1. **`entitlements-api-go`** (`main.go`): The HTTP API server.
2. **`bundle-sync`** (`bundle_sync/main.go`): CLI init container that syncs bundle SKU config to Feature Service. Uses stdlib `log`/`fmt`, not the structured logger. Supports `--dry-run`.

### Why Two API Styles

- **Generated (seats)**: Added later using oapi-codegen for type-safe code generation. The seat manager is disabled by default (`DisableSeatManager: true`) and is marked as obsolete.
- **Hand-written (services, compliance)**: Predates code generation; uses plain `http.HandlerFunc` and the `types/` package.

When adding new endpoints, prefer the generated style. See `docs/api-contracts-guidelines.md`.

### Configuration System

- All env vars prefixed with `ENT_` via Viper `SetEnvPrefix`. Exception: `GLITCHTIP_DSN` read directly via `os.Getenv`.
- **`Set` vs `SetDefault`**: `SetDefault` allows env override; `Set` does not. `PaidFeatureSuffix` uses `Set` intentionally.
- **Clowder integration**: CloudWatch logging config is overridden from Clowder-provided config in managed environments.
- Certificates load from `/certificates` volume mount or env vars (`ENT_CERTS_FROM_ENV=true`). Startup panics if certs cannot be loaded.

### Middleware Stack Order (outermost first)

1. `prometheusMiddleware` — Request duration and response status metrics
2. `middleware.RequestID` — Unique request IDs
3. `middleware.RealIP` — Real client IP extraction
4. `middleware.Recoverer` — Panic recovery (outermost)
5. `chilogger` — Request logging via logrus
6. `sentryhttp` — Sentry panic capture with `Repanic: true`
7. `identity.EnforceIdentityWithLogger` — Per-route via `r.With(enforceIdentity)` (not global)

### Package Responsibilities

| Package | Responsibility |
|---------|---------------|
| `config` | Centralized configuration and TLS certificate loading |
| `logger` | Global structured logger with CloudWatch integration |
| `server` | Router setup, middleware wiring, Prometheus middleware |
| `controllers` | All HTTP handler logic, shared mTLS HTTP client, error mapping |
| `ams` | AMS client (OCM SDK), org ID conversion/caching, query builder |
| `bop` | Back Office Proxy client for user lookups |
| `types` | Shared domain types for bundles, features, and error responses |
| `api` | Generated types and server interface from OpenAPI spec |
| `apispec` | Serves the OpenAPI spec file at runtime |
