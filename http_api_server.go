package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apiserver/pkg/authentication/authenticator"
)

const (
	patternGetNamespaces string = "GET /api/v1/namespaces"
	patternHealthz       string = "GET /healthz"
	patternReadyz        string = "GET /readyz"
)

// APIServer is an HTTP server that serves the List Namespace endpoint
type APIServer struct {
	*http.Server
	useTLS  bool
	tlsOpts []func(*tls.Config)
}

func healthz(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(http.StatusOK)
}

// NewAPIServer builds a new APIServer
func NewAPIServer(l *slog.Logger, ar authenticator.Request, lister NamespaceLister, reg prometheus.Registerer) *APIServer {
	// configure the server
	h := http.NewServeMux()
	h.Handle(patternGetNamespaces,
		addMetricsMiddleware(reg,
			addInjectLoggerMiddleware(*l,
				addLogRequestMiddleware(
					addAuthnMiddleware(ar,
						NewListNamespacesHandler(lister))))))

	h.HandleFunc(patternHealthz, healthz)
	h.HandleFunc(patternReadyz, healthz)

	return &APIServer{
		Server: &http.Server{
			Addr:              getAddress(),
			Handler:           h,
			ReadHeaderTimeout: 3 * time.Second,
		},
	}
}

// WithTLS enables the TLS Support
func (s *APIServer) WithTLS(enableTLS bool) *APIServer {
	s.useTLS = enableTLS
	return s
}

// WithTLSOpts allows to configure the TLS support
func (s *APIServer) WithTLSOpts(tlsOpts ...func(*tls.Config)) *APIServer {
	s.tlsOpts = tlsOpts
	return s
}

// Start starts the APIServer blocking the current routine.
// It monitors in a separate routine shutdown requests by waiting
// for the provided context to be invalidated.
func (s *APIServer) Start(ctx context.Context) error {
	// HTTP Server graceful shutdown
	go func() {
		<-ctx.Done()

		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		//nolint:contextcheck
		if err := s.Shutdown(sctx); err != nil {
			getLoggerFromContext(ctx).Error("error gracefully shutting down the HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	// setup and serve over TLS if configured
	if s.useTLS {
		s.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		for _, fun := range s.tlsOpts {
			fun(s.TLSConfig)
		}
		return s.ListenAndServeTLS("", "")
	}

	// start server
	return s.ListenAndServe()
}
