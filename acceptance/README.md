# Acceptance Tests

Behavior-Driven Development is enforced through [godog](https://github.com/cucumber/godog).

These tests has builtin support to run on [kind](https://kind.sigs.k8s.io/).

## Setups

The Namespace-Lister is usually installed behind a Proxy.
The Namespace-Lister can be configured to delegate authentication to the Proxy.
In this case we speak of a `Smart Proxy`.

Alternatively, the request is authenticate against the APIServer's TokenReview API.
In this case we speak of a `Dumb Proxy`.

We support test cases for both these setups.
You find the `Smart Proxy`'s tests at [./test/smart-proxy/] and the `Dumb Proxy`'s tests at [./test/dumb-proxy/].

To create the cluster, install the Namespace-Lister, and configure the Proxy you can use the `make prepare` command.

To execute the tests, you can use the `make test` command.

