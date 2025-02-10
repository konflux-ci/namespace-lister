package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer is an HTTP Server that serves metrics for the APIServer
type MetricsServer struct {
	*http.Server
	useTLS  bool
	tlsOpts []func(*tls.Config)
}

// NewMetricsServer builds a new MetricsServer
func NewMetricsServer(address string, registry *prometheus.Registry) *MetricsServer {
	// configure the server
	h := http.NewServeMux()

	h.Handle("/metrics",
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			Registry: registry,
		}))

	return &MetricsServer{
		Server: &http.Server{
			Addr:              address,
			Handler:           h,
			ReadHeaderTimeout: 3 * time.Second,
		},
	}
}

// WithTLS enables the TLS Support
func (s *MetricsServer) WithTLS(enableTLS bool) *MetricsServer {
	s.useTLS = enableTLS
	return s
}

// WithTLSOpts allows to configure the TLS support
func (s *MetricsServer) WithTLSOpts(tlsOpts ...func(*tls.Config)) *MetricsServer {
	s.tlsOpts = tlsOpts
	return s
}

// Start starts the MetricsServer blocking the current routine.
// It monitors in a separate routine shutdown requests by waiting
// for the provided context to be invalidated.
func (s *MetricsServer) Start(ctx context.Context) error {
	// HTTP Server graceful shutdown
	go func() {
		<-ctx.Done()

		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		//nolint:contextcheck
		if err := s.Shutdown(sctx); err != nil {
			getLoggerFromContext(ctx).Error("error gracefully shutting down the metrics HTTP server", "error", err)
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
