# Acceptance Tests

Behavior-Driven Development is enforced through [godog](https://github.com/cucumber/godog).

These tests has builtin support to run on [kind](https://kind.sigs.k8s.io/).

## Setups

The Namespace-Lister is usually installed behind a Proxy.
The proxy forwards the `/api/v1/namespaces` ones to the Namespace-Lister and the others to the Kubernetes APIServer.

The Namespace-Lister can be configured to delegate authentication to the Proxy.
In this case we speak of a [Smart Proxy](./test/smart-proxy/).

Alternatively, the request is authenticated against the APIServer's TokenReview API.
In this case we speak of a [Dumb Proxy](./test/dumb-proxy/).

We support test cases for both the setups.

To create the cluster, install the Namespace-Lister, and configure the Proxy you can use the `make prepare` command.
> The preparation phase is different for the two setups.
> For details, refer to the documentation of each setup.

After you have prepared the cluster, you can execute the tests by running `make test`.

