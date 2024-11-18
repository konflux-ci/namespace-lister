package acceptance

import (
	"context"

	"github.com/cucumber/godog"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
)

func InjectHooks(ctx *godog.ScenarioContext) {
	ctx.Before(injectRun)
}

func injectRun(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return tcontext.WithRunId(ctx, sc.Id), nil
}
