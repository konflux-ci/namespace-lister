package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NamespaceTypeLabelKey       string = "konflux-ci.dev/type"
	NamespaceTypeUserLabelValue string = "user"
)

var _ = Describe("Authorizing requests", Serial, Ordered, func() {
	username := "user"
	var restConfig *rest.Config
	var c client.Client
	var ans []client.Object
	var uns []client.Object
	var cache cache.Cache

	BeforeAll(func(ctx context.Context) {
		var err error

		// prepare scheme
		s := runtime.NewScheme()
		utilruntime.Must(corev1.AddToScheme(s))
		utilruntime.Must(rbacv1.AddToScheme(s))

		// get kubernetes client config
		restConfig = ctrl.GetConfigOrDie()
		restConfig.QPS = 500
		restConfig.Burst = 500

		// build kubernetes client
		c, err = client.New(restConfig, client.Options{Scheme: s})
		utilruntime.Must(err)

		// create resources
		err, ans, uns = createResources(ctx, c, username, 300, 800, 1200)
		utilruntime.Must(err)

		// create cache
		ls, err := labels.Parse(fmt.Sprintf("%s=%s", NamespaceTypeLabelKey, NamespaceTypeUserLabelValue))
		utilruntime.Must(err)
		cacheConfig := cacheConfig{restConfig: restConfig, namespacesLabelSector: ls}
		cache, err = BuildAndStartCache(ctx, &cacheConfig)
		utilruntime.Must(err)
	})

	It("efficiently authorize on a huge environment", Serial, Label("perf"), func(ctx context.Context) {
		// new gomega experiment
		experiment := gmeasure.NewExperiment("Authorizing Request")

		// Register the experiment as a ReportEntry - this will cause Ginkgo's reporter infrastructure
		// to print out the experiment's report and to include the experiment in any generated reports
		AddReportEntry(experiment.Name, experiment)

		// create authorizer, namespacelister, and handler
		authzr := NewAuthorizer(ctx, cache)
		nl := NewNamespaceLister(cache, authzr)
		lnh := NewListNamespacesHandler(nl)

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			rctx := context.WithValue(context.Background(), ContextKeyUserDetails, &authenticator.Response{
				User: &user.DefaultInfo{
					Name: username,
				},
			})
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(rctx)
			w := httptest.NewRecorder()

			// measure http Handler
			experiment.MeasureDuration("http listing", func() {
				lnh.ServeHTTP(w, r)
			})
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			var err error
			var nn *corev1.NamespaceList

			// measure ListNamespaces
			experiment.MeasureDuration("internal listing", func() {
				nn, err = nl.ListNamespaces(ctx, username)
			})

			// check results
			if err != nil {
				panic(err)
			}
			if lnn := len(nn.Items); lnn != len(ans) {
				panic(fmt.Errorf("expecting %d namespaces, received %d", len(ans), lnn))
			}
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			nsName := ans[0].GetName()
			// measure how long it takes to allow a request and store the duration in a "authorization-allow" measurement
			var d authorizer.Decision
			var err error
			r := authorizer.AttributesRecord{
				User:            &user.DefaultInfo{Name: username},
				Verb:            "get",
				Resource:        "namespaces",
				APIGroup:        corev1.GroupName,
				APIVersion:      corev1.SchemeGroupVersion.Version,
				Name:            nsName,
				Namespace:       nsName,
				ResourceRequest: true,
			}

			// measure authorization
			experiment.MeasureDuration("authorization-allow", func() {
				d, _, err = authzr.Authorize(ctx, r)
			})

			// check results
			if err != nil {
				panic(err)
			}
			if d != authorizer.DecisionAllow {
				panic(fmt.Sprintf("expected decision Allow, got %d (0 Deny, 1 Allowed, 2 NoOpinion)", d))
			}
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			nsName := uns[0].GetName()
			// measure how long it takes to produce a NoOpinion decision to a request
			// and store the duration in a "authorization-no-opinion" measurement
			var d authorizer.Decision
			var err error
			r := authorizer.AttributesRecord{
				User:            &user.DefaultInfo{Name: username},
				Verb:            "get",
				Resource:        "namespaces",
				APIGroup:        corev1.GroupName,
				APIVersion:      corev1.SchemeGroupVersion.Version,
				Name:            nsName,
				Namespace:       nsName,
				ResourceRequest: true,
			}

			// measure authorization
			experiment.MeasureDuration("authorization-noopinion", func() {
				d, _, err = authzr.Authorize(ctx, r)
			})

			// check results
			if err != nil {
				panic(err)
			}
			if d != authorizer.DecisionNoOpinion {
				panic(fmt.Sprintf("expected decision NoOpinion, got %d (0 Deny, 1 Allowed, 2 NoOpinion)", d))
			}
		}, gmeasure.SamplingConfig{N: 30, Duration: 2 * time.Minute})
		// we'll sample the function up to 30 times or up to 2 minutes, whichever comes first.

		// we get the median listing duration from the experiment we just ran
		httpListingStats := experiment.GetStats("http listing")
		medianDuration := httpListingStats.DurationFor(gmeasure.StatMedian)

		// and assert that it hasn't changed much from ~100ms
		Expect(medianDuration).To(BeNumerically("~", 100*time.Millisecond, 70*time.Millisecond))
	})
})

