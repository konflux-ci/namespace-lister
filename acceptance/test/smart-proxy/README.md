# Smart Proxy

In this setup the proxy is supposed to implement some sort of authentication logic.

It forwards `/api/v1/namespaces` and `/api/v1/namespace/<namespace_name>` to the Namespace-Lister, whereas all the others to the Kubernetes APIServer.

For each request it will inject the Bearer Token for authenticating as a Cluster Admin ServiceAccount and set the `Impersonate-User` header to the authenticated user.

For simplicity, the User is already expected to be in the `Impersonate-User` header.
So, unauthenticated requests that can impersoante anyone are supported.
Be mindful about it.
