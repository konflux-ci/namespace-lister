# Namespace-Lister

The Namespace-Lister is a simple REST Server that implements the endpoint `/api/v1/namespaces`.
It returns the list of Kubernetes namespaces the user has `get` access on.

## Requests Authentication

Requests authentication is **out of scope**.
Another component (e.g. a reverse proxy) is required to implement authentication.

The Namespace-Lister will retrieve the user information from an HTTP Header.
It is possible to declare which Header to use via Environment Variables.

## How it builds the reply

For performance reasons, the Namespace-Lister caches Namespaces, Roles, ClusterRoles, RoleBindings, and ClusterRoleBindings and performs in-memory authorization.

For each request it loops on all existing Namespaces and returns the namespaces on which the user has `get` access to.
To grant `get` access to a Namespace to a user, a ClusterRole or a Role can be used together with a RoleBinding.

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

