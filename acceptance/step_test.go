package acceptance

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/cucumber/godog"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	"github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	//read
	ctx.Step(`^user has access to a namespace$`,
		func(ctx context.Context) (context.Context, error) { return UserHasAccessToNNamespaces(ctx, 1) })
	ctx.Step(`^the user can retrieve the namespace$`, TheUserCanRetrieveTheNamespace)

	// list
	ctx.Step(`^user has access to "([^"]*)" namespaces$`, UserHasAccessToNNamespaces)
	ctx.Step(`^the user can retrieve only the namespaces they have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
}

func UserHasAccessToNNamespaces(ctx context.Context, number int) (context.Context, error) {
	run := tcontext.RunId(ctx)

	cli, err := rest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	// create serviceaccount
	if err := cli.Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user",
			Namespace: "default",
		},
	}); err != nil && !errors.IsAlreadyExists(err) {
		return ctx, err
	}

	// create namespaces
	nn := []corev1.Namespace{}
	for i := range number {
		n := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("run-%s-%d", run, i),
				Labels: map[string]string{
					"namespace-lister/scope":    "acceptance-tests",
					"namespace-lister/test-run": run,
				},
			},
		}
		if err := cli.Create(ctx, &n); err != nil {
			return ctx, err
		}

		if err := cli.Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("run-%s-%d", run, i),
				Namespace: fmt.Sprintf("run-%s-%d", run, i),
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     "namespace-get",
				APIGroup: rbacv1.GroupName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:     "User",
					APIGroup: rbacv1.GroupName,
					Name:     "user",
				},
			},
		}); err != nil {
			return ctx, err
		}

		nn = append(nn, n)
	}

	return tcontext.WithNamespaces(ctx, nn), nil
}

func TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo(ctx context.Context) (context.Context, error) {
	cli, err := buildUserClient()
	if err != nil {
		return ctx, err
	}

	ann := corev1.NamespaceList{}
	if err := cli.List(ctx, &ann); err != nil {
		return ctx, err
	}

	enn := tcontext.Namespaces(ctx)
	if expected, actual := len(enn), len(ann.Items); expected != actual {
		return ctx, fmt.Errorf("expected %d namespaces, actual %d", expected, actual)
	}

	for _, en := range enn {
		if !slices.ContainsFunc(ann.Items, func(an corev1.Namespace) bool {
			return en.Name == an.Name
		}) {
			return ctx, fmt.Errorf("expected namespace %s not found in actual namespace list: %v", en.Name, ann.Items)
		}
	}

	return ctx, nil
}

func TheUserCanRetrieveTheNamespace(ctx context.Context) (context.Context, error) {
	run := tcontext.RunId(ctx)

	cli, err := buildUserClient()
	if err != nil {
		return ctx, err
	}

	n := corev1.Namespace{}
	k := types.NamespacedName{Name: fmt.Sprintf("run-%s-0", run)}
	if err := cli.Get(ctx, k, &n); err != nil {
		return ctx, err
	}

	enn := tcontext.Namespaces(ctx)
	if expected, actual := 1, len(enn); expected != actual {
		return ctx, fmt.Errorf("expected %d namespaces, actual %d: %v", expected, actual, enn)
	}

	if expected, actual := n.Name, enn[0].Name; actual != expected {
		return ctx, fmt.Errorf("expected namespace %s, actual %s", expected, actual)
	}

	return ctx, nil
}

func buildUserClient() (client.Client, error) {
	// build impersonating client
	cfg, err := rest.NewDefaultClientConfig()
	if err != nil {
		return nil, err
	}
	cfg.Impersonate.UserName = "user"
	cfg.Host = cmp.Or(os.Getenv("KONFLUX_ADDRESS"), "https://localhost:10443")
	return rest.BuildClient(cfg)
}
