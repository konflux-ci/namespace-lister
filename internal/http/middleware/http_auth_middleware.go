package middleware

import (
	"context"
	"net/http"

	"k8s.io/apiserver/pkg/authentication/authenticator"

	"github.com/konflux-ci/namespace-lister/internal/contextkey"
	"github.com/konflux-ci/namespace-lister/internal/log"
)

// AddAuthnMiddleware authenticates requests
func AddAuthnMiddleware(ar authenticator.Request, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs, ok, err := ar.AuthenticateRequest(r)

		switch {
		case err != nil: // error contacting the APIServer for authenticating the request
			w.WriteHeader(http.StatusUnauthorized)
			l := log.GetLoggerFromContext(r.Context())
			l.Error("error authenticating request", "error", err)
			return

		case !ok: // request could not be authenticated
			w.WriteHeader(http.StatusUnauthorized)
			return

		default: // request is authenticated
			// Inject authentication details into request context
			ctx := r.Context()
			authCtx := context.WithValue(ctx, contextkey.ContextKeyUserDetails, rs)
			log.GetLoggerFromContext(r.Context()).With("user", rs).Debug("request authenticated")

			// serve next request
			next.ServeHTTP(w, r.WithContext(authCtx))
		}
	}
}
