# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Entitlements Service is a Go-based API that acts as a proxy to various backend Red Hat IT services. It manages user entitlements for Red Hat products including:
- `/api/entitlements/v1/services`: Query subscriptions a user is entitled to
- `/api/entitlements/v1/compliance`: Query user compliance checks
- Note: `/seats` APIs are obsolete and no longer enabled in production

## Architecture

The codebase follows a modular architecture with clean separation of concerns:

- **Entry Points:**
  - `main.go`: Initializes logging, Sentry, bundle information, and launches the server
  - `server/main.go`: Configures routes and starts the HTTP server

- **Key Packages:**
  - `config/`: Environment-based configuration using Viper, manages certificates and defaults
  - `controllers/`: Business logic and request handlers for services, compliance, seats, etc.
  - `server/`: HTTP server setup, route definitions, and middleware configuration
  - `ams/`: Account Management System client
  - `bop/`: Back Office Proxy client
  - `types/`: Shared type definitions
  - `logger/`: Centralized logging configuration

- **Router & Middleware:**
  - Uses Chi router for HTTP routing
  - Middleware stack includes: Prometheus metrics, request ID generation, real IP detection, recovery, Logrus logging, Sentry error tracking, identity enforcement

- **Data Layer:**
  - No traditional database ORM; instead uses external service clients (AMS, BOP)
  - In-memory caching for subscriptions
  - Configuration-driven data management

- **Observability:**
  - Prometheus metrics at `/metrics`
  - Sentry for error tracking
  - Logrus for structured logging
  - Service status at `/status`

## Bundle Management

**IMPORTANT:** The `/bundles/bundles.yml` file is for local development only and is git-ignored.
- Copy `/bundles/bundles.example.yml` to `/bundles/bundles.yml` for local testing
- Production SKU/bundle changes MUST be made in the separate `entitlements-config` repository: https://github.com/RedHatInsights/entitlements-config

## Development Commands

### Building & Running

```bash
# Generate OpenAPI types/stubs and build binaries (main app + bundle-sync)
make

# Build only (no code generation)
make build

# Generate only
make generate

# Run with debugging (builds executable first)
make exe

# Run with go run (optimized build)
make run

# Run with debug mode enabled
make debug-run
```

If your local Go version differs from what the project uses (Go 1.24.6), specify the Go binary path:
```bash
make GO=~/go/bin/go1.24
```

### Testing

```bash
# Run unit tests
make test

# Run tests with race detection and coverage (CI mode)
make test-all

# Run benchmarks
make bench
```

### Docker

```bash
make image
docker run -p 3000:3000 entitlements-api-go
```

### Bundle Sync Tool

```bash
make build
./bundle-sync           # Run against configured environment
./bundle-sync --dry-run # Preview changes without posting updates
```

## Configuration

### Required Setup for Local Development

1. **Enterprise Certificate**: Obtain an enterprise services cert with access to dev subscription endpoint and export compliance service. See README.md for detailed cert request process.

2. **Local Config File**: Create `./local/development.env.sh`:
   ```bash
   export ENT_KEY=./{path_to_key}.key
   export ENT_CERT=./{path_to_cert}.crt
   export ENT_CA_PATH=./{path_to_ca_cert}.crt
   export ENT_SUBS_HOST=https://subscription.dev.api.redhat.com
   export ENT_COMPLIANCE_HOST=https://export-compliance.dev.api.redhat.com
   export ENT_DEBUG=true  # Uses mock clients for AMS and BOP
   ```

3. **Source Config Before Running**:
   ```bash
   source ./local/development.env.sh
   make run
   ```

### Configuration Management

- Uses Viper for flexible, environment-variable-driven configuration
- Supports Clowder (cloud-native) configuration in production
- See `config/` package for all configuration options
- Set `ENT_DEBUG=true` to use mock clients instead of real AMS/BOP services

## Code Generation

The project uses code generation from OpenAPI specs:
- Generated files: `api/server.gen.go`, `api/types.gen.go`
- Source: `apispec/api.spec.json`
- Run `make generate` or `go generate ./...` after modifying API specs
- Generated files should NOT be edited manually

## Authentication & Testing

The API requires a valid `x-redhat-identity` header. See `./scripts/xrhid.sh` for examples.

## Degraded State Handling

When dependencies fail during `/api/entitlements/v1/services` calls, the API returns HTTP 200 with degraded state headers:
- `X-Entitlements-Degraded: true`
- `X-Entitlements-Degraded-Status: {status_code}` (or "0" if none received)

All SKU-based bundles default to `is_entitled: false` in degraded mode.

## Deployment

- **Configuration**: Deployment config in `/deployment/clowdapp.yml`
- **CI/CD**: Konflux pipelines auto-trigger on main branch merges, building images to quay.io and deploying to stage
- **Production**: Update image tag in app-interface deployment config to desired commit SHA
- **Certificates**: Stage/prod use automatic cert renewal via AppSRE (certs placed in `/certificates`). Set `ENT_CERTS_FROM_ENV=false` for auto-renewal.

## Version Management

- All releases follow SemVer
- Releases are tagged on main branch (e.g., `v1.16.1`)
- See GitHub releases for release notes
