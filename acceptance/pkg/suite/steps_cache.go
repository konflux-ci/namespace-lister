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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
	arest "github.com/konflux-ci/namespace-lister/acceptance/pkg/rest"
)

func InjectCacheSteps(ctx *godog.ScenarioContext) {
	ctx.Given(`^a namespace "([^"]*)" exists without access for the current user$`, aNamespaceExistsWithoutAccess)
	ctx.When(`^a new namespace "([^"]*)" is created with access for the current user$`, aNewNamespaceIsCreatedWithAccess)
	ctx.When(`^the namespace "([^"]*)" is deleted$`, theNamespaceIsDeleted)
	ctx.When(`^a RoleBinding granting access is added in namespace "([^"]*)"$`, aRoleBindingIsAddedInNamespace)
	ctx.When(`^the RoleBinding is removed from namespace "([^"]*)"$`, theRoleBindingIsRemovedFromNamespace)
	ctx.When(`^(\d+) namespaces with access are created and (\d+) existing namespaces are deleted$`, bulkNamespaceChanges)

	ctx.Then(`^the user can see namespace "([^"]*)" in the list$`, theUserCanSeeNamespaceInTheList)
	ctx.Then(`^the user cannot see namespace "([^"]*)" in the list$`, theUserCannotSeeNamespaceInTheList)
	ctx.Then(`^the user sees exactly (\d+) namespaces$`, theUserSeesExactlyNNamespaces)
}

func aNamespaceExistsWithoutAccess(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fullName,
			Labels: map[string]string{
				"konflux.ci/type":           "user",
				"namespace-lister/scope":    "acceptance-tests",
				"namespace-lister/test-run": run,
			},
		},
	}
	if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return ctx, err
	}

	return ctx, nil
}

func aNewNamespaceIsCreatedWithAccess(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	user := tcontext.User(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fullName,
			Labels: map[string]string{
				"konflux.ci/type":           "user",
				"namespace-lister/scope":    "acceptance-tests",
				"namespace-lister/test-run": run,
			},
		},
	}
	if err := cli.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return ctx, err
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("run-%s-%s-access", run, nsName),
			Namespace: fullName,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "namespace-get",
			APIGroup: rbacv1.GroupName,
		},
		Subjects: []rbacv1.Subject{user.AsSubject()},
	}
	if err := cli.Create(ctx, rb); err != nil && !errors.IsAlreadyExists(err) {
		return ctx, err
	}

	nn := tcontext.Namespaces(ctx)
	nn = append(nn, *ns)
	return tcontext.WithNamespaces(ctx, nn), nil
}

func theNamespaceIsDeleted(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fullName}}
	if err := cli.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
		return ctx, err
	}

	nn := tcontext.Namespaces(ctx)
	nn = slices.DeleteFunc(nn, func(n corev1.Namespace) bool {
		return n.Name == fullName
	})
	return tcontext.WithNamespaces(ctx, nn), nil
}

func aRoleBindingIsAddedInNamespace(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	user := tcontext.User(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("run-%s-%s-access", run, nsName),
			Namespace: fullName,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "namespace-get",
			APIGroup: rbacv1.GroupName,
		},
		Subjects: []rbacv1.Subject{user.AsSubject()},
	}
	if err := cli.Create(ctx, rb); err != nil && !errors.IsAlreadyExists(err) {
		return ctx, err
	}

	// track the namespace as expected
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fullName}}
	nn := tcontext.Namespaces(ctx)
	if !slices.ContainsFunc(nn, func(n corev1.Namespace) bool { return n.Name == fullName }) {
		nn = append(nn, ns)
	}
	return tcontext.WithNamespaces(ctx, nn), nil
}

func theRoleBindingIsRemovedFromNamespace(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("run-%s-%s-access", run, nsName),
			Namespace: fullName,
		},
	}
	if err := cli.Delete(ctx, rb); err != nil && !errors.IsNotFound(err) {
		return ctx, err
	}

	nn := tcontext.Namespaces(ctx)
	nn = slices.DeleteFunc(nn, func(n corev1.Namespace) bool {
		return n.Name == fullName
	})
	return tcontext.WithNamespaces(ctx, nn), nil
}

func bulkNamespaceChanges(ctx context.Context, toCreate, toDelete int) (context.Context, error) {
	run := tcontext.RunId(ctx)
	user := tcontext.User(ctx)

	cli, err := arest.BuildDefaultHostClient()
	if err != nil {
		return ctx, err
	}

	// delete existing namespaces
	nn := tcontext.Namespaces(ctx)
	deleted := 0
	for i := range nn {
		if deleted >= toDelete {
			break
		}
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nn[i].Name}}
		if err := cli.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
			return ctx, err
		}
		deleted++
	}
	nn = nn[deleted:]

	// create new namespaces
	tn := time.Now().Unix()
	for i := range toCreate {
		name := fmt.Sprintf("run-%s-bulk-%d-%d", run, tn, i)
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"konflux.ci/type":           "user",
					"namespace-lister/scope":    "acceptance-tests",
					"namespace-lister/test-run": run,
				},
			},
		}
		if err := cli.Create(ctx, ns); err != nil {
			return ctx, err
		}

		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("run-%s-bulk-%d-%d-access", run, tn, i),
				Namespace: name,
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "ClusterRole",
				Name:     "namespace-get",
				APIGroup: rbacv1.GroupName,
			},
			Subjects: []rbacv1.Subject{user.AsSubject()},
		}
		if err := cli.Create(ctx, rb); err != nil {
			return ctx, err
		}

		nn = append(nn, *ns)
	}

	return tcontext.WithNamespaces(ctx, nn), nil
}

func theUserCanSeeNamespaceInTheList(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	return ctx, wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		ann := corev1.NamespaceList{}
		if err := cli.List(ctx, &ann); err != nil {
			log.Printf("error listing namespaces: %v", err)
			return false, nil
		}

		return slices.ContainsFunc(ann.Items, func(n corev1.Namespace) bool {
			return n.Name == fullName
		}), nil
	})
}

func theUserCannotSeeNamespaceInTheList(ctx context.Context, nsName string) (context.Context, error) {
	run := tcontext.RunId(ctx)
	fullName := fmt.Sprintf("run-%s-%s", run, nsName)

	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	return ctx, wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		ann := corev1.NamespaceList{}
		if err := cli.List(ctx, &ann); err != nil {
			log.Printf("error listing namespaces: %v", err)
			return false, nil
		}

		return !slices.ContainsFunc(ann.Items, func(n corev1.Namespace) bool {
			return n.Name == fullName
		}), nil
	})
}

func theUserSeesExactlyNNamespaces(ctx context.Context, expected int) (context.Context, error) {
	cli, err := tcontext.InvokeBuildUserClientFunc(ctx)
	if err != nil {
		return ctx, err
	}

	return ctx, wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		ann := corev1.NamespaceList{}
		if err := cli.List(ctx, &ann); err != nil {
			log.Printf("error listing namespaces: %v", err)
			return false, nil
		}

		nn := tcontext.Namespaces(ctx)
		if len(ann.Items) != expected {
			names := make([]string, len(ann.Items))
			for i, n := range ann.Items {
				names[i] = n.Name
			}
			log.Printf("expected %d namespaces, got %d: %v", expected, len(ann.Items), names)
			return false, nil
		}

		for _, en := range nn {
			if !slices.ContainsFunc(ann.Items, func(an corev1.Namespace) bool {
				return en.Name == an.Name
			}) {
				log.Printf("expected namespace %s not found in actual namespace list", en.Name)
				return false, nil
			}
		}

		return true, nil
	})
}
