# Dumb Proxy

In this setup the proxy is not implementing any authentication logic.

It forwards `/api/v1/namespaces` and `/api/v1/namespace/<namespace_name>` to the Namespace-Lister, whereas all the others to the Kubernetes APIServer.

## Prepare

The `prepare` target will deploy an NGINX Proxy that will only route requests to Namespace-Lister or Kubernetes APIServer.

The Namespace-Lister is configured to authenticate the request by forwarding the request's Bearer Token to the Kubernetes APIServer.

This means that for a request to `/api/v1/namespaces` to work, a Bearer Token recognized as valid from the Kubernetes APIServer is required.
