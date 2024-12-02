# Dumb Proxy

In this setup the proxy is not implementing any authentication logic.

It forwards `/api/v1/namespaces` and `/api/v1/namespace/<namespace_name>` to the Namespace-Lister, whereas all the others to the Kubernetes APIServer.
