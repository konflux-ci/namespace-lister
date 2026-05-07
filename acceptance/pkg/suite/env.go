package suite

import (
	"cmp"
	"context"
	"crypto/tls"
	"os"

	"github.com/cucumber/godog"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
)

const EnvKonfluxAddress string = "KONFLUX_ADDRESS"

func EnvKonfluxAddressOrDefault(address string) string {
	return cmp.Or(os.Getenv(EnvKonfluxAddress), address)
}

func BuildTLSConfig() *tls.Config {
	if os.Getenv("E2E_USE_INSECURE_TLS") == "true" {
		return &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &tls.Config{}
}

func InjectServiceAddresses(defaultAddress string) func(context.Context, *godog.Scenario) (context.Context, error) {
	return func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		address := EnvKonfluxAddressOrDefault(defaultAddress)
		ctx = tcontext.WithNamespaceListerAddress(ctx, address)
		ctx = tcontext.WithTLSConfig(ctx, BuildTLSConfig())
		return ctx, nil
	}
}
