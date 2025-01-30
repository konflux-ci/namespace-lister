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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
			l.Error("Unable to load TLS certificates", "error", err)
			return nil, fmt.Errorf("Unable to load TLS certificates: %w", err)
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
	flag.BoolVar(&enableTLS, "enable-tls", true, "Toggle TLS enablement.")
	flag.StringVar(&tlsCertificatePath, "cert-path", "", "Path to TLS certificate store.")
	flag.StringVar(&tlsCertificateKeyPath, "key-path", "", "Path to TLS private key.")
	flag.Parse()

	// get config
	cfg := ctrl.GetConfigOrDie()

	// build the request authenticator
	ar, err := NewAuthenticator(AuthenticatorOptions{
		Config: cfg,
		Header: GetUsernameHeaderFromEnv(),
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
	accessCache, err := buildAndStartAccessCache(ctx, resourceCache)
	if err != nil {
		return err
	}

	// create the namespace lister
	nsl := NewNamespaceListerForSubject(accessCache)

	// build http server
	l.Info("building server")
	s := NewServer(l, ar, nsl)

	// configure TLS
	s.WithTLS(enableTLS).
		WithTLSOpts(loadTLSCert(l, tlsCertificatePath, tlsCertificateKeyPath))

	// start the server
	return s.Start(ctx)
}
