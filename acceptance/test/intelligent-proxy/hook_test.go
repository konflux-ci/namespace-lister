package acceptance

import (
	"cmp"
	"context"
	"fmt"
	"os"

	"github.com/cucumber/godog"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectHooks(ctx *godog.ScenarioContext) {
	ctx.Before(injectRun)
	ctx.Before(prepareTestRunServiceAccount)
	ctx.Before(injectBuildUserClient)
}

func injectRun(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return tcontext.WithRunId(ctx, sc.Id), nil
}

func prepareTestRunServiceAccount(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
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

	cfg.Host = cmp.Or(os.Getenv("KONFLUX_ADDRESS"), "https://localhost:10443")

	return arest.BuildClient(cfg)
}

func buildUserClientWithTokenReview(ctx context.Context) (client.Client, error) {
	// build client with bearer token
	cfg, err := arest.NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}

	cfg.BearerToken = tcontext.User(ctx).Token
	cfg.Host = cmp.Or(os.Getenv("KONFLUX_ADDRESS"), "https://localhost:10443")

	return arest.BuildClient(cfg)
}
