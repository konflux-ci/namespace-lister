package suite

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	corev1 "k8s.io/api/core/v1"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
)

func InjectResponseSteps(ctx *godog.ScenarioContext) {
	ctx.Then(`^the response is a valid NamespaceList$`, theResponseIsAValidNamespaceList)
}

func theResponseIsAValidNamespaceList(ctx context.Context) (context.Context, error) {
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, fmt.Errorf("error building user client: %w", err)
	}

	nn := corev1.NamespaceList{}
	if err := cli.List(ctx, &nn); err != nil {
		return ctx, fmt.Errorf("error listing namespaces: %w", err)
	}

	if nn.Kind == "" && nn.APIVersion == "" {
		nn.Kind = "NamespaceList"
		nn.APIVersion = "v1"
	}

	for i, item := range nn.Items {
		if item.Name == "" {
			return ctx, fmt.Errorf("item[%d] has empty metadata.name", i)
		}
	}

	return ctx, nil
}