func createResources(ctx context.Context, cli client.Client, user string, numAllowedNamespaces, numUnallowedNamespaces, numNonMatchingClusterRoles int) (error, []client.Object, []client.Object) {
	// cluster scoped resources
	mcr, nmcr := matchingClusterRoles(1), nonMatchingClusterRoles(numNonMatchingClusterRoles)
	ans, uns := namespaces("allowed-tenant-", numAllowedNamespaces), namespaces("unallowed-tenant-", numUnallowedNamespaces)
	if err := create(ctx, cli, slices.Concat(mcr, nmcr, ans, uns)); err != nil {
		return fmt.Errorf("could not create cluster scoped resources: %w", err), nil, nil
	}

	// namespace scoped resources
	atr := allowedTenants(user, ans, 10, "ClusterRole", mcr[0].GetName(), "ClusterRole", nmcr[0].GetName())
	utr := unallowedTenants(user, uns, 10, "ClusterRole", mcr[0].GetName(), "ClusterRole", nmcr[0].GetName())
	if err := create(ctx, cli, slices.Concat(atr, utr)); err != nil {
		return fmt.Errorf("could not create namespaced scoped resources: %w", err), nil, nil
	}
	return nil, ans, uns
}

func create(ctx context.Context, cli client.Client, rr []client.Object) error {
	for _, r := range rr {
		if err := cli.Create(ctx, r); client.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}
	return nil
}

func matchingClusterRoles(quantity int) []client.Object {
	crr := make([]client.Object, quantity)
	for i := range quantity {
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("perf-cluster-role-matching-%d", i),
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{corev1.GroupName},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get"},
				},
			},
		}
		crr[i] = cr
	}
	return crr
}

func nonMatchingClusterRoles(quantity int) []client.Object {
	crr := make([]client.Object, quantity)
	for i := range quantity {
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "perf-cluster-role-non-matching-",
			},
		}
		crr[i] = cr
	}
	return crr
}

func namespaces(generateName string, quantity int) []client.Object {
	rr := make([]client.Object, quantity)
	for i := range quantity {
		rr[i] = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
				Labels: map[string]string{
					NamespaceTypeLabelKey: NamespaceTypeUserLabelValue,
				},
			},
		}
	}
	return rr
}

func allowedTenants(user string, namespaces []client.Object, pollutingRoleBindings int, matchingRoleRefKind, matchingRoleRefName, nonMatchingRoleRefKind, nonMatchingRoleRefName string) []client.Object {
	rr := make([]client.Object, 0, len(namespaces)*(pollutingRoleBindings+1))
	for _, n := range namespaces {

		// add access role binding
		rr = append(rr, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "allowed-tenant-",
				Namespace:    n.GetName(),
			},
			Subjects: []rbacv1.Subject{{Kind: "User", APIGroup: rbacv1.GroupName, Name: user}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     matchingRoleRefKind,
				Name:     matchingRoleRefName,
			},
		})

		// add pollution
		for range pollutingRoleBindings {
			rr = append(rr, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "pollution-",
					Namespace:    n.GetName(),
				},
				Subjects: []rbacv1.Subject{{Kind: "User", APIGroup: rbacv1.GroupName, Name: user}},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     nonMatchingRoleRefKind,
					Name:     nonMatchingRoleRefName,
				},
			})
		}
	}
	return rr
}

func unallowedTenants(user string, namespaces []client.Object, pollutingRoleBindings int, matchingRoleRefKind, matchingRoleRefName, nonMatchingRoleRefKind, nonMatchingRoleRefName string) []client.Object {
	rr := make([]client.Object, 0, len(namespaces)*(pollutingRoleBindings+1))
	for _, n := range namespaces {
		// add access role binding
		rr = append(rr, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "non-allowed-tenant-",
				Namespace:    n.GetName(),
			},
			Subjects: []rbacv1.Subject{{Kind: "User", APIGroup: rbacv1.GroupName, Name: "not-the-perf-test-user"}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     matchingRoleRefKind,
				Name:     matchingRoleRefName,
			},
		})

		// add pollution
		for range pollutingRoleBindings {
			rr = append(rr, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "pollution-",
					Namespace:    n.GetName(),
				},
				Subjects: []rbacv1.Subject{{Kind: "User", APIGroup: rbacv1.GroupName, Name: user}},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     nonMatchingRoleRefKind,
					Name:     nonMatchingRoleRefName,
				},
			})
		}
	}
	return rr
}
