package suite

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectSteps(ctx *godog.ScenarioContext) {
	ctx.Given(`^ServiceAccount has access to "([^"]*)" namespaces$`, UserInfoHasAccessToNNamespaces)
	ctx.Given(`^User has access to "([^"]*)" namespaces$`, UserHasAccessToNNamespaces)
	ctx.Given(`^Group "([^"]*)" has access to "([^"]*)" namespaces$`, GroupHasAccessToNNamespaces)
	ctx.Given(`^User is part of group "([^"]*)"$`, UserIsPartOfGroup)
	ctx.Given(`^the ServiceAccount has Cluster-scoped get permission on namespaces$`, UserInfoHasClusterScopedGetPermissionOnNamespaces)
	ctx.Given(`^(\d+) tenant namespaces exist$`, NTenantNamespacesExist)

	ctx.Then(`^the ServiceAccount can retrieve the namespaces they and their groups have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
	ctx.Then(`^the User can retrieve the namespaces they and their groups have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
	ctx.Then(`^the ServiceAccount can retrieve only the namespaces they have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
	ctx.Then(`^the User can retrieve only the namespaces they have access to$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
	ctx.Then(`^the ServiceAccount retrieves no namespaces$`, TheUserCanRetrieveOnlyTheNamespacesTheyHaveAccessTo)
	ctx.Then(`^the User request is rejected with unauthorized error$`, userRequestIsRejectedWithUnauthorizerError)
}

func userRequestIsRejectedWithUnauthorizerError(ctx context.Context) (context.Context, error) {
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	nn := corev1.NamespaceList{}
	if err := cli.List(ctx, &nn); !errors.IsUnauthorized(err) {
		return ctx, err
	}

	return ctx, nil
}

func NTenantNamespacesExist(ctx context.Context, limit int) (context.Context, error) {
	run := tcontext.RunId(ctx)
	tn := time.Now().Unix()

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	for i := range limit {
		n := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("run-%s-%d-%d", run, tn, i),
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
	}

	return ctx, nil
}

func UserInfoHasClusterScopedGetPermissionOnNamespaces(ctx context.Context) (context.Context, error) {
	user := tcontext.User(ctx)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	// ensure the cluster role get-namespace exists
	cr := &rbacv1.ClusterRole{}
	if err := cli.Get(ctx, types.NamespacedName{Name: "namespace-get"}, cr); err != nil {
		return ctx, err
	}

	cli.Create(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("test-%s-get-namespaces", user.Name),
		},
		RoleRef: rbacv1.RoleRef{
			Name:     cr.Name,
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
		},
		Subjects: []rbacv1.Subject{user.AsSubject()},
	})

	return ctx, nil
}

func UserIsPartOfGroup(ctx context.Context, group string) (context.Context, error) {
	user := tcontext.User(ctx)
	user.Groups = append(user.Groups, group)
	return tcontext.WithUser(ctx, user), nil
}

func GroupHasAccessToNNamespaces(ctx context.Context, group string, number int) (context.Context, error) {
	sub := rbacv1.Subject{
		Kind:     rbacv1.GroupKind,
		APIGroup: rbacv1.GroupName,
		Name:     group,
	}
	return subjectHasAccessToNNamespaces(ctx, sub, number)
}

func UserHasAccessToNNamespaces(ctx context.Context, number int) (context.Context, error) {
	runId := tcontext.RunId(ctx)
	username := fmt.Sprintf("user-%s", runId)
	user := tcontext.UserInfoFromUsername(username)
	ctx = tcontext.WithUser(ctx, user)
	return UserInfoHasAccessToNNamespaces(ctx, number)
}

func UserInfoHasAccessToNNamespaces(ctx context.Context, number int) (context.Context, error) {
	user := tcontext.User(ctx)
	return subjectHasAccessToNNamespaces(ctx, user.AsSubject(), number)
}

func subjectHasAccessToNNamespaces(ctx context.Context, subject rbacv1.Subject, number int) (context.Context, error) {
	run := tcontext.RunId(ctx)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	// create namespaces
	nn := tcontext.Namespaces(ctx)
	for i := range number {
		n := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("run-%s-%s-%d", run, strings.ReplaceAll(subject.Name, ":", "-"), i),
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
				Name:      fmt.Sprintf("run-%s-%s-%d", run, strings.ReplaceAll(subject.Name, ":", "-"), i),
				Namespace: fmt.Sprintf("run-%s-%s-%d", run, strings.ReplaceAll(subject.Name, ":", "-"), i),
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     "namespace-get",
				APIGroup: rbacv1.GroupName,
			},
			Subjects: []rbacv1.Subject{subject},
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
			ad := make([]string, len(ann.Items))
			for _, n := range ann.Items {
				ad = append(ad, n.Name)
			}
			log.Printf("expected %d namespaces, actual %d: %v", expected, actual, ad)
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
