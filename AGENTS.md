# namespace-lister

Go REST service (`/api/v1/namespaces`) that returns Kubernetes namespaces
a user has `get` access on, using in-memory RBAC caching for performance.

## Quick Commands

| Action            | Command                                    |
|-------------------|--------------------------------------------|
| Build             | `go build -o bin/namespace-lister .`       |
| Unit tests        | `make test`                                |
| Performance tests | `make test-perf`                           |
| Acceptance tests  | `make -C acceptance/test/dumb-proxy prepare && make -C acceptance/test/dumb-proxy test` |
| Lint              | `make lint`                                |
| Format            | `make fmt`                                 |

## Project Layout

- `*.go` (root) — core server: authenticator, RBAC authorizer, HTTP handler,
  namespace lister, metrics, environment config.
- `internal/` — `constants`, `contextkey`, `http`, `log`, `resourcecache`.
- `pkg/` — public packages: `auth`, `metricsutil`.
- `acceptance/` — godog BDD tests; two setups: `dumb-proxy` and `smart-proxy`.
- `config/` — Kustomize deployment manifests.

## Key Conventions

- Authentication is out of scope — pre-authenticated headers or bearer token
  for TokenAccessReview.
- Ginkgo for unit/perf tests; godog (Cucumber BDD) for acceptance tests.
- Performance tests use kwokctl for lightweight cluster simulation.
- golangci-lint version pinned in Makefile (`GOLANGCI_LINT_VERSION`).
- Coverage instrumentation toggled via `ENABLE_COVERAGE=true`.
- All changes via PR; OWNERS approval required.

## Testing

- **Unit tests**: Ginkgo-based, run with `make test`. Cover RBAC logic,
  HTTP handlers, authenticator, and metrics.
- **Performance tests**: Ginkgo with kwokctl cluster. Require kwokctl.
- **Acceptance tests**: godog BDD with Kind cluster. Two setups: `dumb-proxy`
  (direct) and `smart-proxy` (behind proxy). `make prepare` builds image,
  creates Kind cluster, and deploys all components.
- Coverage collected via coverport-cli and uploaded to Codecov.

## CI Pipeline (GitHub Actions)

- `unit-tests` — Ginkgo tests with coverage, uploads to Codecov.
- `performance-tests` — perf tests with kwokctl.
- `acceptance` — matrix of `dumb-proxy`/`smart-proxy` on Kind, e2e coverage.
- `go-tidy` — verifies `go.mod` and `go.sum` are tidy.
- `lint-go` — golangci-lint (version from Makefile).
- `lint-yaml` — yamllint on all YAML manifests.
- `dep-triage` — auto-triages Renovate/Konflux bot dependency PRs.
- `auto-merge` — merges approved dependency PRs when all checks pass.

## Gotchas

- Acceptance tests build/load a container image into Kind — requires Docker
  or Podman (`IMAGE_BUILDER` variable).
- Both `dumb-proxy` and `smart-proxy` acceptance setups must pass — they
  test different authentication flows.
- Coverage collection requires coverport-cli and a running instrumented pod.
