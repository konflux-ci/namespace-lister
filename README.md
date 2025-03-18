# Namespace-Lister

The Namespace-Lister is a simple REST Server that implements the endpoint `/api/v1/namespaces`.
It returns the list of Kubernetes namespaces the user has `get` access on.

## Requests Authentication

Requests authentication is **out of scope**.

The namespace-lister requires requests to be already authenticated or that their authentication can deferred to the Kubernetes APIServer via the TokenAccessReview API.

### Already authenticated requests

Another component (e.g. a reverse proxy) is required to implement authentication.

The Namespace-Lister will retrieve the user information from an HTTP Header.
It is possible to declare which Header to use via Environment Variables.

### TokenAccessReview API

The namespace-lister can defer the request authentication to the Kubernetes APIServer leveraging on the TokenAccessReview API.

For this mechanism to work, the request is required to have a bearer token.

## How it builds the reply

For performance reasons, the Namespace-Lister caches Namespaces, Roles, ClusterRoles, RoleBindings to perform in-memory authorization.

For each requests it looks into a cache of already calculated subject accesses.
The cache is invalidated and updated for each event on the cached resources, or when a resync period elapses.

Users will be provided with all the Namespaces on which a RoleBinding is providing them `get` access to.
To grant a user the `get` access to a Namespace, a (Cluster)Role can be used together with a RoleBinding.

In the following an example using a ClusterRole:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: namespace-get
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: user-access
  namespace: my-namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: namespace-get
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: user
```

## Tests

Acceptance tests are implemented in the [acceptance folder](./acceptance/).

Behavior-Driven Development is enforced through [godog](https://github.com/cucumber/godog).
You can find the specification of the implemented Features at in the [acceptance/features folder](./acceptance/features/).

## Try

The easiest way to try this component locally is by using the `make prepare` target in `acceptance/test/dumb-proxy` or `acceptance/test/smart-proxy`.
These commands will build the image, create the Kind cluster, load the image in it, and deploy all needed components.

Please take a look at the [Acceptance Tests README](./acceptance/README.md) for more information on the two setups and how to access the namespace-lister once deployed.

