package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	l := buildLogger()
	if err := run(l); err != nil {
		l.Error("error running the server", "error", err)
		os.Exit(1)
	}
}

func loadTLSCert(l *slog.Logger, certPath, keyPath string) func(*tls.Config) {
	getCertificate := func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			l.Error("unable to load TLS certificates", "error", err)
			return nil, fmt.Errorf("unable to load TLS certificates: %w", err)
		}

		return &cert, err
	}

	return func(config *tls.Config) {
		config.GetCertificate = getCertificate
	}
}

func run(l *slog.Logger) error {
	log.SetLogger(logr.FromSlogHandler(l.Handler()))

	var enableTLS bool
	var tlsCertificatePath string
	var tlsCertificateKeyPath string
	var enableMetrics bool
	var metricsAddress string
	flag.BoolVar(&enableTLS, "enable-tls", true, "Toggle TLS enablement.")
	flag.StringVar(&tlsCertificatePath, "cert-path", "", "Path to TLS certificate store.")
	flag.StringVar(&tlsCertificateKeyPath, "key-path", "", "Path to TLS private key.")
	flag.BoolVar(&enableMetrics, "enable-metrics", true, "Enable metrics server.")
	flag.StringVar(&metricsAddress, "metrics-address", ":9100", "metrics server address.")
	flag.Parse()

	reg := metrics.Registry
	InitRegistry(metrics.Registry)

	// get config
	cfg := ctrl.GetConfigOrDie()

	// build the request authenticator
	ar, err := NewAuthenticator(AuthenticatorOptions{
		Config:         cfg,
		UsernameHeader: GetUsernameHeaderFromEnv(),
		GroupsHeader:   GetGroupsHeaderFromEnv(),
	})
	if err != nil {
		return err
	}

	// setup context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ctx = setLoggerIntoContext(ctx, l)

	// create resource cache
	l.Info("creating resource cache")
	cacheCfg, err := NewResourceCacheConfigFromEnv(cfg)
	if err != nil {
		return err
	}
	resourceCache, err := BuildAndStartResourceCache(ctx, cacheCfg)
	if err != nil {
		return err
	}

	// create access cache
	l.Info("creating access cache")
	accessCache, err := buildAndStartSynchronizedAccessCache(ctx, resourceCache, reg)
	if err != nil {
		return err
	}

	// create the namespace lister
	nsl := NewSubjectNamespaceLister(accessCache)

	// build and start http metrics server
	if enableMetrics {
		l.Info("building metrics server")
		httpClient, err := rest.HTTPClientFor(cacheCfg.restConfig)
		if err != nil {
			l.Error("unable to build http client for metrics server", "error", err)
			return err
		}
		ms, err := server.NewServer(server.Options{
			SecureServing: enableTLS,
			TLSOpts: []func(*tls.Config){
				loadTLSCert(l, tlsCertificatePath, tlsCertificateKeyPath),
			},
			BindAddress:    metricsAddress,
			FilterProvider: filters.WithAuthenticationAndAuthorization,
		}, cacheCfg.restConfig, httpClient)
		if err != nil {
			l.Error("unable to build metrics server", "error", err)
			return err
		}

		l.Info("starting metrics server in background")
		go func() {
			defer cancel()

			if err := ms.Start(ctx); err != nil {
				l.Error("error running metrics server: invalidating context, application will be terminated", "error", err)
				return
			}
			l.Info("metrics server terminated as context has been invalidated")
		}()
	} else {
		l.Info("metrics server disabled via flags")
	}

	// build http api server
	l.Info("building api server")
	s := NewAPIServer(l, ar, nsl, reg).
		WithTLS(enableTLS).
		WithTLSOpts(loadTLSCert(l, tlsCertificatePath, tlsCertificateKeyPath))

	// start the server
	return s.Start(ctx)
}
