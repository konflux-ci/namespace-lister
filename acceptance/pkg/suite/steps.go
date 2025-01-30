package suite

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/cucumber/godog"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	// read
	ctx.Step(`^ServiceAccount has access to a namespace$`,
		func(ctx context.Context) (context.Context, error) { return UserInfoHasAccessToNNamespaces(ctx, 1) })
	ctx.Step(`^User has access to a namespace$`,
		func(ctx context.Context) (context.Context, error) { return UserHasAccessToNNamespaces(ctx, 1) })
	ctx.Step(`^the ServiceAccount can retrieve the namespace$`, TheUserCanRetrieveTheNamespace)

	// list
	ctx.Step(`^ServiceAccount has access to "([^"]*)" namespaces$`, UserInfoHasAccessToNNamespaces)
	ctx.Step(`^User has access to "([^"]*)" namespaces$`, UserHasAccessToNNamespaces)
	ctx.Step(`^the ServiceAccount can retrieve only the namespaces they have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
	ctx.Step(`^the User can retrieve only the namespaces they have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
}

func UserHasAccessToNNamespaces(ctx context.Context, number int) (context.Context, error) {
	runId := tcontext.RunId(ctx)
	username := fmt.Sprintf("user-%s", runId)
	userId := tcontext.UserInfoFromUsername(username)
	ctx = tcontext.WithUser(ctx, userId)
	return UserInfoHasAccessToNNamespaces(ctx, number)
}

func UserInfoHasAccessToNNamespaces(ctx context.Context, number int) (context.Context, error) {
	run := tcontext.RunId(ctx)
	user := tcontext.User(ctx)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	// create namespaces
	nn := []corev1.Namespace{}
	for i := range number {
		n := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("run-%s-%d", run, i),
				Labels: map[string]string{
					"konflux.ci/type":           "user",
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
			Subjects: []rbacv1.Subject{user.AsSubject()},
		}); err != nil {
			return ctx, err
		}

		nn = append(nn, n)
	}

	return tcontext.WithNamespaces(ctx, nn), nil
}

func TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo(ctx context.Context) (context.Context, error) {
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	return ctx, wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		ann := corev1.NamespaceList{}
		if err := cli.List(ctx, &ann); err != nil {
			log.Printf("error listing namespaces: %v", err)
			return false, nil
		}

		enn := tcontext.Namespaces(ctx)
		if expected, actual := len(enn), len(ann.Items); expected != actual {
			log.Printf("expected %d namespaces, actual %d", expected, actual)
			return false, nil
		}

		for _, en := range enn {
			if !slices.ContainsFunc(ann.Items, func(an corev1.Namespace) bool {
				return en.Name == an.Name
			}) {
				log.Printf("expected namespace %s not found in actual namespace list: %v", en.Name, ann.Items)
				return false, nil
			}
		}
		return true, nil
	})
}

func TheUserCanRetrieveTheNamespace(ctx context.Context) (context.Context, error) {
	run := tcontext.RunId(ctx)
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	return ctx, wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		n := corev1.Namespace{}
		k := types.NamespacedName{Name: fmt.Sprintf("run-%s-0", run)}
		if err := cli.Get(ctx, k, &n); err != nil {
			log.Printf("error getting namespace %v: %v", k, err)
			return false, nil
		}

		enn := tcontext.Namespaces(ctx)
		if expected, actual := 1, len(enn); expected != actual {
			log.Printf("expected %d namespaces, actual %d: %v", expected, actual, enn)
			return false, nil
		}

		if expected, actual := n.Name, enn[0].Name; actual != expected {
			log.Printf("expected namespace %s, actual %s", expected, actual)
			return false, nil
		}
		return true, nil
	})
}
