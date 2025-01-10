# Smart Proxy

In this setup the proxy is supposed to implement some sort of authentication logic.

It forwards `/api/v1/namespaces` and `/api/v1/namespace/<namespace_name>` to the Namespace-Lister, whereas all the others to the Kubernetes APIServer.

For each request it will inject the Bearer Token for authenticating as a Cluster Admin ServiceAccount and set the `Impersonate-User` header to the authenticated user.

For simplicity, the User is already expected to be in the `Impersonate-User` header.
So, unauthenticated requests that can impersonate anyone are supported.

## Prepare

The `prepare` target will deploy an NGINX Proxy that is in charge of authenticating user requests.

To forward authentication details to the API Server, a token is required from the user.
This means that the default certificate-based authentication is not supported.

Token validation is not really implemented in the Proxy, as it is not required for testing purposes.
This means that any request to `/api/v1/namespaces` will just work, the only required field is the `Impersonate-User` Header.

## Tests

### Manual testing

In this setup any request to `/api/v1/namespaces` with the `Impersonate-User` Header will just work.
You can just forge a new request with the header value set to whichever user you want to impersonate:

```
curl -sk -X GET https://localhost:10443/api/v1/namespaces -H 'Impersonate-User: any-user-i-like'
```

The other requests will be forwarded to the Kubernetes APIServer, thus they'll also be validated.

### Test suite

To execute the tests, you can use the `make test` command.
