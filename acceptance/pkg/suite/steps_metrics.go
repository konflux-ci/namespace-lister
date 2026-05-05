package suite

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectMetricsSteps(ctx *godog.ScenarioContext) {
	ctx.Then(`^the metrics service is available in the cluster$`, metricsServiceIsAvailable)
}

func metricsServiceIsAvailable(ctx context.Context) (context.Context, error) {
	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	svc := &corev1.Service{}
	if err := cli.Get(ctx, types.NamespacedName{
		Name:      "namespace-lister-metrics",
		Namespace: "namespace-lister",
	}, svc); err != nil {
		return ctx, fmt.Errorf("metrics service not found: %w", err)
	}

	found := false
	for _, p := range svc.Spec.Ports {
		if p.Port == 9100 {
			found = true
			break
		}
	}
	if !found {
		return ctx, fmt.Errorf("metrics service does not expose port 9100")
	}

	return ctx, nil
}
