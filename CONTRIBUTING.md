# Contributing to entitlements-api-go

## Development Environment Setup

See the [README](README.md) for full instructions on installing Go, cloning the repo, configuring certificates, and running the application locally.

Quick start:

```bash
go get ./...
cp bundles/bundles.example.yml bundles/bundles.yml
make generate
```

## Making Changes

### Branching

- Create a feature branch from `main`.
- Use descriptive branch names (e.g., `add-compliance-endpoint`, `fix-bundle-parsing`).

### Commit Messages

- Write clear, imperative-mood commit messages (e.g., "Add compliance timeout config" not "Added compliance timeout config").
- Keep the subject line under 72 characters. Use the body for additional context when needed.

### Code Style

- Follow existing naming conventions: snake_case filenames, single-word lowercase package names, role-based interface names (not `I`-prefixed). See [AGENTS.md](AGENTS.md) for the full list.
- Use `fmt.Errorf` with `%w` for error wrapping — do not use `pkg/errors`.
- Use `controllers.getClient()` for HTTP clients — do not create new `http.Client` instances.
- Singleton package-level vars (`bundleInfo`, `featuresQuery`, the HTTP client) are set once at startup. Do not mutate them after initialization.
- Generated files (`*.gen.go`) are gitignored. Never commit or edit them directly.

## Running Tests

Always run tests before submitting a PR:

```bash
# Standard test run
make test

# Full CI-equivalent run (race detector, serial execution, coverage)
make test-all

# Benchmarks
make bench
```

Both `make test` and `make test-all` run `make generate` automatically. If you have modified the OpenAPI spec (`apispec/api.spec.json`), the generated types will be rebuilt before tests run.

For detailed information on the test framework (Ginkgo/Gomega), mock patterns, and HTTP testing helpers, see [docs/testing-guidelines.md](docs/testing-guidelines.md).

## Pull Request Expectations

### CI Checks

Every PR must pass the following before merge:

- **Konflux/Tekton pipeline** (`.tekton/`): Runs `make generate` then `make test-all` with the race detector. This is the primary CI gate.
- **GitHub Actions** (`.github/workflows/`): Security scanning (Grype/Syft), JSON/YAML validation, and PR labeling. These do not run Go tests but must pass.

### PR Template Checklist

The PR template includes a Secure Coding Practices Checklist. Review each item and check the ones relevant to your change. Security-related items (input validation, access control, error handling) are especially important for this service since it handles identity and entitlement data.

### Review Process

- At least one approving review is required.
- Address all reviewer feedback before merge.
- PRs are merged to `main`, which automatically triggers the Konflux build and stage deployment pipeline.

## Adding New Endpoints or Service Integrations

- For new API endpoints, prefer the **generated (oapi-codegen) style** over hand-written handlers. See [docs/api-contracts-guidelines.md](docs/api-contracts-guidelines.md) for the OpenAPI spec workflow and schema conventions.
- For new external service integrations (HTTP clients, resilience patterns, caching), see [docs/integration-guidelines.md](docs/integration-guidelines.md).
- Review [docs/security-guidelines.md](docs/security-guidelines.md) for identity/auth middleware requirements and input validation rules.
- Review [docs/error-handling-guidelines.md](docs/error-handling-guidelines.md) for logging and error response conventions.

## Notes for AI Agents

If you are an AI agent or using AI-assisted development tools, read [AGENTS.md](AGENTS.md) before making changes. It contains project structure, architectural context, common pitfalls, and conventions specific to this codebase.
