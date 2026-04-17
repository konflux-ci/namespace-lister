package acceptance

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	suite.InjectSteps(ctx)
	suite.InjectCacheSteps(ctx)
	suite.InjectErrorSteps(ctx)
	suite.InjectHealthSteps(ctx)
	suite.InjectMetricsSteps(ctx)
	suite.InjectResponseSteps(ctx)

	ctx.Given("^User is not authenticated$", dumbProxyUserIsNotAuthenticated)
}

func dumbProxyUserIsNotAuthenticated(ctx context.Context) (context.Context, error) {
	runId := tcontext.RunId(ctx)
	username := fmt.Sprintf("user-%s", runId)
	user := tcontext.UserInfoFromUsername(username)
	ctx = tcontext.WithUser(ctx, user)

	ctx = tcontext.WithBuildUserClientFunc(ctx, func(ctx context.Context) (client.Client, error) {
		cfg, err := arest.NewDefaultClientConfig()
		if err != nil {
			return nil, err
		}
		cfg.BearerToken = "invalid-token"
		cfg.Host = suite.EnvKonfluxAddressOrDefault(defaultTestAddress)
		return arest.BuildClient(cfg)
	})
	return ctx, nil
}
