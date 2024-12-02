package suite

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectBaseHooks(ctx *godog.ScenarioContext) {
	ctx.Before(InjectRunId)
	ctx.Before(PrepareTestRunServiceAccount)
}

func InjectRunId(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return tcontext.WithRunId(ctx, sc.Id), nil
}

func PrepareTestRunServiceAccount(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	cli, err := rest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	// create serviceaccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("user-%s", sc.Id),
			Namespace: "acceptance-tests",
			Labels: map[string]string{
				"namespace-lister/scope":    "acceptance-tests",
				"namespace-lister/test-run": sc.Id,
			},
		},
	}
	if err := cli.Create(ctx, sa); err != nil && !errors.IsAlreadyExists(err) {
		return ctx, err
	}

	// create a token for authenticating as the service account
	tkn := &authenticationv1.TokenRequest{}
	if err := cli.SubResource("token").Create(ctx, sa, tkn); err != nil {
		return ctx, err
	}

	// store auth info in context for future use
	ui := tcontext.UserInfoFromServiceAccount(*sa, tkn)
	return tcontext.WithUser(ctx, ui), nil
}
