package suite

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
)

func InjectErrorSteps(ctx *godog.ScenarioContext) {
	ctx.Then(`^the user request returns unauthorized$`, theUserRequestReturnsUnauthorized)
	ctx.Then(`^the user gets an empty namespace list$`, theUserGetsAnEmptyNamespaceList)
}

func theUserRequestReturnsUnauthorized(ctx context.Context) (context.Context, error) {
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	nn := corev1.NamespaceList{}
	if err := cli.List(ctx, &nn); !errors.IsUnauthorized(err) {
		if err == nil {
			return ctx, fmt.Errorf("expected unauthorized error, but request succeeded with %d items", len(nn.Items))
		}
		return ctx, fmt.Errorf("expected unauthorized error, got: %v", err)
	}

	return ctx, nil
}

func theUserGetsAnEmptyNamespaceList(ctx context.Context) (context.Context, error) {
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	nn := corev1.NamespaceList{}
	if err := cli.List(ctx, &nn); err != nil {
		return ctx, fmt.Errorf("expected successful response, got error: %v", err)
	}

	if len(nn.Items) != 0 {
		return ctx, fmt.Errorf("expected empty namespace list, got %d items", len(nn.Items))
	}

	return ctx, nil
}
