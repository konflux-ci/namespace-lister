package acceptance

import (
	"github.com/cucumber/godog"

	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	suite.InjectSteps(ctx)
	suite.InjectHealthSteps(ctx)
	suite.InjectMetricsSteps(ctx)
	suite.InjectCacheSteps(ctx)
}
