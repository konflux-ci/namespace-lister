package middleware

import (
	"cmp"
	"log/slog"
	"net/http"

	"github.com/konflux-ci/namespace-lister/internal/log"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// AddLogRequestMiddleware logs before and after each request for debugging purposes.
func AddLogRequestMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := log.GetLoggerFromContext(r.Context()).With("request", r.URL.Path)

		next.ServeHTTP(w, r)
		l.Info("request processed")
	}
}

// AddLogCorrelationIDMiddleware retrieves the correlation ID from the request's header
// X-Correlation-ID and adds it to the logs. If the header is not present, it generates
// a new Correlation-ID.
func AddLogCorrelationIDMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get logger from context
		l := log.GetLoggerFromContext(r.Context())

		// get Correlation ID from the request.
		// If not present, generate a new one.
		cid := cmp.Or(r.Header.Get("X-Correlation-ID"), string(uuid.NewUUID()))
		l = l.With("correlation-id", cid)

		// run the next handler
		ctx := log.SetLoggerIntoContext(r.Context(), l)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AddInjectLoggerMiddleware injects the provided logger in each request context.
func AddInjectLoggerMiddleware(l slog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := log.SetLoggerIntoContext(r.Context(), &l)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
