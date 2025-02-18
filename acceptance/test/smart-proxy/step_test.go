package acceptance

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	suite.InjectSteps(ctx)

	ctx.Given("^User is not authenticated$", userIsNotAuthenticated)
}

func userIsNotAuthenticated(ctx context.Context) (context.Context, error) {
	runId := tcontext.RunId(ctx)
	username := fmt.Sprintf("user-%s", runId)
	userId := tcontext.UserInfoFromUsername(username)
	ctx = tcontext.WithUser(ctx, userId)

	// override BuildUserClientFunc to set up access for unauthenticated user
	ctx = tcontext.WithBuildUserClientFunc(ctx, buildUnauthenticatedUserClientForAuthProxy)
	return ctx, nil
}
