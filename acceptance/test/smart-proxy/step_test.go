package acceptance

import (
	"github.com/cucumber/godog"

	"github.com/konflux-ci/namespace-lister/acceptance/pkg/suite"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	suite.InjectSteps(ctx)
}
