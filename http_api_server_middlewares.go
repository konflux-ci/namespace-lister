package main

import (
	"cmp"
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/authentication/authenticator"
)

// addLogCorrelationIDMiddleware retrieves the correlation ID from the request's header
// X-Correlation-ID and adds it to the logs. If the header is not present, it generates
// a new Correlation-ID.
func addLogCorrelationIDMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get logger from context
		l := getLoggerFromContext(r.Context())

		// get Correlation ID from te request.
		// If not present, generate a new one.
		cid := cmp.Or(r.Header.Get("X-Correlation-ID"), string(uuid.NewUUID()))
		l = l.With("correlation-id", cid)

		// run the next handler
		ctx := setLoggerIntoContext(r.Context(), l)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// addInjectLoggerMiddleware injects the provided logger in each request context.
func addInjectLoggerMiddleware(l slog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := setLoggerIntoContext(r.Context(), &l)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// addLogRequestMiddleware logs before and after each request for debugging purposes.
func addLogRequestMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := getLoggerFromContext(r.Context()).With("request", r.URL.Path)

		next.ServeHTTP(w, r)
		l.Info("request processed")
	}
}

// addAuthnMiddleware authenticates requests
func addAuthnMiddleware(ar authenticator.Request, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs, ok, err := ar.AuthenticateRequest(r)

		switch {
		case err != nil: // error contacting the APIServer for authenticating the request
			w.WriteHeader(http.StatusUnauthorized)
			l := getLoggerFromContext(r.Context())
			l.Error("error authenticating request", "error", err, "request-headers", r.Header)
			return

		case !ok: // request could not be authenticated
			w.WriteHeader(http.StatusUnauthorized)
			return

		default: // request is authenticated
			// Inject authentication details into request context
			ctx := r.Context()
			authCtx := context.WithValue(ctx, ContextKeyUserDetails, rs)
			getLoggerFromContext(r.Context()).With("user", rs).Debug("request authenticated")

			// serve next request
			next.ServeHTTP(w, r.WithContext(authCtx))
		}
	}
}

// addMetricsMiddleware adds a set of middlewares that collect metrics for each requests
func addMetricsMiddleware(reg prometheus.Registerer, handler http.Handler) http.Handler {
	if reg == nil {
		return handler
	}

	m := newHTTPMetrics(reg)
	return promhttp.InstrumentHandlerDuration(
		m.requestTiming,
		promhttp.InstrumentHandlerCounter(
			m.requestCounter,
			promhttp.InstrumentHandlerResponseSize(
				m.responseSize,
				promhttp.InstrumentHandlerInFlight(
					m.inFlightGauge,
					handler))))
}
