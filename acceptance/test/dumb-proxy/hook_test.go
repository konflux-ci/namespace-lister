package acceptance

import (
	"context"
	"crypto/tls"

	"github.com/cucumber/godog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"
)

const defaultTestAddress string = "https://localhost:10443"

func InjectHooks(ctx *godog.ScenarioContext) {
	suite.InjectBaseHooks(ctx)

	ctx.Before(injectBuildUserClient)
	ctx.Before(injectServiceAddresses)
}

func injectBuildUserClient(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return tcontext.WithBuildUserClientFunc(ctx, buildUserClientWithTokenReview), nil
}

func injectServiceAddresses(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	address := suite.EnvKonfluxAddressOrDefault(defaultTestAddress)
	ctx = tcontext.WithNamespaceListerAddress(ctx, address)
	ctx = tcontext.WithMetricsAddress(ctx, address)
	ctx = tcontext.WithTLSConfig(ctx, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	return ctx, nil
}

func buildUserClientWithTokenReview(ctx context.Context) (client.Client, error) {
	cfg, err := arest.NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}

	cfg.BearerToken = tcontext.User(ctx).Token
	cfg.Host = suite.EnvKonfluxAddressOrDefault(defaultTestAddress)

	return arest.BuildClient(cfg)
}
