---
name: ci-troubleshooting
description: >
  Use when a CI check fails on a PR in namespace-lister and you need to
  understand what failed, how to read the logs, and how to fix it.
---

# CI Troubleshooting

## Overview

How to investigate and fix CI failures on namespace-lister PRs.

## When to Use

- A CI check failed on your PR
- You need to understand what a CI comment or status means
- You want to re-trigger a flaky test

## Prerequisites

Verify `gh` CLI is installed and authenticated:

```bash
gh auth status
```

All CI investigation commands below depend on it.

## Reading CI Logs

### GitHub Actions checks

```bash
gh pr checks <PR-number> --repo konflux-ci/namespace-lister
```

To investigate a failed check:

```bash
gh run view <run-id> --repo konflux-ci/namespace-lister
gh run view <run-id> --repo konflux-ci/namespace-lister --log-failed
```

### Tekton / Konflux pipeline checks

Tekton pipeline runs (pull-request and push pipelines defined in `.tekton/`) execute inside the Konflux platform. These are best investigated manually through the Konflux UI.

If a Tekton pipeline check fails, you can comment `/retest` on the PR to re-trigger, but if the failure persists, escalate to a human who can inspect the logs in the Konflux UI.

## Common Failures

### unit-tests

Ginkgo test failures. The GitHub Actions log shows which specs failed and the failure messages. To reproduce locally:

```bash
make test
```

If a specific test is failing, run it in isolation:

```bash
go run github.com/onsi/ginkgo/v2/ginkgo --focus "description of failing test" ./...
```

### performance-tests

Ginkgo performance tests using kwokctl. These require kwokctl to be installed. If failed, check if it's a flaky infrastructure issue (kwokctl cluster setup) or a real regression. To reproduce locally:

```bash
make test-perf
```

### acceptance

Matrix of `dumb-proxy` and `smart-proxy` acceptance test setups running on Kind clusters with godog (Cucumber BDD). Both setups must pass.

Common causes of failure:
- Container image build failure (check Podman/Docker availability)
- Kind cluster setup failure (infrastructure flake — rerun)
- Actual test scenario failure (check the godog output for the failing step)

To reproduce locally:

```bash
make -C acceptance/test/dumb-proxy prepare && make -C acceptance/test/dumb-proxy test
make -C acceptance/test/smart-proxy prepare && make -C acceptance/test/smart-proxy test
```

If logs show no relevant errors and the failure looks intermittent, rerun the failed job:

```bash
gh run rerun <run-id> --repo konflux-ci/namespace-lister --failed
```

### go-tidy

`go.mod` or `go.sum` are not tidy. Fix:

```bash
go mod tidy
```

Then commit the changes.

### lint-go

golangci-lint violations. The log shows the exact file, line, and linter rule. Fix the reported issues or run locally:

```bash
make lint-go
```

### lint-yaml

YAML formatting errors (trailing whitespace, wrong indentation, missing newline at end of file). Fix the reported issues or run locally:

```bash
make lint-yaml
```

### Tekton pipeline failures

The `.tekton/` pipelines run security scans (Clair, Snyk, Coverity, ClamAV, SAST) and multi-arch container builds. These run inside Konflux and their logs are not accessible from the CLI. If they fail:

1. Comment `/retest` on the PR to re-trigger.
2. If the failure persists, these are best investigated manually through the Konflux UI by a human.
