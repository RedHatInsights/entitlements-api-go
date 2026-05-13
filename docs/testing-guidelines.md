# Testing Guidelines

## Framework and Libraries

- **Test framework**: Ginkgo v2 (`github.com/onsi/ginkgo/v2`) with Gomega matchers (`github.com/onsi/gomega`).
- **HTTP mock servers**: `github.com/onsi/gomega/ghttp` for faking external HTTP services (AMS, subscriptions, token servers).
- **HTTP test infrastructure**: `net/http/httptest` for recording handler responses.
- All Ginkgo and Gomega symbols are dot-imported; do not use qualified names.

## Suite Setup

Every package with tests must have a `*_suite_test.go` file following this pattern:

```go
package mypkg

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestMyPkg(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "MyPkg Suite")
}
```

- Packages that use logging must call `InitLogger()` (dot-imported from `logger`) before `RunSpecs`.
- Suite-level config (e.g., timeouts) goes in `BeforeSuite`, not in the test function.

## Running Tests

- `make test` — runs all tests with verbose output.
- `make test-all` — runs with race detector, serial packages (`-p 1`), and coverage.
- `make bench` — runs benchmarks.
- Tests require `make generate` first (codegen from OpenAPI spec); both `make test` and `make test-all` handle this automatically.

## Test Structure Conventions

- Use `Describe` for the top-level component/controller under test.
- Use `Context` for preconditions ("When X is Y").
- Use `When` as an alias for `Context` — both are used interchangeably throughout the codebase.
- Use `It` for individual assertions.
- Use `BeforeEach` to reset state (bundle info, mock servers, recorder instances).
- Use `AfterEach` for cleanup (closing servers, unsetting env vars).
- Arrange/Act/Assert sections in `It` blocks use `// given`, `// when`, `// then` comments.

## Mock Patterns

### Interface-Based Mocks (AMS, BOP)

Mocks live in the **same package** as the real implementation, not in test files.

- `ams.Mock` implements `ams.AMSInterface`. Its methods delegate to **package-level `var` functions** (e.g., `ams.MockGetSubscriptions`) that tests can reassign.
- `bop.Mock` implements `bop.Bop` with a configurable `OrgId` field.
- Both use compile-time interface checks: `var _ AMSInterface = &Mock{}`.

To override mock behavior in a test, reassign the package-level var:

```go
ams.MockGetSubscriptions = func(organizationId string, searchParams api.GetSeatsParams, size, page int) (*v1.SubscriptionList, error) {
    // custom behavior
}
```

### Function Variable Mocking (Controllers)

`controllers.GetFeatureStatus` is a package-level `var` holding a function. Tests swap it by assigning a fake:

```go
GetFeatureStatus = func(params GetFeatureStatusParams) FeatureResponse {
    return FeatureResponse{StatusCode: 200, Data: FeatureStatus{}, CacheHit: false}
}
```

Save and restore the real function when mixing unit and integration tests in the same file:
```go
var realGetFeatureStatus = GetFeatureStatus
// In BeforeEach for integration tests:
GetFeatureStatus = realGetFeatureStatus
```

## HTTP Testing Patterns

### Handler Tests (controllers)

Use helper functions that build `http.Request` with identity context, invoke the handler, and return the recorder + parsed body:

```go
rr, body, rawJSON := testRequest("GET", "/", accNum, orgId, isInternal, email, fakeCaller)
```

- Identity is injected via `identity.WithIdentity(ctx, identity.XRHID{...})` from `platform-go-middlewares/v2/identity`.
- The `MakeRequest` helper uses functional options (`opt` type) to override defaults like `OrgAdmin(false)` or `OrgId("12345")`.
- Separate helpers exist for Service Account identity: `MakeServiceAccountRequest`, `testRequestWithServiceAccount`.

### External Service Mocks (ghttp)

For testing real HTTP client code against AMS or subscriptions services:

```go
server = ghttp.NewServer()
server.AppendHandlers(
    ghttp.CombineHandlers(
        ghttp.VerifyRequest("GET", "/api/path"),
        ghttp.RespondWith(http.StatusOK, `{"key":"value"}`, http.Header{"Content-Type": {"application/json"}}),
    ),
)
// Point config at mock server
config.GetConfig().Options.SetDefault(config.Keys.AMSHost, server.URL())
```

- Always close ghttp servers in `AfterEach`.
- Use `server.ReceivedRequests()` to assert call counts (e.g., verifying cache behavior).
- Use `server.Writer = GinkgoWriter` to route server logs to Ginkgo output.

## Config in Tests

- Use `config.GetConfig().Options.Set(key, value)` or `.SetDefault(key, value)` to configure test values.
- Config keys are accessed via `config.Keys.*` constants.
- When setting env vars (`os.Setenv`), always restore them in `AfterEach`.

## Test Data

- Static test fixtures live in `/test_data/` at the repo root (e.g., `test_bundle.yml`, `err_bundle.yml`, cert files).
- Reference them with relative paths from the test package: `"../test_data/test_bundle.yml"`.
- For dynamic test data, use `os.CreateTemp` and clean up with `defer os.Remove(path)` or `AfterEach`.

## Benchmarks

Benchmarks use standard `testing.B` and are placed alongside Ginkgo tests in the same file. They can reuse test helpers:

```go
func BenchmarkRequest(b *testing.B) {
    b.ResetTimer()
    for n := 0; n < b.N; n++ {
        testRequestWithDefaultOrgId("GET", "/", fakeCaller)
    }
}
```

## Error Assertion Patterns

- Use `BeAssignableToTypeOf` to assert custom error types, then `errors.As` to inspect fields:
  ```go
  var clientError *ClientError
  Expect(err).To(BeAssignableToTypeOf(clientError))
  errors.As(err, &clientError)
  Expect(clientError.StatusCode).To(BeEquivalentTo(http.StatusBadRequest))
  ```
- Use `Expect(err).To(BeNil())` for success paths (not `Succeed()`).
- Use `Expect(err).To(HaveOccurred())` or `Expect(err).ToNot(HaveOccurred())` for error checks on non-nil errors.

## Common Matchers

| Pattern | Usage |
|---|---|
| `Equal(expected)` | Exact value match (used for status codes, strings, booleans) |
| `BeEquivalentTo(expected)` | Type-coercing equality (used for comparing string/int variants) |
| `ContainSubstring(s)` | Partial string match in error messages |
| `HaveLen(n)` | Collection length |
| `HaveKey(k)` | Map key existence |
| `ContainElement(e)` | Slice membership |
| `HaveExactElements(...)` | Ordered slice match |
| `BeTrue()` / `BeFalse()` | Boolean assertions |
| `BeNil()` / `BeEmpty()` | Nil/empty checks |
| `BeIdenticalTo(x)` | Pointer identity (singleton verification) |
