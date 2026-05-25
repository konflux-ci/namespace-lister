package acceptance

import (
	"context"

	"github.com/cucumber/godog"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"
)

const defaultTestAddress string = "https://localhost:11443"

func InjectHooks(ctx *godog.ScenarioContext) {
	suite.InjectBaseHooks(ctx)

	ctx.Before(injectBuildUserClient)
	ctx.Before(suite.InjectServiceAddresses(defaultTestAddress))
}

func injectBuildUserClient(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return tcontext.WithBuildUserClientFunc(ctx, buildUserClientForAuthProxy), nil
}

func buildUserClientForAuthProxy(ctx context.Context) (client.Client, error) {
	cfg, err := arest.NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}

	user := tcontext.User(ctx)
	cfg.Impersonate.UserName = user.FullName()
	cfg.Impersonate.Groups = user.Groups

	cfg.Host = suite.EnvKonfluxAddressOrDefault(defaultTestAddress)
	return arest.BuildClient(cfg)
}

func buildUnauthenticatedUserClientForAuthProxy(ctx context.Context) (client.Client, error) {
	cfg, err := arest.NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}

	// build a new config with only connectivity settings (no credentials)
	unauthCfg := &rest.Config{
		Host: suite.EnvKonfluxAddressOrDefault(defaultTestAddress),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: cfg.Insecure,
			CAFile:   cfg.CAFile,
			CAData:   cfg.CAData,
		},
	}
	return arest.BuildClient(unauthCfg)
}
