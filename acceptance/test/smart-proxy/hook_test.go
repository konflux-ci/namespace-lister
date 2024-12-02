package acceptance

import (
	"context"

	"github.com/cucumber/godog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"
)

const defaultTestAddress string = "https://localhost:11443"

func InjectHooks(ctx *godog.ScenarioContext) {
	suite.InjectBaseHooks(ctx)

	ctx.Before(injectBuildUserClient)
}

func injectBuildUserClient(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return tcontext.WithBuildUserClientFunc(ctx, buildUserClientForAuthProxy), nil
}

func buildUserClientForAuthProxy(ctx context.Context) (client.Client, error) {
	// build impersonating client
	cfg, err := arest.NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}

	user := tcontext.User(ctx)
	cfg.Impersonate.UserName = user.Name

	cfg.Host = suite.EnvKonfluxAddressOrDefault(defaultTestAddress)
	return arest.BuildClient(cfg)
}
